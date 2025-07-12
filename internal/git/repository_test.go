package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestRepo(t *testing.T) string {
	tempDir := t.TempDir()

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Set default branch to main (works with older git versions too)
	cmd = exec.Command("git", "config", "init.defaultBranch", "main")
	cmd.Dir = tempDir
	_ = cmd.Run() // Ignore error if git version is too old

	// Configure git user
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git email: %v", err)
	}

	// Create initial commit
	readmeFile := filepath.Join(tempDir, "README.md")
	if err := os.WriteFile(readmeFile, []byte("# Test Repo"), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add README: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = tempDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Ensure the default branch is named 'main'
	// Check current branch name
	cmd = exec.Command("git", "branch", "--show-current")
	cmd.Dir = tempDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	currentBranch := strings.TrimSpace(string(output))
	if currentBranch == "master" {
		// Rename master to main
		cmd = exec.Command("git", "branch", "-m", "master", "main")
		cmd.Dir = tempDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to rename branch to main: %v", err)
		}
	}

	return tempDir
}

func TestNewRepository(t *testing.T) {
	// Test with valid git repository
	repoDir := setupTestRepo(t)

	repo, err := NewRepository(repoDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if repo.Path() != repoDir {
		t.Errorf("Expected path %s, got %s", repoDir, repo.Path())
	}

	// Test with non-git directory
	tempDir := t.TempDir()
	_, err = NewRepository(tempDir)
	if err == nil {
		t.Error("Expected error for non-git directory, got nil")
	}
}

func TestParseWorktreeList(t *testing.T) {
	tests := []struct {
		name     string
		output   string
		expected []Worktree
	}{
		{
			name: "single worktree",
			output: `worktree /path/to/main
HEAD abcd1234

`,
			expected: []Worktree{
				{
					Path: "/path/to/main",
					HEAD: "abcd1234",
				},
			},
		},
		{
			name: "multiple worktrees",
			output: `worktree /path/to/main
HEAD abcd1234
branch refs/heads/main

worktree /path/to/feature
HEAD efgh5678
branch refs/heads/feature/test

`,
			expected: []Worktree{
				{
					Path:   "/path/to/main",
					HEAD:   "abcd1234",
					Branch: "main",
				},
				{
					Path:   "/path/to/feature",
					HEAD:   "efgh5678",
					Branch: "feature/test",
				},
			},
		},
		{
			name:     "empty output",
			output:   "",
			expected: []Worktree{},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := parseWorktreeList(tt.output)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}

			if len(result) != len(tt.expected) {
				t.Errorf("Expected %d worktrees, got %d", len(tt.expected), len(result))
				return
			}

			for i, expected := range tt.expected {
				if result[i].Path != expected.Path {
					t.Errorf("Worktree %d: expected path %s, got %s", i, expected.Path, result[i].Path)
				}
				if result[i].HEAD != expected.HEAD {
					t.Errorf("Worktree %d: expected HEAD %s, got %s", i, expected.HEAD, result[i].HEAD)
				}
				if result[i].Branch != expected.Branch {
					t.Errorf("Worktree %d: expected branch %s, got %s", i, expected.Branch, result[i].Branch)
				}
			}
		})
	}
}

func TestExecuteGitCommand(t *testing.T) {
	repoDir := setupTestRepo(t)
	repo, err := NewRepository(repoDir)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Test successful command
	err = repo.ExecuteGitCommand("status")
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	// Test command with arguments
	err = repo.ExecuteGitCommand("log", "--oneline", "-1")
	if err != nil {
		t.Errorf("Expected no error but got: %v", err)
	}

	// Test failing command
	err = repo.ExecuteGitCommand("invalid-command")
	if err == nil {
		t.Error("Expected error for invalid command")
	}
}

