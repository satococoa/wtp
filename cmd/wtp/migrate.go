package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/satococoa/wtp/internal/config"
	"github.com/satococoa/wtp/internal/errors"
	"github.com/satococoa/wtp/internal/git"
)

// NewMigrateCommand creates the migrate-worktrees command definition
func NewMigrateCommand() *cli.Command {
	return &cli.Command{
		Name:  "migrate-worktrees",
		Usage: "Migrate legacy worktrees to namespaced layout",
		Description: "Migrates worktrees from legacy layout (../worktrees/feature/auth) to " +
			"namespaced layout (../worktrees/aragorn/feature/auth).\n\n" +
			"This command:\n" +
			"  1. Detects legacy worktrees in base_dir\n" +
			"  2. Moves them to base_dir/<repo-name>/\n" +
			"  3. Updates git's internal worktree paths\n" +
			"  4. Sets namespace_by_repo: true in .wtp.yml\n\n" +
			"If using a custom base_dir like ../aragorn-worktrees, you can consolidate\n" +
			"to the default ../worktrees using --new-base-dir since namespacing prevents\n" +
			"conflicts between projects.\n\n" +
			"Use --dry-run to preview changes without making them.",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "dry-run",
				Aliases: []string{"n"},
				Usage:   "Preview changes without making them",
			},
			&cli.StringFlag{
				Name:  "new-base-dir",
				Usage: "Move worktrees to a new base directory (e.g., ../worktrees)",
			},
		},
		Action: migrateWorktrees,
	}
}

func migrateWorktrees(_ context.Context, cmd *cli.Command) error {
	dryRun := cmd.Bool("dry-run")
	newBaseDir := cmd.String("new-base-dir")

	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return errors.DirectoryAccessFailed("access current", ".", err)
	}

	// Initialize repository to check if we're in a git repo
	repo, err := git.NewRepository(cwd)
	if err != nil {
		return errors.NotInGitRepository()
	}

	// Get main worktree path
	mainRepoPath, err := repo.GetMainWorktreePath()
	if err != nil {
		return errors.GitCommandFailed("get main worktree path", err.Error())
	}

	// Load config
	cfg, err := config.LoadConfig(mainRepoPath)
	if err != nil {
		return fmt.Errorf("failed to load config: %w", err)
	}

	// Get the writer from cli.Command
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	// Check if already using namespaced layout
	if cfg.ShouldNamespaceByRepo() && !cfg.UsesLegacyLayout() && newBaseDir == "" {
		fmt.Fprintln(w, "✅ Already using namespaced layout, nothing to migrate")
		return nil
	}

	// Determine target base directory
	targetBaseDir := cfg.Defaults.BaseDir
	if newBaseDir != "" {
		targetBaseDir = newBaseDir
	}

	// Detect if using repo-specific base_dir pattern and suggest consolidation
	repoName := filepath.Base(mainRepoPath)
	repoSpecificPattern := fmt.Sprintf("../%s-worktrees", repoName)
	if newBaseDir == "" && cfg.Defaults.BaseDir == repoSpecificPattern {
		fmt.Fprintf(w, "💡 Detected repo-specific base_dir: %s\n", cfg.Defaults.BaseDir)
		fmt.Fprintf(w, "   With namespacing, you can consolidate to: ../worktrees\n")
		fmt.Fprintf(w, "   Use --new-base-dir=../worktrees to migrate and update config\n\n")
	}

	// Find legacy worktrees
	legacyWorktrees, err := findLegacyWorktreesForMigration(mainRepoPath, cfg)
	if err != nil {
		return fmt.Errorf("failed to find legacy worktrees: %w", err)
	}

	if len(legacyWorktrees) == 0 && newBaseDir == "" {
		fmt.Fprintln(w, "✅ No legacy worktrees found to migrate")
		return nil
	}

	// Display migration plan
	if dryRun {
		fmt.Fprintf(w, "🔍 DRY RUN: Would migrate %d worktree(s):\n", len(legacyWorktrees))
		if newBaseDir != "" {
			fmt.Fprintf(w, "           Moving to new base_dir: %s\n", targetBaseDir)
		}
		fmt.Fprintln(w)
	} else {
		fmt.Fprintf(w, "📦 Migrating %d worktree(s) to namespaced layout...\n", len(legacyWorktrees))
		if newBaseDir != "" {
			fmt.Fprintf(w, "   Moving to new base_dir: %s\n", targetBaseDir)
		}
		fmt.Fprintln(w)
	}

	for _, wt := range legacyWorktrees {
		oldPath := wt.Path
		relativePath, err := filepath.Rel(filepath.Join(mainRepoPath, cfg.Defaults.BaseDir), oldPath)
		if err != nil {
			relativePath = filepath.Base(oldPath)
		}

		// Use targetBaseDir (which may be newBaseDir or current base_dir)
		newPath := filepath.Join(mainRepoPath, targetBaseDir, repoName, relativePath)

		fmt.Fprintf(w, "  %s\n", relativePath)
		fmt.Fprintf(w, "    From: %s\n", oldPath)
		fmt.Fprintf(w, "    To:   %s\n\n", newPath)

		if !dryRun {
			// Perform the migration
			if err := migrateWorktree(repo, oldPath, newPath); err != nil {
				return fmt.Errorf("failed to migrate worktree %s: %w", relativePath, err)
			}
		}
	}

	if dryRun {
		fmt.Fprintln(w, "💡 Run without --dry-run to perform migration")
		return nil
	}

	// Update config to set namespace_by_repo: true and new base_dir if specified
	namespaceTrue := true
	cfg.Defaults.NamespaceByRepo = &namespaceTrue
	if newBaseDir != "" {
		cfg.Defaults.BaseDir = targetBaseDir
	}
	if err := config.SaveConfig(mainRepoPath, cfg); err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	fmt.Fprintln(w, "✅ Migration complete!")
	fmt.Fprintf(w, "\nUpdated .wtp.yml with:\n")
	fmt.Fprintf(w, "  defaults:\n")
	if newBaseDir != "" {
		fmt.Fprintf(w, "    base_dir: %s\n", targetBaseDir)
	}
	fmt.Fprintf(w, "    namespace_by_repo: true\n")

	return nil
}

