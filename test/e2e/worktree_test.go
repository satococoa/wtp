package e2e

import (
	"strings"
	"testing"

	"github.com/satococoa/wtp/test/e2e/framework"
)

func TestWorktreeCreation(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("LocalBranch", func(t *testing.T) {
		repo := env.CreateTestRepo("worktree-local")
		repo.CreateBranch("feature/test-branch")

		output, err := repo.RunWTP("add", "feature/test-branch")
		framework.AssertNoError(t, err)
		framework.AssertWorktreeCreated(t, output, "feature/test-branch")
		
		worktrees := repo.ListWorktrees()
		framework.AssertEqual(t, 2, len(worktrees))
		framework.AssertWorktreeExists(t, repo, "feature/test-branch")
	})

	t.Run("NonexistentBranch", func(t *testing.T) {
		repo := env.CreateTestRepo("worktree-nonexistent")

		output, err := repo.RunWTP("add", "nonexistent-branch")
		framework.AssertError(t, err)
		framework.AssertOutputContains(t, output, "not found in local or remote branches")
		framework.AssertOutputContains(t, output, "Create a new branch with")
		framework.AssertHelpfulError(t, output)
	})

	t.Run("NewBranch", func(t *testing.T) {
		repo := env.CreateTestRepo("worktree-new")

		output, err := repo.RunWTP("add", "-b", "new-feature")
		framework.AssertNoError(t, err)
		framework.AssertWorktreeCreated(t, output, "new-feature")
		
		worktrees := repo.ListWorktrees()
		framework.AssertEqual(t, 2, len(worktrees))
		framework.AssertWorktreeExists(t, repo, "new-feature")
	})

	t.Run("NewBranchFromCommit", func(t *testing.T) {
		repo := env.CreateTestRepo("worktree-new-commit")
		repo.CreateBranch("develop")

		output, err := repo.RunWTP("add", "-b", "hotfix", "develop")
		framework.AssertNoError(t, err)
		framework.AssertWorktreeCreated(t, output, "hotfix")
		framework.AssertWorktreeExists(t, repo, "hotfix")
	})

	t.Run("CustomPath", func(t *testing.T) {
		repo := env.CreateTestRepo("worktree-custom-path")
		repo.CreateBranch("feature/custom")

		customPath := env.TmpDir() + "/custom-worktree"
		output, err := repo.RunWTP("add", "--path", customPath, "feature/custom")
		framework.AssertNoError(t, err)
		framework.AssertWorktreeCreated(t, output, "feature/custom")
		framework.AssertWorktreeExists(t, repo, customPath)
	})

	t.Run("DetachedHead", func(t *testing.T) {
		repo := env.CreateTestRepo("worktree-detached")
		
		commit := repo.GetCommitHash()[:7]

		_, err := repo.RunWTP("add", "--detach", commit)
		framework.AssertNoError(t, err)
		framework.AssertWorktreeExists(t, repo, commit)
	})

	t.Run("ForceCheckout", func(t *testing.T) {
		repo := env.CreateTestRepo("worktree-force")
		repo.CreateBranch("feature/force")

		_, err := repo.RunWTP("add", "feature/force")
		framework.AssertNoError(t, err)

		customPath := env.TmpDir() + "/force-worktree"
		output, err := repo.RunWTP("add", "--path", customPath, "--force", "feature/force")
		framework.AssertNoError(t, err)
		framework.AssertWorktreeCreated(t, output, "feature/force")
	})

	t.Run("BranchWithSlashes", func(t *testing.T) {
		repo := env.CreateTestRepo("worktree-slashes")
		repo.CreateBranch("feature/auth/login")

		output, err := repo.RunWTP("add", "feature/auth/login")
		framework.AssertNoError(t, err)
		framework.AssertWorktreeCreated(t, output, "feature/auth/login")
		framework.AssertWorktreeExists(t, repo, "feature/auth/login")
	})
}

