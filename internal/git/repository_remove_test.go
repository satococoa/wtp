package git

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func setupTestRepoWithBranches(t *testing.T) (repoDir string, mergedBranch string, unmergedBranch string) {
	repoDir = t.TempDir()

	// Initialize git repository
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to init git repo: %v", err)
	}

	// Configure git user
	cmd = exec.Command("git", "config", "user.name", "Test User")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git user: %v", err)
	}

	cmd = exec.Command("git", "config", "user.email", "test@example.com")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to configure git email: %v", err)
	}

	// Create initial commit
	readmeFile := filepath.Join(repoDir, "README.md")
	if err := os.WriteFile(readmeFile, []byte("# Test Repo"), 0644); err != nil {
		t.Fatalf("Failed to write README: %v", err)
	}

	cmd = exec.Command("git", "add", "README.md")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add README: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Initial commit")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit: %v", err)
	}

	// Create merged branch
	mergedBranch = "merged-branch"
	cmd = exec.Command("git", "checkout", "-b", mergedBranch)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create merged branch: %v", err)
	}

	// Add a commit on merged branch
	mergedFile := filepath.Join(repoDir, "merged.txt")
	if err := os.WriteFile(mergedFile, []byte("merged content"), 0644); err != nil {
		t.Fatalf("Failed to write merged file: %v", err)
	}

	cmd = exec.Command("git", "add", "merged.txt")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add merged file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Add merged file")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit merged file: %v", err)
	}

	// Switch back to main/master and merge
	cmd = exec.Command("git", "checkout", "main")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		// Try master if main doesn't exist
		cmd = exec.Command("git", "checkout", "master")
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to checkout main/master: %v", err)
		}
	}

	cmd = exec.Command("git", "merge", mergedBranch)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to merge branch: %v", err)
	}

	// Create unmerged branch
	unmergedBranch = "unmerged-branch"
	cmd = exec.Command("git", "checkout", "-b", unmergedBranch)
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to create unmerged branch: %v", err)
	}

	// Add a commit on unmerged branch
	unmergedFile := filepath.Join(repoDir, "unmerged.txt")
	if err := os.WriteFile(unmergedFile, []byte("unmerged content"), 0644); err != nil {
		t.Fatalf("Failed to write unmerged file: %v", err)
	}

	cmd = exec.Command("git", "add", "unmerged.txt")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to add unmerged file: %v", err)
	}

	cmd = exec.Command("git", "commit", "-m", "Add unmerged file")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("Failed to commit unmerged file: %v", err)
	}

	// Switch back to main/master
	cmd = exec.Command("git", "checkout", "main")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		// Try master if main doesn't exist
		cmd = exec.Command("git", "checkout", "master")
		cmd.Dir = repoDir
		if err := cmd.Run(); err != nil {
			t.Fatalf("Failed to checkout main/master: %v", err)
		}
	}

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