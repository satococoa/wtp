package e2e

import (
	"os"
	"strings"
	"testing"

	"github.com/satococoa/wtp/test/e2e/framework"
)

func TestShellIntegration(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("CDCommandOutputsPath", func(t *testing.T) {
		repo := env.CreateTestRepo("shell-cd-path")
		repo.CreateBranch("test-branch")
		_, _ = repo.RunWTP("add", "test-branch")

		// cd should always output the path
		output, err := repo.RunWTP("cd", "test-branch")
		framework.AssertNoError(t, err)
		framework.AssertOutputContains(t, output, "test-branch")

		// Output should be a valid path
		outputPath := strings.TrimSpace(output)
		framework.AssertTrue(t, strings.Contains(outputPath, "test-branch"), "Should contain worktree name")
	})

	t.Run("CDCommandWithBranchResolution", func(t *testing.T) {
		repo := env.CreateTestRepo("shell-cd-branch")
		repo.CreateBranch("feature/test")
		_, _ = repo.RunWTP("add", "feature/test")

		// cd should resolve branch name to path
		output, err := repo.RunWTP("cd", "test")
		framework.AssertNoError(t, err)
		// Should output the path
		framework.AssertTrue(t,
			strings.Contains(output, "worktrees/feature/test") ||
				strings.Contains(output, "feature/test"),
			"Should output worktree path")

		// Should not contain error messages
		framework.AssertFalse(t,
			strings.Contains(output, "error") ||
				strings.Contains(output, "Error"),
			"Should not contain error messages")
	})

	t.Run("CDNonexistentWorktree", func(t *testing.T) {
		repo := env.CreateTestRepo("shell-cd-nonexistent")

		// Simulate shell integration
		os.Setenv("WTP_SHELL_INTEGRATION", "1")
		defer os.Unsetenv("WTP_SHELL_INTEGRATION")

		output, err := repo.RunWTP("cd", "nonexistent")
		framework.AssertError(t, err)
		framework.AssertTrue(t,
			strings.Contains(output, "not found") ||
				strings.Contains(output, "does not exist") ||
				strings.Contains(output, "Available worktrees:"),
			"Should show helpful error for non-existent worktree")
	})
}

func TestShellCompletionCommands(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("BashCompletion", func(t *testing.T) {
		repo := env.CreateTestRepo("shell-completion-bash")

		output, err := repo.RunWTP("completion", "bash")
		framework.AssertNoError(t, err)
		framework.AssertOutputContains(t, output, "complete")
		framework.AssertTrue(t,
			strings.Contains(output, "wtp") ||
				strings.Contains(output, "_wtp"),
			"Should output bash completion script")
	})

	t.Run("ZshCompletion", func(t *testing.T) {
		repo := env.CreateTestRepo("shell-completion-zsh")

		output, err := repo.RunWTP("completion", "zsh")
		framework.AssertNoError(t, err)
		framework.AssertTrue(t,
			strings.Contains(output, "compdef") ||
				strings.Contains(output, "#compdef") ||
				strings.Contains(output, "compadd"),
			"Should output zsh completion script")
	})

	t.Run("FishCompletion", func(t *testing.T) {
		repo := env.CreateTestRepo("shell-completion-fish")

		output, err := repo.RunWTP("completion", "fish")
		framework.AssertNoError(t, err)
		framework.AssertTrue(t,
			strings.Contains(output, "complete") ||
				strings.Contains(output, "fish"),
			"Should output fish completion script")
	})

	t.Run("InvalidShell", func(t *testing.T) {
		repo := env.CreateTestRepo("shell-completion-invalid")

		output, err := repo.RunWTP("completion", "invalid-shell")
		framework.AssertError(t, err)
		framework.AssertTrue(t,
			strings.Contains(output, "unsupported") ||
				strings.Contains(output, "invalid") ||
				strings.Contains(output, "bash") ||
				strings.Contains(output, "zsh") ||
				strings.Contains(output, "fish"),
			"Should show error for invalid shell type")
	})
}

func TestShellCompletionBehavior(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("CompletionWithMultipleWorktrees", func(t *testing.T) {
		repo := env.CreateTestRepo("completion-multiple")
		repo.CreateBranch("feature/one")
		repo.CreateBranch("feature/two")
		repo.CreateBranch("bugfix/three")

		_, _ = repo.RunWTP("add", "feature/one")
		_, _ = repo.RunWTP("add", "feature/two")
		_, _ = repo.RunWTP("add", "bugfix/three")

		// Simulate shell integration for cd command
		os.Setenv("WTP_SHELL_INTEGRATION", "1")
		defer os.Unsetenv("WTP_SHELL_INTEGRATION")

		// List command should work and show all worktrees
		output, err := repo.RunWTP("list")
		framework.AssertNoError(t, err)
		framework.AssertOutputContains(t, output, "feature/one")
		framework.AssertOutputContains(t, output, "feature/two")
		framework.AssertOutputContains(t, output, "bugfix/three")
	})

	t.Run("CDWithPartialMatch", func(t *testing.T) {
		repo := env.CreateTestRepo("cd-partial")
		repo.CreateBranch("feature/authentication")
		repo.CreateBranch("feature/authorization")
		_, _ = repo.RunWTP("add", "feature/authentication")
		_, _ = repo.RunWTP("add", "feature/authorization")

		os.Setenv("WTP_SHELL_INTEGRATION", "1")
		defer os.Unsetenv("WTP_SHELL_INTEGRATION")

		// Try partial match (this behavior might vary)
		output, err := repo.RunWTP("cd", "auth")
		if err != nil {
			// Should suggest available options
			framework.AssertTrue(t,
				strings.Contains(output, "authentication") ||
					strings.Contains(output, "authorization") ||
					strings.Contains(output, "Available"),
				"Should suggest matching worktrees")
		}
	})
}

func TestShellEnvironment(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("PowerShellCompletion", func(t *testing.T) {
		repo := env.CreateTestRepo("powershell-completion")

		// PowerShell might not be supported, but test the behavior
		output, err := repo.RunWTP("completion", "powershell")
		if err != nil {
			framework.AssertTrue(t,
				strings.Contains(output, "not supported") ||
					strings.Contains(output, "unsupported") ||
					strings.Contains(output, "bash") ||
					strings.Contains(output, "zsh"),
				"Should handle PowerShell completion request appropriately")
		} else {
			// If supported, should have PowerShell syntax
			framework.AssertTrue(t,
				strings.Contains(output, "Register-ArgumentCompleter") ||
					strings.Contains(output, "param"),
				"Should output PowerShell completion if supported")
		}
	})
}
