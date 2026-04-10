package e2e

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"gopkg.in/yaml.v3"

	"github.com/satococoa/wtp/v2/test/e2e/framework"
)

func TestHookOutputStreaming(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	repo := env.CreateTestRepo("hook-streaming")

	// Create config with a hook that outputs with delays
	hookCmd := "echo 'Starting hook...'; sleep 0.1; echo 'Processing...'; sleep 0.1; echo 'Completed!'"
	if runtime.GOOS == "windows" {
		hookCmd = "echo Starting hook... & echo Processing... & echo Completed!"
	}
	config := map[string]any{
		"version": "1",
		"hooks": map[string]any{
			"post_create": []map[string]any{
				{
					"type":    "command",
					"command": hookCmd,
				},
			},
		},
	}

	configData, err := yaml.Marshal(config)
	framework.AssertNoError(t, err)

	configPath := filepath.Join(repo.Path(), ".wtp.yml")
	framework.AssertNoError(t, os.WriteFile(configPath, configData, 0644))

	// Run add command with -b flag to create new branch and capture output
	output, err := repo.RunWTP("add", "-b", "test-branch")
	framework.AssertNoError(t, err)

	// Verify output contains all expected messages in order
	framework.AssertOutputContains(t, output, "Executing post-create hooks...")
	framework.AssertOutputContains(t, output, "Starting hook...")
	framework.AssertOutputContains(t, output, "Processing...")
	framework.AssertOutputContains(t, output, "Completed!")
	framework.AssertOutputContains(t, output, "✓ All hooks executed successfully")

	// Verify worktree was created (default location is ../worktrees/branch-name)
	worktreePath := filepath.Join(repo.Path(), "..", "worktrees", "test-branch")
	framework.AssertWorktreeExists(t, repo, worktreePath)
}
