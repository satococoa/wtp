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

		// Should provide helpful error message (--force flag removed in simplified interface)
		framework.AssertTrue(t,
			strings.Contains(output, "already checked out") ||
				strings.Contains(output, "already exists") ||
				strings.Contains(output, "Tip:"),
			"Should provide helpful error message")
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

		// Should provide helpful guidance for ambiguous remote branches
		framework.AssertTrue(t,
			strings.Contains(output, "-b") ||
				strings.Contains(output, "specify") ||
				strings.Contains(output, "remote") ||
				strings.Contains(output, "wtp add"),
			"Should provide helpful guidance for multiple remotes")
	})
}

func TestErrorMessagesValidation(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

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
		_, _ = repo.RunWTP("add", "test-branch")

		// Run cd command without shell integration
		output, err := repo.RunWTP("cd", "test-branch")
		framework.AssertError(t, err)
		framework.AssertOutputContains(t, output, "requires shell integration")
		framework.AssertHelpfulError(t, output)

		// Should provide setup instructions
		framework.AssertTrue(t,
			strings.Contains(output, "eval") ||
				strings.Contains(output, "completion") ||
				strings.Contains(output, "Setup:"),
			"Should provide shell integration setup instructions")
	})
}

func TestValidationErrors(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

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
