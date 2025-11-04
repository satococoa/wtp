package e2e

import (
	"os"
	"strings"
	"testing"

	"github.com/satococoa/wtp/v2/test/e2e/framework"
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

	t.Run("BranchConflict", func(t *testing.T) {
		repo := env.CreateTestRepo("worktree-conflict")
		repo.CreateBranch("feature/conflict")

		_, err := repo.RunWTP("add", "feature/conflict")
		framework.AssertNoError(t, err)

		// Try to add the same branch again (should fail without --force flag which is removed)
		output, err := repo.RunWTP("add", "feature/conflict")
		framework.AssertError(t, err)
		framework.AssertOutputContains(t, output, "already exists")
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

	t.Run("RemoveOnlyWorksWithinBaseDir", func(t *testing.T) {
		repo := env.CreateTestRepo("remove-different-basedir")

		// Create worktree with default location
		env.RunInDir(repo.Path(), "git", "worktree", "add", "../worktrees/feature/remove-test", "-b", "feature/remove-test")

		// Create config with different base_dir
		configContent := `version: 1
defaults:
  base_dir: custom-location`
		env.WriteFile(repo.Path()+"/.wtp.yml", configContent)

		// Remove should NOT work because worktree is outside the configured base_dir
		output, err := repo.RunWTP("remove", "remove-test")
		framework.AssertError(t, err)
		framework.AssertOutputContains(t, output, "not found")

		// Verify worktree is still there since remove failed
		worktreePath := env.TmpDir() + "/worktrees/feature/remove-test"
		framework.AssertTrue(t, env.FileExists(worktreePath), "Worktree should still exist")
	})
}

func TestWorktreeList(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("EmptyList", func(t *testing.T) {
		repo := env.CreateTestRepo("list-empty")

		output, err := repo.RunWTP("list")
		framework.AssertNoError(t, err)
		framework.AssertTrue(t,
			strings.Contains(output, "@") || strings.Contains(output, "main"),
			"Should show main worktree")
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
		framework.AssertTrue(t,
			strings.Contains(output, "@") || strings.Contains(output, "main"),
			"Should show main worktree")
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
		repo := env.CreateTestRepo("config-basedir")

		// Create config with custom base_dir
		configContent := `version: "1.0"
defaults:
  base_dir: custom-worktrees`
		env.WriteFile(repo.Path()+"/.wtp.yml", configContent)

		// Create worktree with config
		output, err := repo.RunWTP("add", "-b", "feature/custom-dir")
		framework.AssertNoError(t, err)

		// Check if worktree was created in custom base_dir
		customPath := repo.Path() + "/custom-worktrees/feature/custom-dir"
		framework.AssertTrue(t, env.FileExists(customPath+"/.git"), "Worktree .git should exist")
		framework.AssertOutputContains(t, output, "custom-worktrees/feature/custom-dir")
	})

	t.Run("PostCreateHook", func(t *testing.T) {
		repo := env.CreateTestRepo("config-hooks")

		// Create source file for copy hook
		env.WriteFile(repo.Path()+"/template.txt", "template content")

		// Create config with hooks
		configContent := `version: "1.0"
defaults:
  base_dir: ../worktrees
hooks:
  post_create:
    - type: copy
      from: template.txt
      to: copied.txt
    - type: command
      command: touch hook-executed.txt`
		env.WriteFile(repo.Path()+"/.wtp.yml", configContent)

		// Create worktree with hooks
		output, err := repo.RunWTP("add", "-b", "feature/hooks")
		framework.AssertNoError(t, err)

		// Verify hooks were executed
		worktreePath := env.TmpDir() + "/worktrees/feature/hooks"
		framework.AssertTrue(t, env.FileExists(worktreePath+"/copied.txt"), "Copied file should exist")
		framework.AssertTrue(t, env.FileExists(worktreePath+"/hook-executed.txt"), "Hook-executed file should exist")
		framework.AssertOutputContains(t, output, "Executing post-create hooks")
		framework.AssertOutputContains(t, output, "All hooks executed successfully")

		// Verify copied file content manually since worktree isn't a TestRepo
		copiedContent, err := os.ReadFile(worktreePath + "/copied.txt")
		framework.AssertNoError(t, err)
		framework.AssertEqual(t, "template content", string(copiedContent))
	})
}
