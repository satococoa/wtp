package git

import (
	"fmt"
	"path/filepath"
	"strings"
)

const (
	MainBranch   = "main"
	MasterBranch = "master"
)

type Worktree struct {
	Path   string
	Branch string
	HEAD   string
}

func (w *Worktree) Name() string {
	return filepath.Base(w.Path)
}

func (w *Worktree) String() string {
	if w.Branch != "" {
		return fmt.Sprintf("%s [%s]", w.Path, w.Branch)
	}
	return fmt.Sprintf("%s [%s]", w.Path, w.HEAD)
}

// CompletionName returns the name to display for shell completion.
//
// Format: <worktreeName>@<commit-ish>[(root worktree)]
//
// Examples:
//   - Root worktree: "giselle@fix-nodes(root worktree)"
//   - Matching names: "develop" (when worktree dir and branch both = "develop")
//   - Different names: "feature-awesome@feature/awesome"
//   - Full path match: "feature/new-top-page" (when path ends with branch name)
func (w *Worktree) CompletionName(repoName string) string {
	// Check if this is the main/root worktree
	if w.isMainWorktreeHeuristic() {
		return fmt.Sprintf("%s@%s(root worktree)", repoName, w.Branch)
	}

	// For other worktrees, determine optimal display format
	return w.formatNonRootWorktreeCompletion()
}

// isMainWorktreeHeuristic uses path heuristics to determine if this is the main worktree.
// Main worktrees typically don't have "/worktrees/" in their path.
func (w *Worktree) isMainWorktreeHeuristic() bool {
	return !strings.Contains(w.Path, filepath.Join("", "worktrees", ""))
}

// formatNonRootWorktreeCompletion formats completion name for non-root worktrees.
// Priority:
// 1. If path ends with branch name → show branch only
// 2. If directory name = branch name → show branch only
// 3. Otherwise → show "directory@branch"
func (w *Worktree) formatNonRootWorktreeCompletion() string {
	if w.Branch == "" {
		return filepath.Base(w.Path)
	}

	// Check if path ends with full branch name (handles prefixed paths)
	if strings.HasSuffix(w.Path, w.Branch) {
		return w.Branch
	}

	// Check if directory name matches branch name
	worktreeName := filepath.Base(w.Path)
	if w.Branch == worktreeName {
		return w.Branch
	}

	// Different names: show both
	return fmt.Sprintf("%s@%s", worktreeName, w.Branch)
}

// IsMainWorktree returns true if this is the main/root worktree
func (w *Worktree) IsMainWorktree(mainWorktreePath string) bool {
	// If mainWorktreePath is provided, compare paths
	if mainWorktreePath != "" {
		return w.Path == mainWorktreePath
	}

	// Heuristic: main worktree usually has "main" or "master" branch
	// and the path doesn't contain "worktrees" directory
	return w.Branch == MainBranch || w.Branch == MasterBranch
}
