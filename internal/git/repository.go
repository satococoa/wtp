package git

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
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

func (r *Repository) GetBranches() ([]string, error) {
	cmd := exec.Command("git", "branch", "-a")
	cmd.Dir = r.path
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get branches: %w", err)
	}

	var branches []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if after, found := strings.CutPrefix(line, "* "); found {
			branches = append(branches, after)
		} else {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

// GetLocalBranches returns only local branches
func (r *Repository) GetLocalBranches() ([]string, error) {
	cmd := exec.Command("git", "branch")
	cmd.Dir = r.path
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get local branches: %w", err)
	}

	var branches []string
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if after, found := strings.CutPrefix(line, "* "); found {
			branches = append(branches, after)
		} else {
			branches = append(branches, line)
		}
	}
	return branches, nil
}

// GetRemoteBranches returns remote branches grouped by remote
func (r *Repository) GetRemoteBranches() (map[string][]string, error) {
	cmd := exec.Command("git", "branch", "-r")
	cmd.Dir = r.path
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get remote branches: %w", err)
	}

	remoteBranches := make(map[string][]string)
	for _, line := range strings.Split(string(output), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.Contains(line, "->") {
			continue
		}
		
		parts := strings.SplitN(line, "/", 2)
		if len(parts) == 2 {
			remote := parts[0]
			branch := parts[1]
			remoteBranches[remote] = append(remoteBranches[remote], branch)
		}
	}
	return remoteBranches, nil
}

// ResolveBranch resolves a branch name following Git's behavior:
// 1. Check local branches first
// 2. If not found, search remote branches
// 3. If multiple remotes have the same branch, return error
func (r *Repository) ResolveBranch(branchName string) (string, error) {
	// First check local branches
	localBranches, err := r.GetLocalBranches()
	if err != nil {
		return "", fmt.Errorf("failed to get local branches: %w", err)
	}
	
	if slices.Contains(localBranches, branchName) {
		return branchName, nil
	}

	// Check remote branches
	remoteBranches, err := r.GetRemoteBranches()
	if err != nil {
		return "", fmt.Errorf("failed to get remote branches: %w", err)
	}

	var matchingRemotes []string
	for remote, branches := range remoteBranches {
		for _, branch := range branches {
			if branch == branchName {
				matchingRemotes = append(matchingRemotes, remote)
				break
			}
		}
	}

	if len(matchingRemotes) == 0 {
		return "", fmt.Errorf("branch '%s' not found in local or remote branches", branchName)
	}

	if len(matchingRemotes) > 1 {
		return "", fmt.Errorf("branch '%s' exists in multiple remotes: %s. Please specify remote explicitly", 
			branchName, strings.Join(matchingRemotes, ", "))
	}

	return fmt.Sprintf("%s/%s", matchingRemotes[0], branchName), nil
}

// CreateBranchFromRemote creates a local branch tracking a remote branch
func (r *Repository) CreateBranchFromRemote(localBranch, remoteBranch string) error {
	cmd := exec.Command("git", "checkout", "-b", localBranch, remoteBranch)
	cmd.Dir = r.path
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to create branch from remote: %w", err)
	}
	return nil
}

// CreateWorktreeFromBranch creates a worktree from a branch, handling local/remote automatically
func (r *Repository) CreateWorktreeFromBranch(path, branchName string) error {
	// If branchName is empty, just create worktree at current HEAD
	if branchName == "" {
		return r.CreateWorktree(path, "")
	}

	// Resolve the branch (local or remote)
	resolvedBranch, err := r.ResolveBranch(branchName)
	if err != nil {
		return fmt.Errorf("failed to resolve branch: %w", err)
	}

	// If it's a remote branch, we need to create a local tracking branch first
	if resolvedBranch != branchName && strings.Contains(resolvedBranch, "/") {
		// This is a remote branch, create local tracking branch
		if err := r.CreateBranchFromRemote(branchName, resolvedBranch); err != nil {
			return fmt.Errorf("failed to create tracking branch: %w", err)
		}
		// Now create worktree with the local branch
		return r.CreateWorktree(path, branchName)
	}

	// It's a local branch, create worktree directly
	return r.CreateWorktree(path, resolvedBranch)
}

func isGitRepository(path string) bool {
	gitDir := filepath.Join(path, ".git")
	if stat, err := os.Stat(gitDir); err == nil {
		return stat.IsDir()
	}
	return false
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