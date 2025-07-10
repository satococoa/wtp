package git

import (
	"fmt"
	"path/filepath"
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

// CompletionName returns the name to display for shell completion
// For the main worktree, it returns the repository name with "(root worktree)" suffix
// For other worktrees, it returns the branch name preserving any prefixes
func (w *Worktree) CompletionName(repoName string) string {
	if w.IsMainWorktree("") {
		// For main worktree, show repo name with indicator
		return fmt.Sprintf("%s(root worktree)", repoName)
	}

	// For other worktrees, prefer branch name over directory name
	if w.Branch != "" {
		return w.Branch
	}

	// Fallback to directory name if no branch
	return filepath.Base(w.Path)
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