// findLegacyWorktreesForMigration finds all legacy worktrees that need migration
func findLegacyWorktreesForMigration(repoRoot string, cfg *config.Config) ([]git.Worktree, error) {
	baseDir := cfg.Defaults.BaseDir
	if !filepath.IsAbs(baseDir) {
		baseDir = filepath.Join(repoRoot, baseDir)
	}

	// Check if base directory exists
	if _, err := os.Stat(baseDir); os.IsNotExist(err) {
		return nil, nil
	}

	// Get all worktrees from git
	repo, err := git.NewRepository(repoRoot)
	if err != nil {
		return nil, err
	}

	allWorktrees, err := repo.GetWorktrees()
	if err != nil {
		return nil, err
	}

	// Find legacy worktrees (directly under baseDir, not in a repo subdirectory)
	var legacyWorktrees []git.Worktree
	repoName := filepath.Base(repoRoot)

	for _, wt := range allWorktrees {
		// Skip main worktree
		if wt.IsMain {
			continue
		}

		// Check if worktree is directly under baseDir (legacy layout)
		relativePath, err := filepath.Rel(baseDir, wt.Path)
		if err != nil || strings.HasPrefix(relativePath, "..") {
			// Not under baseDir
			continue
		}

		// If the first path component is the repo name, it's already namespaced
		pathComponents := strings.Split(relativePath, string(filepath.Separator))
		if len(pathComponents) > 0 && pathComponents[0] == repoName {
			// Already namespaced
			continue
		}

		// This is a legacy worktree
		legacyWorktrees = append(legacyWorktrees, wt)
	}

	return legacyWorktrees, nil
}

// migrateWorktree moves a worktree to a new location and updates git's internal paths
func migrateWorktree(repo *git.Repository, oldPath, newPath string) error {
	// Create parent directory for new path
	if err := os.MkdirAll(filepath.Dir(newPath), 0o755); err != nil {
		return fmt.Errorf("failed to create directory: %w", err)
	}

	// Use git worktree move to relocate the worktree
	// This handles updating all internal git paths
	if err := repo.ExecuteGitCommand("worktree", "move", oldPath, newPath); err != nil {
		return fmt.Errorf("git worktree move failed: %w", err)
	}

	return nil
}
