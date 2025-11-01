package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/satococoa/wtp/internal/config"
	"github.com/satococoa/wtp/internal/git"
)

const legacyWarningExampleLimit = 3

type legacyWorktreeMigration struct {
	currentRel   string
	suggestedRel string
}

func maybeWarnLegacyWorktreeLayout(
	w io.Writer,
	mainRepoPath string,
	cfg *config.Config,
	worktrees []git.Worktree,
) {
	if w == nil {
		w = os.Stderr
	}

	if cfg == nil || mainRepoPath == "" || len(worktrees) == 0 {
		return
	}

	if hasConfigFile(mainRepoPath) {
		return
	}

	migrations := detectLegacyWorktreeMigrations(mainRepoPath, cfg, worktrees)
	if len(migrations) == 0 {
		return
	}

	repoBase := filepath.Base(mainRepoPath)
	fmt.Fprintln(w, "⚠️  Legacy worktree layout detected.")
	fmt.Fprintf(w, "    wtp now expects worktrees under '../worktrees/%s/...'\n", repoBase)
	fmt.Fprintln(w, "    Move existing worktrees to the new layout (run from the repository root):")

	limit := len(migrations)
	if limit > legacyWarningExampleLimit {
		limit = legacyWarningExampleLimit
	}
	for i := 0; i < limit; i++ {
		migration := migrations[i]
		fmt.Fprintf(w, "      git worktree move %s %s\n", migration.currentRel, migration.suggestedRel)
	}

	if len(migrations) > legacyWarningExampleLimit {
		fmt.Fprintf(w, "      ... and %d more\n", len(migrations)-legacyWarningExampleLimit)
	}

	fmt.Fprintln(w, "    (Alternatively, run 'wtp init' and set defaults.base_dir to keep a custom layout.)")
	fmt.Fprintln(w)
}

func detectLegacyWorktreeMigrations(
	mainRepoPath string,
	cfg *config.Config,
	worktrees []git.Worktree,
) []legacyWorktreeMigration {
	if cfg == nil || mainRepoPath == "" {
		return nil
	}

	mainRepoPath = filepath.Clean(mainRepoPath)

	newBaseDir := filepath.Clean(cfg.ResolveWorktreePath(mainRepoPath, ""))
	legacyBaseDir := filepath.Clean(filepath.Join(filepath.Dir(mainRepoPath), "worktrees"))
	repoBase := filepath.Base(mainRepoPath)

	if legacyBaseDir == newBaseDir {
		return nil
	}

	var migrations []legacyWorktreeMigration
	for _, wt := range worktrees {
		if wt.IsMain {
			continue
		}

		worktreePath := filepath.Clean(wt.Path)

		if strings.HasPrefix(worktreePath, newBaseDir+string(os.PathSeparator)) ||
			worktreePath == newBaseDir {
			continue
		}

		if !strings.HasPrefix(worktreePath, legacyBaseDir+string(os.PathSeparator)) {
			continue
		}

		legacyRel, err := filepath.Rel(legacyBaseDir, worktreePath)
		if err != nil || legacyRel == "." {
			continue
		}

		if strings.HasPrefix(legacyRel, repoBase+string(os.PathSeparator)) {
			// Already under the new structure (worktrees/<repo>/...)
			continue
		}

		suggestedPath := filepath.Join(legacyBaseDir, repoBase, legacyRel)

		currentRel := relativeToRepo(mainRepoPath, worktreePath)
		suggestedRel := relativeToRepo(mainRepoPath, suggestedPath)

		migrations = append(migrations, legacyWorktreeMigration{
			currentRel:   currentRel,
			suggestedRel: suggestedRel,
		})
	}

	return migrations
}

func relativeToRepo(mainRepoPath, targetPath string) string {
	rel, err := filepath.Rel(mainRepoPath, targetPath)
	if err != nil {
		return targetPath
	}
	if !strings.HasPrefix(rel, "..") {
		rel = filepath.Join(".", rel)
	}
	return filepath.Clean(rel)
}

func hasConfigFile(mainRepoPath string) bool {
	configPath := filepath.Join(mainRepoPath, config.ConfigFileName)
	_, err := os.Stat(configPath)
	return err == nil
}
