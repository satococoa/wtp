package e2e

import (
	"strings"
	"testing"

	"github.com/satococoa/wtp/test/e2e/framework"
	"github.com/stretchr/testify/assert"
)

// TestConfigBaseDirIntegration tests that base_dir configuration is properly handled
func TestConfigBaseDirIntegration(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("AddUsesConfigBaseDir", func(t *testing.T) {
		repo := env.CreateTestRepo("config-base-dir")

		// Create config with custom base_dir
		config := `version: 1.0
defaults:
  base_dir: "../my-custom-worktrees"
`
		repo.WriteConfig(config)

		// Verify config file exists
		framework.AssertFileExists(t, repo, ".wtp.yml")

		repo.CreateBranch("feature/test")

		// Add worktree
		output, err := repo.RunWTP("add", "feature/test")
		framework.AssertNoError(t, err)
		framework.AssertWorktreeCreated(t, output, "feature/test")

		// Verify worktree was created in custom directory
		worktrees := repo.ListWorktrees()

		// Using testify's assert for more complex checks
		assert.NotEmpty(t, worktrees, "Should have at least one worktree")

		// Find the feature/test worktree path
		var featureWorktreePath string
		for _, wt := range worktrees {
			if strings.Contains(wt, "feature/test") && wt != repo.Path() {
				featureWorktreePath = wt
				break
			}
		}

		assert.NotEmpty(t, featureWorktreePath, "Should find feature/test worktree")
		assert.Contains(t, featureWorktreePath, "my-custom-worktrees", "Worktree should be created in custom base_dir")
	})

	t.Run("RemoveFindsWorktreeRegardlessOfConfig", func(t *testing.T) {
		repo := env.CreateTestRepo("remove-config-test")

		// Create initial config
		config := `version: 1.0
defaults:
  base_dir: "../initial-worktrees"
`
		repo.WriteConfig(config)
		repo.CreateBranch("feature/movable")

		// Add worktree with initial config
		_, err := repo.RunWTP("add", "feature/movable")
		framework.AssertNoError(t, err)
		framework.AssertWorktreeCount(t, repo, 2)

		// Change config to different base_dir
		newConfig := `version: 1.0
defaults:
  base_dir: "../different-worktrees"
`
		repo.WriteConfig(newConfig)

		// Remove should still find the worktree even though config changed
		output, err := repo.RunWTP("remove", "movable")
		framework.AssertNoError(t, err)
		framework.AssertOutputContains(t, output, "Removed worktree")
		framework.AssertWorktreeCount(t, repo, 1)
	})

	t.Run("ListShowsAllWorktreesRegardlessOfConfig", func(t *testing.T) {
		repo := env.CreateTestRepo("list-config-test")

		// Create worktrees with different configs
		config1 := `version: 1.0
defaults:
  base_dir: "../worktrees-a"
`
		repo.WriteConfig(config1)
		repo.CreateBranch("feature/a")
		_, _ = repo.RunWTP("add", "feature/a")

		config2 := `version: 1.0
defaults:
  base_dir: "../worktrees-b"
`
		repo.WriteConfig(config2)
		repo.CreateBranch("feature/b")
		_, _ = repo.RunWTP("add", "feature/b")

		// List should show all worktrees
		output, err := repo.RunWTP("list")
		framework.AssertNoError(t, err)
		framework.AssertOutputContains(t, output, "feature/a")
		framework.AssertOutputContains(t, output, "feature/b")
		framework.AssertWorktreeCount(t, repo, 3) // main + 2 features
	})
}