func TestRepository_GetRepositoryName(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		expected string
	}{
		{
			name:     "simple repository name",
			path:     "/Users/user/repos/wtp",
			expected: "wtp",
		},
		{
			name:     "repository with long path",
			path:     "/home/developer/projects/my-awesome-project",
			expected: "my-awesome-project",
		},
		{
			name:     "repository in nested directory",
			path:     "/var/lib/git/repositories/backend-api",
			expected: "backend-api",
		},
		{
			name:     "root directory",
			path:     "/",
			expected: "/",
		},
		{
			name:     "current directory",
			path:     ".",
			expected: ".",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			repo := &Repository{path: tt.path}
			result := repo.GetRepositoryName()
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestIsGitRepository(t *testing.T) {
	// Test with valid git repository
	repoDir := setupTestRepo(t)
	if !isGitRepository(repoDir) {
		t.Error("Expected true for git repository")
	}

	// Test with non-git directory
	tempDir := t.TempDir()
	if isGitRepository(tempDir) {
		t.Error("Expected false for non-git directory")
	}

	// Test with non-existent directory
	if isGitRepository("/path/that/does/not/exist") {
		t.Error("Expected false for non-existent directory")
	}
}

func TestBranchResolution(t *testing.T) {
	// Create a temporary directory for test repository
	repoDir := setupTestRepo(t)

	// Create local branch
	runCmd(t, repoDir, "git", "branch", "local-feature")

	// Add remotes and their branches
	// Remote "origin"
	runCmd(t, repoDir, "git", "remote", "add", "origin", "https://example.com/repo.git")

	// Create fake remote refs
	originRefsDir := filepath.Join(repoDir, ".git", "refs", "remotes", "origin")
	if err := os.MkdirAll(originRefsDir, 0755); err != nil {
		t.Fatalf("Failed to create origin refs dir: %v", err)
	}

	// Get current HEAD commit
	headCommit := getHeadCommit(t, repoDir)

	// Create remote branches
	if err := os.WriteFile(filepath.Join(originRefsDir, "remote-only"), []byte(headCommit), 0644); err != nil {
		t.Fatalf("Failed to create origin/remote-only: %v", err)
	}
	if err := os.WriteFile(filepath.Join(originRefsDir, "shared-branch"), []byte(headCommit), 0644); err != nil {
		t.Fatalf("Failed to create origin/shared-branch: %v", err)
	}

	// Remote "upstream"
	runCmd(t, repoDir, "git", "remote", "add", "upstream", "https://example.com/upstream.git")
	upstreamRefsDir := filepath.Join(repoDir, ".git", "refs", "remotes", "upstream")
	if err := os.MkdirAll(upstreamRefsDir, 0755); err != nil {
		t.Fatalf("Failed to create upstream refs dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(upstreamRefsDir, "shared-branch"), []byte(headCommit), 0644); err != nil {
		t.Fatalf("Failed to create upstream/shared-branch: %v", err)
	}

	// Create repository instance
	repo, err := NewRepository(repoDir)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Test cases
	tests := []struct {
		name          string
		branch        string
		expectError   bool
		expectRemote  bool
		expectBranch  string
		errorContains string
	}{
		{
			name:         "Local branch exists",
			branch:       "local-feature",
			expectError:  false,
			expectRemote: false,
			expectBranch: "local-feature",
		},
		{
			name:         "Remote branch exists in single remote",
			branch:       "remote-only",
			expectError:  false,
			expectRemote: true,
			expectBranch: "origin/remote-only",
		},
		{
			name:          "Branch exists in multiple remotes",
			branch:        "shared-branch",
			expectError:   true,
			errorContains: "exists in multiple remotes",
		},
		{
			name:          "Branch does not exist",
			branch:        "nonexistent",
			expectError:   true,
			errorContains: "not found in local or remote branches",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			resolvedBranch, isRemote, err := repo.ResolveBranch(tt.branch)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error but got none")
				} else if tt.errorContains != "" && !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("Expected error containing '%s', got: %v", tt.errorContains, err)
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if isRemote != tt.expectRemote {
					t.Errorf("Expected isRemote=%v, got %v", tt.expectRemote, isRemote)
				}
				if resolvedBranch != tt.expectBranch {
					t.Errorf("Expected branch '%s', got '%s'", tt.expectBranch, resolvedBranch)
				}
			}
		})
	}
}

func runCmd(t *testing.T, dir, _ string, args ...string) {
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if output, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("command failed: %s\nOutput: %s", err, output)
	}
}

func getHeadCommit(t *testing.T, dir string) string {
	cmd := exec.Command("git", "rev-parse", "HEAD")
	cmd.Dir = dir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get HEAD commit: %v", err)
	}
	return strings.TrimSpace(string(output))
}