func TestWorktreeRemoval(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("RemoveWorktree", func(t *testing.T) {
		repo := env.CreateTestRepo("remove-test")
		repo.CreateBranch("feature/remove")

		_, err := repo.RunWTP("add", "feature/remove")
		framework.AssertNoError(t, err)
		framework.AssertWorktreeCount(t, repo, 2)

		output, err := repo.RunWTP("remove", "feature/remove")
		framework.AssertNoError(t, err)
		framework.AssertOutputContains(t, output, "Removed worktree")
		framework.AssertWorktreeCount(t, repo, 1)
	})

	t.Run("RemoveNonexistent", func(t *testing.T) {
		repo := env.CreateTestRepo("remove-nonexistent")

		output, err := repo.RunWTP("remove", "nonexistent")
		framework.AssertError(t, err)
		framework.AssertOutputContains(t, output, "not found")
		// Should show available worktrees
		framework.AssertTrue(t, 
			strings.Contains(output, "Available worktrees:") ||
			strings.Contains(output, "No worktrees found"),
			"Should show available worktrees or indicate none found")
	})

	t.Run("ForceRemove", func(t *testing.T) {
		repo := env.CreateTestRepo("remove-force")
		repo.CreateBranch("feature/force-remove")

		_, err := repo.RunWTP("add", "feature/force-remove")
		framework.AssertNoError(t, err)

		// Get actual worktree path from git
		worktrees := repo.ListWorktrees()
		var worktreePath string
		for _, wt := range worktrees {
			if strings.Contains(wt, "feature/force-remove") {
				worktreePath = wt
				break
			}
		}
		
		if worktreePath != "" {
			env.WriteFile(worktreePath+"/dirty.txt", "uncommitted changes")
		}

		output, err := repo.RunWTP("remove", "--force", "feature/force-remove")
		framework.AssertNoError(t, err)
		framework.AssertOutputContains(t, output, "Removed worktree")
	})

	t.Run("RemoveWithDifferentBaseDir", func(t *testing.T) {
		t.Skip("Testing with different base_dir requires config support")
		// This test is to ensure remove works regardless of config base_dir
		// since we now use git's worktree list to find the actual path
	})
}

func TestWorktreeList(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("EmptyList", func(t *testing.T) {
		repo := env.CreateTestRepo("list-empty")

		output, err := repo.RunWTP("list")
		framework.AssertNoError(t, err)
		framework.AssertTrue(t, strings.Contains(output, repo.Path()) || strings.Contains(output, "main"), "Should show main worktree")
	})

	t.Run("MultipleWorktrees", func(t *testing.T) {
		repo := env.CreateTestRepo("list-multiple")
		repo.CreateBranch("feature/one")
		repo.CreateBranch("feature/two")

		_, err := repo.RunWTP("add", "feature/one")
		framework.AssertNoError(t, err)
		_, err = repo.RunWTP("add", "feature/two")
		framework.AssertNoError(t, err)

		output, err := repo.RunWTP("list")
		framework.AssertNoError(t, err)
		framework.AssertOutputContains(t, output, "feature/one")
		framework.AssertOutputContains(t, output, "feature/two")
		framework.AssertTrue(t, strings.Contains(output, "main") || strings.Contains(output, repo.Path()), "Should show main worktree")
	})

}

func TestWorktreeValidation(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("EmptyBranchName", func(t *testing.T) {
		repo := env.CreateTestRepo("validate-empty")

		output, err := repo.RunWTP("add")
		framework.AssertError(t, err)
		framework.AssertOutputContains(t, output, "branch name is required")
		framework.AssertHelpfulError(t, output)
	})

	t.Run("InvalidBranchName", func(t *testing.T) {
		repo := env.CreateTestRepo("validate-invalid")

		invalidNames := []string{
			"..",
			"branch with spaces",
			"branch@{invalid}",
			"branch..name",
			"branch~name",
			"branch^name",
			"branch:name",
		}

		for _, name := range invalidNames {
			_, err := repo.RunWTP("add", "-b", name)
			framework.AssertError(t, err)
			framework.AssertTrue(t, err != nil, "Should fail for invalid branch name: "+name)
		}
	})

	t.Run("AlreadyCheckedOut", func(t *testing.T) {
		repo := env.CreateTestRepo("validate-checked-out")
		repo.CreateBranch("feature/duplicate")

		_, err := repo.RunWTP("add", "feature/duplicate")
		framework.AssertNoError(t, err)

		output, err := repo.RunWTP("add", "feature/duplicate")
		framework.AssertError(t, err)
		framework.AssertOutputContains(t, output, "already exists")
	})
}

func TestWorktreeWithConfig(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("CustomBaseDir", func(t *testing.T) {
		t.Skip("Skipping config test - config functionality may vary")
	})

	t.Run("PostCreateHook", func(t *testing.T) {
		t.Skip("Skipping hook test - hook functionality may vary")
	})
}