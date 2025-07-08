package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runGitCommand is a helper to run git commands in tests
func runGitCommand(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to run git %v: %v", args, err)
	}
}

// checkoutMainBranch checks out the main branch
func checkoutMainBranch(t *testing.T, repoDir string) {
	t.Helper()
	cmd := exec.Command("git", "checkout", "main")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to checkout main: %v", err)
	}
}

// initializeTestRepo sets up a git repository with initial commit
func initializeTestRepo(t *testing.T, repoDir string) {
	t.Helper()
	runGitCommand(t, repoDir, "init")
	runGitCommand(t, repoDir, "config", "user.name", "Test User")
	runGitCommand(t, repoDir, "config", "user.email", "test@example.com")

	// Create initial commit
	readmeFile := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readmeFile, []byte("# Test Repo"), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}
	runGitCommand(t, repoDir, "add", "README.md")
	runGitCommand(t, repoDir, "commit", "-m", "Initial commit")

	// Ensure the default branch is named 'main'
	cmd := exec.Command("git", "branch", "--show-current")
	cmd.Dir = repoDir
	output, err := cmd.Output()
	if err != nil {
		t.Fatalf("Failed to get current branch: %v", err)
	}

	currentBranch := strings.TrimSpace(string(output))
	if currentBranch == "master" {
		// Rename master to main
		runGitCommand(t, repoDir, "branch", "-m", "master", "main")
	}
}

// createMergedBranch creates and merges a branch
func createMergedBranch(t *testing.T, repoDir, branchName string) {
	t.Helper()
	runGitCommand(t, repoDir, "checkout", "-b", branchName)

	// Add a commit on merged branch
	mergedFile := filepath.Join(repoDir, "merged.txt")
	if err := os.WriteFile(mergedFile, []byte("merged content"), 0644); err != nil {
		t.Fatalf("Failed to write merged file: %v", err)
	}
	runGitCommand(t, repoDir, "add", "merged.txt")
	runGitCommand(t, repoDir, "commit", "-m", "Add merged file")

	// Switch back to main/master and merge
	checkoutMainBranch(t, repoDir)
	runGitCommand(t, repoDir, "merge", branchName)
}

// createUnmergedBranch creates a branch with commits that are not merged
func createUnmergedBranch(t *testing.T, repoDir, branchName string) {
	t.Helper()
	runGitCommand(t, repoDir, "checkout", "-b", branchName)

	// Add a commit on unmerged branch
	unmergedFile := filepath.Join(repoDir, "unmerged.txt")
	if err := os.WriteFile(unmergedFile, []byte("unmerged content"), 0644); err != nil {
		t.Fatalf("Failed to write unmerged file: %v", err)
	}
	runGitCommand(t, repoDir, "add", "unmerged.txt")
	runGitCommand(t, repoDir, "commit", "-m", "Add unmerged file")

	// Switch back to main/master
	checkoutMainBranch(t, repoDir)
}

func setupTestRepoWithBranches(t *testing.T) (repoDir, mergedBranch, unmergedBranch string) {
	repoDir = t.TempDir()
	mergedBranch = "merged-branch"
	unmergedBranch = "unmerged-branch"

	initializeTestRepo(t, repoDir)
	createMergedBranch(t, repoDir, mergedBranch)
	createUnmergedBranch(t, repoDir, unmergedBranch)

	return repoDir, mergedBranch, unmergedBranch
}

func TestBranchDeletion(t *testing.T) {
	repoDir, mergedBranch, unmergedBranch := setupTestRepoWithBranches(t)

	repo, err := NewRepository(repoDir)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Test deleting merged branch with -d
	err = repo.ExecuteGitCommand("branch", "-d", mergedBranch)
	if err != nil {
		t.Errorf("Failed to delete merged branch: %v", err)
	}

	// Test deleting unmerged branch with -d (should fail)
	err = repo.ExecuteGitCommand("branch", "-d", unmergedBranch)
	if err == nil {
		t.Error("Expected error when deleting unmerged branch with -d")
	}
	if err != nil && !strings.Contains(err.Error(), "not fully merged") {
		t.Errorf("Expected 'not fully merged' error, got: %v", err)
	}

	// Test deleting unmerged branch with -D (should succeed)
	err = repo.ExecuteGitCommand("branch", "-D", unmergedBranch)
	if err != nil {
		t.Errorf("Failed to force delete unmerged branch: %v", err)
	}

	// Verify branches are deleted
	cmd := exec.Command("git", "branch")
	cmd.Dir = repoDir
	output, _ := cmd.Output()
	branches := string(output)

	if strings.Contains(branches, mergedBranch) {
		t.Error("Merged branch should have been deleted")
	}
	if strings.Contains(branches, unmergedBranch) {
		t.Error("Unmerged branch should have been deleted")
	}
}

func TestWorktreeWithBranchRemoval(t *testing.T) {
	repoDir, _, unmergedBranch := setupTestRepoWithBranches(t)

	repo, err := NewRepository(repoDir)
	if err != nil {
		t.Fatalf("Failed to create repository: %v", err)
	}

	// Create worktree for unmerged branch
	worktreePath := filepath.Join(repoDir, "..", "worktrees", unmergedBranch)
	err = repo.CreateWorktree(worktreePath, unmergedBranch)
	if err != nil {
		t.Fatalf("Failed to create worktree: %v", err)
	}

	// Remove worktree
	err = repo.RemoveWorktree(worktreePath, false)
	if err != nil {
		t.Errorf("Failed to remove worktree: %v", err)
	}

	// Try to delete the branch (should fail because it's unmerged)
	err = repo.ExecuteGitCommand("branch", "-d", unmergedBranch)
	if err == nil {
		t.Error("Expected error when deleting unmerged branch")
	}

	// Force delete the branch
	err = repo.ExecuteGitCommand("branch", "-D", unmergedBranch)
	if err != nil {
		t.Errorf("Failed to force delete branch: %v", err)
	}
}
