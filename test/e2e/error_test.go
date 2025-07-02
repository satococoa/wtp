package e2e

import (
	"strings"
	"testing"

	"github.com/satococoa/wtp/test/e2e/framework"
)

func TestErrorMessages(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("NotInGitRepository", func(t *testing.T) {
		cmd := env.CreateNonRepoDir("not-git-repo")

		output, err := cmd.RunWTP("add", "branch")
		framework.AssertError(t, err)
		framework.AssertOutputContains(t, output, "not in a git repository")
		framework.AssertHelpfulError(t, output)

		// Check for helpful suggestions
		framework.AssertTrue(t,
			strings.Contains(output, "git init") ||
				strings.Contains(output, "Navigate to") ||
				strings.Contains(output, "Solutions:"),
			"Should provide helpful suggestions for git repository error")
	})

	t.Run("BranchNotFound", func(t *testing.T) {
		repo := env.CreateTestRepo("error-branch-not-found")

		output, err := repo.RunWTP("add", "nonexistent-branch")
		framework.AssertError(t, err)
		framework.AssertOutputContains(t, output, "not found in local or remote branches")
		framework.AssertHelpfulError(t, output)

		// Check for specific suggestions
		suggestions := []string{
			"Check the branch name spelling",
			"git branch -a",
			"Create a new branch",
			"-b",
			"git fetch",
		}

		foundSuggestion := false
		for _, suggestion := range suggestions {
			if strings.Contains(output, suggestion) {
				foundSuggestion = true
				break
			}
		}
		framework.AssertTrue(t, foundSuggestion, "Should provide helpful suggestions for branch not found")
	})

	t.Run("WorktreeAlreadyExists", func(t *testing.T) {
		repo := env.CreateTestRepo("error-worktree-exists")
		repo.CreateBranch("existing-branch")

		_, err := repo.RunWTP("add", "existing-branch")
		framework.AssertNoError(t, err)

		output, err := repo.RunWTP("add", "existing-branch")
		framework.AssertError(t, err)
		framework.AssertOutputContains(t, output, "already exists")

		// Should suggest --force flag or provide helpful error
		framework.AssertTrue(t,
			strings.Contains(output, "--force") ||
				strings.Contains(output, "already checked out") ||
				strings.Contains(output, "Tip:"),
			"Should suggest --force flag or provide helpful tip")
	})

	t.Run("EmptyBranchName", func(t *testing.T) {
		repo := env.CreateTestRepo("error-empty-branch")

		output, err := repo.RunWTP("add")
		framework.AssertError(t, err)
		framework.AssertOutputContains(t, output, "branch name is required")
		framework.AssertHelpfulError(t, output)

		// Should show usage examples
		framework.AssertTrue(t,
			strings.Contains(output, "Usage:") ||
				strings.Contains(output, "Examples:") ||
				strings.Contains(output, "wtp add"),
			"Should show usage examples for missing branch name")
	})

	t.Run("MultipleRemotesError", func(t *testing.T) {
		repo := env.CreateTestRepo("error-multiple-remotes")
		repo.AddRemote("origin", "https://example.com/repo.git")
		repo.AddRemote("upstream", "https://example.com/upstream.git")
		repo.CreateRemoteBranch("origin", "ambiguous-branch")
		repo.CreateRemoteBranch("upstream", "ambiguous-branch")

		output, err := repo.RunWTP("add", "ambiguous-branch")
		framework.AssertError(t, err)
		framework.AssertOutputContains(t, output, "exists in multiple remotes")
		framework.AssertOutputContains(t, output, "origin")
		framework.AssertOutputContains(t, output, "upstream")
		framework.AssertHelpfulError(t, output)

		// Should suggest --track flag
		framework.AssertOutputContains(t, output, "--track")
	})

	t.Run("InvalidBranchName", func(t *testing.T) {
		repo := env.CreateTestRepo("error-invalid-branch")

		invalidNames := []string{
			"..",
			"branch with spaces",
			"branch@{invalid}",
		}

		for _, name := range invalidNames {
			output, err := repo.RunWTP("add", "-b", name)
			framework.AssertError(t, err)
			framework.AssertTrue(t,
				strings.Contains(output, "invalid") ||
					strings.Contains(output, "failed") ||
					strings.Contains(output, "error"),
				"Should show error for invalid branch name: "+name)
		}
	})

	t.Run("RemoveNonexistentWorktree", func(t *testing.T) {
		repo := env.CreateTestRepo("error-remove-nonexistent")

		output, err := repo.RunWTP("remove", "nonexistent-worktree")
		framework.AssertError(t, err)
		framework.AssertTrue(t,
			strings.Contains(output, "failed to remove") ||
				strings.Contains(output, "not found") ||
				strings.Contains(output, "does not exist"),
			"Should show clear error for removing non-existent worktree")
	})

	t.Run("CDWithoutShellIntegration", func(t *testing.T) {
		repo := env.CreateTestRepo("error-cd-no-shell")
		repo.CreateBranch("test-branch")
		repo.RunWTP("add", "test-branch")

		// Run cd command without shell integration
		output, err := repo.RunWTP("cd", "test-branch")
		framework.AssertError(t, err)
		framework.AssertOutputContains(t, output, "requires shell integration")
		framework.AssertHelpfulError(t, output)

		// Should provide setup instructions
		framework.AssertTrue(t,
			strings.Contains(output, "eval") ||
				strings.Contains(output, "shell-init") ||
				strings.Contains(output, "Setup:"),
			"Should provide shell integration setup instructions")
	})
}

