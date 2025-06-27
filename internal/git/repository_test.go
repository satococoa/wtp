package git

import (
	"os"
	"os/exec"
	"path/filepath"
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
