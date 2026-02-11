package main

import (
	"path/filepath"
	"strings"

	"github.com/satococoa/wtp/v2/internal/config"
	"github.com/satococoa/wtp/v2/internal/git"
)

func findMainWorktreePath(worktrees []git.Worktree) string {
	// The first worktree is always the main worktree (git worktree list behavior)
	if len(worktrees) > 0 {
		return worktrees[0].Path
	}
	return ""
}

func resolveWorktreePathByName(worktreeName string, worktrees []git.Worktree, mainWorktreePath string) string {
	// Remove asterisk marker from completion (e.g., "feature*" -> "feature", "@*" -> "@")
	worktreeName = strings.TrimSuffix(worktreeName, "*")

	// Load config for unified naming
	cfg, err := config.LoadConfig(mainWorktreePath)
	if err != nil {
		cfg = &config.Config{
			Defaults: config.Defaults{
				BaseDir: config.DefaultBaseDir,
			},
		}
	}

	// The order matters: more specific matches come first
	for i := range worktrees {
		wt := &worktrees[i]

		if path := tryDirectWorktreeMatches(wt, worktreeName, cfg, mainWorktreePath); path != "" {
			return path
		}

		if path := tryMainWorktreeMatches(wt, worktreeName, mainWorktreePath); path != "" {
			return path
		}
	}

	return ""
}

func availableManagedWorktreeNames(worktrees []git.Worktree, mainRepoPath string) []string {
	cfg, err := config.LoadConfig(mainRepoPath)
	if err != nil {
		cfg = &config.Config{
			Defaults: config.Defaults{
				BaseDir: config.DefaultBaseDir,
			},
		}
	}

	names := make([]string, 0, len(worktrees))
	for i := range worktrees {
		wt := &worktrees[i]
		if !isWorktreeManagedCommon(wt.Path, cfg, mainRepoPath, wt.IsMain) {
			continue
		}
		names = append(names, getWorktreeNameFromPath(wt.Path, cfg, mainRepoPath, wt.IsMain))
	}

	return names
}

func tryDirectWorktreeMatches(
	wt *git.Worktree, worktreeName string, cfg *config.Config, mainWorktreePath string,
) string {
	// Skip unmanaged worktrees - they cannot be navigated to by wtp
	if !isWorktreeManagedCommon(wt.Path, cfg, mainWorktreePath, wt.IsMain) {
		return ""
	}

	// Priority 1: Exact branch name match (supports prefixes like "feature/awesome")
	if wt.Branch == worktreeName {
		return wt.Path
	}

	// Priority 2: Unified worktree name match (relative to base_dir)
	if cfg != nil {
		worktreeNameFromPath := getWorktreeNameFromPath(wt.Path, cfg, mainWorktreePath, wt.IsMain)
		if worktreeNameFromPath == worktreeName {
			return wt.Path
		}
	}

	// Priority 3: Worktree directory name match (legacy behavior)
	if filepath.Base(wt.Path) == worktreeName {
		return wt.Path
	}

	return ""
}

func tryMainWorktreeMatches(wt *git.Worktree, worktreeName, mainWorktreePath string) string {
	if !wt.IsMainWorktree(mainWorktreePath) {
		return ""
	}

	// Priority 4: Root worktree alias ("root" -> main worktree)
	if worktreeName == "root" {
		return wt.Path
	}

	// Priority 5: @ symbol for main worktree ("@" -> main worktree)
	if worktreeName == "@" {
		return wt.Path
	}

	// Priority 6: Repository name for root worktree ("giselle" -> root worktree)
	if worktreeName == filepath.Base(wt.Path) {
		return wt.Path
	}

	// Priority 7: Legacy completion display format ("wtp(root worktree)" -> root worktree)
	repoRootFormat := filepath.Base(wt.Path) + "(root worktree)"
	if worktreeName == repoRootFormat {
		return wt.Path
	}

	// Priority 8: Current completion display format ("giselle@fix-nodes(root worktree)" -> root worktree)
	if strings.HasSuffix(worktreeName, "(root worktree)") {
		// Extract repo name and branch from format "repo@branch(root worktree)"
		prefix := strings.TrimSuffix(worktreeName, "(root worktree)")
		// Check if this matches the worktree by comparing repo name and branch
		expectedPrefix := filepath.Base(wt.Path) + "@" + wt.Branch
		if prefix == expectedPrefix {
			return wt.Path
		}
	}

	return ""
}
