package git

import (
	stdErrors "errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/satococoa/wtp/v2/internal/errors"
)

type Repository struct {
	path string
}

func NewRepository(path string) (*Repository, error) {
	if !isGitRepository(path) {
		return nil, errors.NotInGitRepository()
	}
	return &Repository{path: path}, nil
}

func (r *Repository) Path() string {
	return r.path
}

// GetRepositoryName returns the name of the repository
func (r *Repository) GetRepositoryName() string {
	return filepath.Base(r.path)
}

// GetMainWorktreePath returns the path to the main worktree (original repository)
// This is useful when running commands from within a worktree
func (r *Repository) GetMainWorktreePath() (string, error) {
	// Get the common directory which points to the main repository's .git
	cmd := exec.Command("git", "rev-parse", "--git-common-dir")
	cmd.Dir = r.path
	output, err := cmd.Output()
	if err != nil {
		return "", fmt.Errorf("failed to get main repository path: %w", err)
	}

	commonDir := strings.TrimSpace(string(output))

	// If commonDir ends with .git, get its parent directory
	if strings.HasSuffix(commonDir, ".git") {
		// Get the parent directory of .git
		parent := filepath.Dir(commonDir)
		// Convert to absolute path if needed
		if !filepath.IsAbs(parent) {
			absPath, absErr := filepath.Abs(filepath.Join(r.path, parent))
			if absErr != nil {
				return "", fmt.Errorf("failed to get absolute path: %w", absErr)
			}
			return absPath, nil
		}
		return parent, nil
	}

	// If commonDir doesn't end with .git, it's likely already the worktree path
	if !filepath.IsAbs(commonDir) {
		absPath, absErr := filepath.Abs(filepath.Join(r.path, commonDir))
		if absErr != nil {
			return "", fmt.Errorf("failed to get absolute path: %w", absErr)
		}
		return absPath, nil
	}

	return commonDir, nil
}

func (r *Repository) GetWorktrees() ([]Worktree, error) {
	cmd := exec.Command("git", "worktree", "list", "--porcelain")
	cmd.Dir = r.path
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to list worktrees: %w", err)
	}

	worktrees := parseWorktreeList(string(output))

	// The first worktree in the list is always the main worktree
	if len(worktrees) > 0 {
		worktrees[0].IsMain = true
	}

	return worktrees, nil
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
	// Debug: print the command being executed
	// fmt.Printf("DEBUG: Executing: git %s\n", strings.Join(args, " "))
	output, err := cmd.CombinedOutput()
	if err != nil {
		return errors.GitCommandFailed(fmt.Sprintf("git %s", strings.Join(args, " ")), string(output))
	}
	return nil
}

// BranchExists checks if a branch exists locally
func (r *Repository) BranchExists(branch string) (bool, error) {
	// Validate branch name to prevent command injection
	if strings.Contains(branch, "..") || strings.ContainsAny(branch, "\n\r") {
		return false, errors.InvalidBranchName(branch)
	}

	// #nosec G204 - branch is validated above
	cmd := exec.Command("git", "show-ref", "--verify", "--quiet", fmt.Sprintf("refs/heads/%s", branch))
	cmd.Dir = r.path
	err := cmd.Run()
	if err != nil {
		var exitErr *exec.ExitError
		if stdErrors.As(err, &exitErr) && exitErr.ExitCode() == 1 {
			return false, nil
		}
		return false, fmt.Errorf("failed to check branch existence: %w", err)
	}
	return true, nil
}

// GetRemoteBranches returns a map of remote branches by remote name
func (r *Repository) GetRemoteBranches(branch string) (map[string]string, error) {
	// Validate branch name to prevent command injection
	if strings.Contains(branch, "..") || strings.ContainsAny(branch, "\n\r") {
		return nil, errors.InvalidBranchName(branch)
	}

	remotes := make(map[string]string)

	// Get all remote branches that match the branch name
	// #nosec G204 - branch is validated above
	cmd := exec.Command("git", "for-each-ref", "--format=%(refname:short)", fmt.Sprintf("refs/remotes/*/%s", branch))
	cmd.Dir = r.path
	output, err := cmd.Output()
	if err != nil {
		return nil, fmt.Errorf("failed to get remote branches: %w", err)
	}

	lines := strings.Split(strings.TrimSpace(string(output)), "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}

		// Parse remote/branch format
		const remotePartCount = 2
		parts := strings.SplitN(line, "/", remotePartCount)
		if len(parts) == remotePartCount {
			remote := parts[0]
			remotes[remote] = line
		}
	}

	return remotes, nil
}

// ResolveBranch resolves a branch name following git's behavior:
// 1. Check if branch exists locally
// 2. If not, check remote branches
// 3. If multiple remotes have the branch, return an error
func (r *Repository) ResolveBranch(branch string) (resolvedBranch string, isRemote bool, err error) {
	// First check if branch exists locally
	exists, err := r.BranchExists(branch)
	if err != nil {
		return "", false, err
	}
	if exists {
		return branch, false, nil
	}

	// Check remote branches
	remoteBranches, err := r.GetRemoteBranches(branch)
	if err != nil {
		return "", false, err
	}

	if len(remoteBranches) == 0 {
		return "", false, errors.BranchNotFound(branch)
	}

	if len(remoteBranches) > 1 {
		// Multiple remotes have this branch
		remoteNames := make([]string, 0, len(remoteBranches))
		for remote := range remoteBranches {
			remoteNames = append(remoteNames, remote)
		}
		return "", false, errors.MultipleBranchesFound(branch, remoteNames)
	}

	// Single remote has this branch
	for _, remoteBranch := range remoteBranches {
		return remoteBranch, true, nil
	}

	return "", false, nil
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

func parseWorktreeList(output string) []Worktree {
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

	return worktrees
}
