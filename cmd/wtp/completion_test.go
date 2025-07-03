package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

func TestNewCompletionCommand(t *testing.T) {
	cmd := NewCompletionCommand()

	assert.NotNil(t, cmd)
	assert.Equal(t, "completion", cmd.Name)
	assert.Equal(t, "Generate shell completion script", cmd.Usage)
	assert.NotEmpty(t, cmd.Description)

	// Verify subcommands exist
	subcommands := make(map[string]*cli.Command)
	for _, sub := range cmd.Commands {
		subcommands[sub.Name] = sub
	}

	// Verify required subcommands exist
	assert.Contains(t, subcommands, "bash")
	assert.Contains(t, subcommands, "zsh")
	assert.Contains(t, subcommands, "fish")
	assert.Contains(t, subcommands, "powershell")
	assert.Contains(t, subcommands, "__branches")
	assert.Contains(t, subcommands, "__worktrees")

	// Verify each shell subcommand has proper action
	assert.NotNil(t, subcommands["bash"].Action)
	assert.NotNil(t, subcommands["zsh"].Action)
	assert.NotNil(t, subcommands["fish"].Action)
	assert.NotNil(t, subcommands["powershell"].Action)

	// Verify internal commands are hidden
	assert.True(t, subcommands["__branches"].Hidden)
	assert.True(t, subcommands["__worktrees"].Hidden)
}

func TestCompletionBash(t *testing.T) {
	app := &cli.Command{
		Commands: []*cli.Command{
			NewCompletionCommand(),
		},
	}

	var buf bytes.Buffer
	app.Writer = &buf

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "completion", "bash"})
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "#!/bin/bash")
	assert.Contains(t, output, "_wtp_completion")
	assert.Contains(t, output, "complete -F _wtp_completion wtp")
	assert.Contains(t, output, "wtp cd command integration")
}

func TestCompletionZsh(t *testing.T) {
	app := &cli.Command{
		Commands: []*cli.Command{
			NewCompletionCommand(),
		},
	}

	var buf bytes.Buffer
	app.Writer = &buf

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "completion", "zsh"})
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "#compdef wtp")
	assert.Contains(t, output, "_wtp()")
	assert.Contains(t, output, "compdef _wtp wtp")
	assert.Contains(t, output, "wtp cd command integration")
}

func TestCompletionFish(t *testing.T) {
	app := &cli.Command{
		Commands: []*cli.Command{
			NewCompletionCommand(),
		},
	}

	var buf bytes.Buffer
	app.Writer = &buf

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "completion", "fish"})
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "wtp")
	assert.Contains(t, output, "complete")
	assert.Contains(t, output, "wtp cd command integration")
	assert.Contains(t, output, "function wtp")
}

func TestCompletionPowerShell(t *testing.T) {
	cmd := NewCompletionCommand()
	powershellCmd := findSubcommand(cmd, "powershell")

	assert.NotNil(t, powershellCmd)

	ctx := context.Background()
	err := powershellCmd.Action(ctx, &cli.Command{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "PowerShell completion is not supported")
}

func TestCompleteBranches(t *testing.T) {
	// Create a Git repository
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	err := os.MkdirAll(gitDir, 0755)
	assert.NoError(t, err)

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	err = os.Chdir(tempDir)
	assert.NoError(t, err)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call printBranches directly
	printBranches(w)

	w.Close()
	os.Stdout = oldStdout

	// Read output - should be empty since not a real git repo
	readBuf := make([]byte, 1024)
	n, _ := r.Read(readBuf)
	output := string(readBuf[:n])

	// Should be empty or have no output
	assert.Empty(t, output)
}

func TestCompleteWorktrees(t *testing.T) {
	// Create a Git repository
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	err := os.MkdirAll(gitDir, 0755)
	assert.NoError(t, err)

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	err = os.Chdir(tempDir)
	assert.NoError(t, err)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	// Call printWorktrees directly
	printWorktrees(w)

	w.Close()
	os.Stdout = oldStdout

	// Read output - should be empty since not a real git repo
	readBuf := make([]byte, 1024)
	n, _ := r.Read(readBuf)
	output := string(readBuf[:n])

	// Should be empty or have no output
	assert.Empty(t, output)
}

func TestPrintBranches(t *testing.T) {
	// Since printBranches uses git commands, we test it indirectly
	// through the completion functions
	t.Run("function exists", func(t *testing.T) {
		// Ensure the function exists and can be called
		// Actual output testing requires git setup
		assert.NotPanics(t, func() {
			// Redirect stdout to avoid noise
			oldStdout := os.Stdout
			os.Stdout = os.NewFile(0, os.DevNull)
			defer func() { os.Stdout = oldStdout }()

			printBranches(os.Stdout)
		})
	})
}

func TestPrintWorktrees(t *testing.T) {
	// Since printWorktrees uses git commands, we test it indirectly
	// through the completion functions
	t.Run("function exists", func(t *testing.T) {
		// Ensure the function exists and can be called
		// Actual output testing requires git setup
		assert.NotPanics(t, func() {
			// Redirect stdout to avoid noise
			oldStdout := os.Stdout
			os.Stdout = os.NewFile(0, os.DevNull)
			defer func() { os.Stdout = oldStdout }()

			printWorktrees(os.Stdout)
		})
	})
}

func TestCompletionScriptIntegration(t *testing.T) {
	// Verify completion scripts are generated correctly for each shell
	shells := []string{"bash", "zsh"}

	for _, shell := range shells {
		t.Run(shell, func(t *testing.T) {
			// Capture stdout instead of using Writer
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			app := &cli.Command{
				Commands: []*cli.Command{
					NewCompletionCommand(),
				},
			}

			ctx := context.Background()
			err := app.Run(ctx, []string{"wtp", "completion", shell})

			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout
			buf := make([]byte, 65536) // Larger buffer for completion scripts
			n, _ := r.Read(buf)
			output := string(buf[:n])

			assert.NoError(t, err)

			// Common content that should be included
			assert.Contains(t, output, "wtp")
			assert.Contains(t, output, "cd")

			// Check for shell-specific syntax
			switch shell {
			case "bash":
				assert.Contains(t, output, "complete -F")
			case "zsh":
				assert.Contains(t, output, "#compdef wtp")
			}
		})
	}
}

func TestHiddenCommands(t *testing.T) {
	cmd := NewCompletionCommand()

	// __branches and __worktrees should be hidden commands
	for _, sub := range cmd.Commands {
		if strings.HasPrefix(sub.Name, "__") {
			assert.True(t, sub.Hidden, "%s should be hidden", sub.Name)
		}
	}
}

// Helper function
func findSubcommand(cmd *cli.Command, name string) *cli.Command {
	for _, sub := range cmd.Commands {
		if sub.Name == name {
			return sub
		}
	}
	return nil
}
