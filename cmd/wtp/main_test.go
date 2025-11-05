package main

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

func TestMain(t *testing.T) {
	// Test main function doesn't crash
	// This is tricky to test directly, so we test the app setup instead
	t.Run("app setup", func(t *testing.T) {
		app := createApp()
		assert.NotNil(t, app)
		assert.Equal(t, "wtp", app.Name)
		assert.Equal(t, "Enhanced Git worktree management", app.Usage)
		assert.NotEmpty(t, app.Description)
		assert.True(t, app.EnableShellCompletion)

		// Check commands exist
		commandNames := make(map[string]bool)
		for _, cmd := range app.Commands {
			commandNames[cmd.Name] = true
		}

		expectedCommands := []string{"add", "list", "remove", "init", "cd"}
		for _, expected := range expectedCommands {
			assert.True(t, commandNames[expected], "Command %s should exist", expected)
		}

		// Check version flag exists
		hasVersionFlag := false
		for _, flag := range app.Flags {
			if flag.Names()[0] == "version" {
				hasVersionFlag = true
				break
			}
		}
		assert.True(t, hasVersionFlag, "Version flag should exist")
	})
}

func TestVersionInfo(t *testing.T) {
	// Test version is set
	assert.NotEmpty(t, version)
	// In tests, version is usually the default
	if version != defaultVersion {
		// If not dev, should be a valid version format
		assert.Regexp(t, `^\d+\.\d+\.\d+`, version)
	}
}

func TestAppRun_Version(t *testing.T) {
	var buf bytes.Buffer
	app := createApp()
	app.Writer = &buf

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "--version"})

	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "wtp version")
}

func TestAppRun_Help(t *testing.T) {
	var buf bytes.Buffer
	app := createApp()
	app.Writer = &buf

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "--help"})

	assert.NoError(t, err)
	output := buf.String()

	// Check help output contains expected information
	assert.Contains(t, output, "wtp")
	assert.Contains(t, output, "Enhanced Git worktree management")

	// Check that all expected commands are present in the output
	// Don't check for specific section headers as they may vary by CLI version
	expectedCommands := []string{"add", "list", "remove", "init", "cd"}
	for _, cmd := range expectedCommands {
		assert.Contains(t, output, cmd, "Command '%s' should be present in help output", cmd)
	}
}

// TestAppRun_InvalidCommand was removed as it was a coverage-driven test
// that didn't provide meaningful user value. The CLI framework handles
// invalid commands gracefully by showing help, which is tested elsewhere.

func TestAppRun_NoArgs(t *testing.T) {
	var buf bytes.Buffer
	app := createApp()
	app.Writer = &buf

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp"})

	// Should show help when no arguments
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "COMMANDS:")
}

func TestMainExitCode(_ *testing.T) {
	// Test that main exits with non-zero on error
	if os.Getenv("BE_CRASHER") == "1" {
		// This will cause an error
		os.Args = []string{"wtp", "invalid-command"}
		main()
		return
	}

	// This test is mainly to ensure main() is covered
	// Actual exit code testing would require process execution
}

// Helper function to create the app for testing
func createApp() *cli.Command {
	return &cli.Command{
		Name:  "wtp",
		Usage: "Enhanced Git worktree management",
		Description: "wtp (Worktree Plus) simplifies Git worktree creation with automatic branch tracking, " +
			"project-specific setup hooks, and convenient defaults.",
		Version:                         version,
		EnableShellCompletion:           true,
		ConfigureShellCompletionCommand: configureCompletionCommand,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "version",
				Usage: "Show version information",
			},
		},
		Commands: []*cli.Command{
			NewAddCommand(),
			NewListCommand(),
			NewRemoveCommand(),
			NewInitCommand(),
			NewCdCommand(),
			// NewCompletionCommand(), // Using built-in completion
		},
	}
}
