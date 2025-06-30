package git

import (
	"fmt"
	"os/exec"
	"strings"
)

type Repository struct {
	path string
}

func NewRepository(path string) (*Repository, error) {
	if !isGitRepository(path) {
		return nil, fmt.Errorf("not a git repository: %s", path)
	}
	return &Repository{path: path}, nil
}

func (r *Repository) Path() string {
	return r.path
}

func (r *Repository) GetWorktrees() ([]Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = r.path
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	return parseWorktreeList(string(output))
}

func (r *Repository) CreateWorktree(path, branch string) error {
	args := []string{"worktree", "add"}
	args = append(args, path)
	if branch != "" {
		args = append(args, branch)
	}

	cmd := exec.Command("git", args...)
	cmd.Dir = r.path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}
	return nil
}

func (r *Repository) RemoveWorktree(path string, force bool) error {
	args := []string{"worktree", "remove"}
	if force {
		args = append(args, "--force")
	}
	args = append(args, path)

	cmd := exec.Command("git", args...)
	cmd.Dir = r.path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}
	return nil
}

// ExecuteGitCommand executes a git command in the repository directory
func (r *Repository) ExecuteGitCommand(args ...string) error {
	cmd := exec.Command("git", args...)
	cmd.Dir = r.path
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("git command failed: %s", string(output))
	}
	return nil
}

func isGitRepository(path string) bool {
	// Use git rev-parse to check if we're in a git repository
	// This works for both regular repos and worktrees
	cmd := exec.Command("git", "rev-parse", "--git-dir")
	cmd.Dir = path
	if err := cmd.Run(); err != nil {
		return false
	}
	return true
}

func parseWorktreeList(output string) ([]Worktree, error) {
	var worktrees []Worktree
	lines := strings.Split(output, "\n")

	var current *Worktree
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			if current != nil {
				worktrees = append(worktrees, *current)
				current = nil
			}
			continue
		}

		if after, found := strings.CutPrefix(line, "worktree "); found {
			current = &Worktree{
				Path: after,
			}
		} else if current != nil {
			if after, found := strings.CutPrefix(line, "HEAD "); found {
				current.HEAD = after
			} else if after, found := strings.CutPrefix(line, "branch refs/heads/"); found {
				current.Branch = after
			}
		}
	}

	if current != nil {
		worktrees = append(worktrees, *current)
	}

	return worktrees, nil
}
