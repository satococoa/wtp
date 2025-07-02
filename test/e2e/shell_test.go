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

	t.Run("CDCommandWithoutIntegration", func(t *testing.T) {
		repo := env.CreateTestRepo("shell-cd-without")
		repo.CreateBranch("test-branch")
		repo.RunWTP("add", "test-branch")

		// Try cd without shell integration
		output, err := repo.RunWTP("cd", "test-branch")
		framework.AssertError(t, err)
		framework.AssertOutputContains(t, output, "requires shell integration")
		framework.AssertHelpfulError(t, output)

		// Should provide setup instructions
		framework.AssertOutputContains(t, output, "eval")
		framework.AssertTrue(t,
			strings.Contains(output, "shell-init") ||
				strings.Contains(output, "Setup:"),
			"Should provide shell integration setup instructions")
	})

	t.Run("CDCommandWithIntegration", func(t *testing.T) {
		repo := env.CreateTestRepo("shell-cd-with")
		repo.CreateBranch("feature/test")
		repo.RunWTP("add", "feature/test")

		// Simulate shell integration environment
		os.Setenv("WTP_SHELL_INTEGRATION", "1")
		defer os.Unsetenv("WTP_SHELL_INTEGRATION")

		output, err := repo.RunWTP("cd", "feature/test")
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

	t.Run("ShellInitCommand", func(t *testing.T) {
		repo := env.CreateTestRepo("shell-init")

		output, err := repo.RunWTP("shell-init")
		framework.AssertNoError(t, err)

		// Should output shell functions
		framework.AssertTrue(t,
			strings.Contains(output, "function") ||
				strings.Contains(output, "wtp()") ||
				strings.Contains(output, "complete") ||
				strings.Contains(output, "compdef"),
			"Should output shell integration code")
	})

	t.Run("ShellInitWithCDFlag", func(t *testing.T) {
		repo := env.CreateTestRepo("shell-init-cd")

		output, err := repo.RunWTP("shell-init", "--cd")
		framework.AssertNoError(t, err)

		// Should include cd functionality
		framework.AssertTrue(t,
			strings.Contains(output, "cd") ||
				strings.Contains(output, "WTP_SHELL_INTEGRATION"),
			"Should include cd functionality with --cd flag")

		// Should include completion
		framework.AssertTrue(t,
			strings.Contains(output, "complete") ||
				strings.Contains(output, "compdef"),
			"Should include completion functionality")
	})

	t.Run("CompletionCommand", func(t *testing.T) {
		repo := env.CreateTestRepo("shell-completion")

		// Test bash completion
		output, err := repo.RunWTP("completion", "bash")
		framework.AssertNoError(t, err)
		framework.AssertOutputContains(t, output, "complete")
		framework.AssertTrue(t,
			strings.Contains(output, "wtp") ||
				strings.Contains(output, "_wtp"),
			"Should output bash completion script")

		// Test zsh completion
		output, err = repo.RunWTP("completion", "zsh")
		framework.AssertNoError(t, err)
		framework.AssertTrue(t,
			strings.Contains(output, "compdef") ||
				strings.Contains(output, "#compdef") ||
				strings.Contains(output, "compadd"),
			"Should output zsh completion script")

		// Test fish completion
		output, err = repo.RunWTP("completion", "fish")
		framework.AssertNoError(t, err)
		framework.AssertTrue(t,
			strings.Contains(output, "complete") ||
				strings.Contains(output, "fish"),
			"Should output fish completion script")
	})

	t.Run("CompletionInvalidShell", func(t *testing.T) {
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

		repo.RunWTP("add", "feature/one")
		repo.RunWTP("add", "feature/two")
		repo.RunWTP("add", "bugfix/three")

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
		repo.RunWTP("add", "feature/authentication")
		repo.RunWTP("add", "feature/authorization")

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

	t.Run("ShellDetection", func(t *testing.T) {
		repo := env.CreateTestRepo("shell-detection")

		// Test with different SHELL env vars
		shells := map[string]string{
			"/bin/bash":     "bash",
			"/usr/bin/zsh":  "zsh",
			"/usr/bin/fish": "fish",
		}

		for shellPath, shellName := range shells {
			os.Setenv("SHELL", shellPath)
			output, err := repo.RunWTP("shell-init")
			os.Unsetenv("SHELL")

			framework.AssertNoError(t, err)
			framework.AssertTrue(t,
				strings.Contains(output, shellName) ||
					strings.Contains(output, "complete") ||
					strings.Contains(output, "function"),
				"Should generate appropriate shell code for "+shellName)
		}
	})

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