func TestGitCommandErrors(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("GitCommandFailure", func(t *testing.T) {
		repo := env.CreateTestRepo("error-git-command")

		// Try to add worktree with invalid path characters
		output, err := repo.RunWTP("add", "--path", "/dev/null/invalid", "-b", "test")
		framework.AssertError(t, err)
		framework.AssertTrue(t,
			strings.Contains(output, "failed") ||
				strings.Contains(output, "error") ||
				strings.Contains(output, "Original error:"),
			"Should show git command failure")

		// Should provide helpful context
		framework.AssertTrue(t,
			strings.Contains(output, "Tip:") ||
				strings.Contains(output, "Details:") ||
				strings.Contains(output, "git command"),
			"Should provide context for git command failure")
	})

	t.Run("PermissionDenied", func(t *testing.T) {
		repo := env.CreateTestRepo("error-permission")

		// Create a directory with restricted permissions
		restrictedPath := env.TmpDir() + "/restricted"
		env.RunInDir(env.TmpDir(), "mkdir", "-p", restrictedPath)
		env.RunInDir(env.TmpDir(), "chmod", "000", restrictedPath)

		// Cleanup permission after test
		defer func() {
			env.RunInDir(env.TmpDir(), "chmod", "755", restrictedPath)
		}()

		output, err := repo.RunWTP("add", "--path", restrictedPath+"/worktree", "-b", "test")
		framework.AssertError(t, err)
		framework.AssertTrue(t,
			strings.Contains(output, "permission") ||
				strings.Contains(output, "denied") ||
				strings.Contains(output, "failed"),
			"Should handle permission errors gracefully")
	})
}

func TestValidationErrors(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("ConflictingFlags", func(t *testing.T) {
		repo := env.CreateTestRepo("error-conflicting-flags")

		// Try conflicting flags (this might vary based on implementation)
		output, err := repo.RunWTP("add", "-b", "new-branch", "--detach", "main")
		// This might or might not be an error depending on git's behavior
		if err != nil {
			framework.AssertTrue(t,
				strings.Contains(output, "conflict") ||
					strings.Contains(output, "cannot") ||
					strings.Contains(output, "incompatible"),
				"Should handle conflicting flags")
		}
	})

	t.Run("MissingRequiredArguments", func(t *testing.T) {
		repo := env.CreateTestRepo("error-missing-args")

		// Test remove without branch name
		output, err := repo.RunWTP("remove")
		framework.AssertError(t, err)
		framework.AssertTrue(t,
			strings.Contains(output, "required") ||
				strings.Contains(output, "missing") ||
				strings.Contains(output, "Usage:"),
			"Should show error for missing required arguments")
	})
}
