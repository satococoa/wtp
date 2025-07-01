package main

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

func TestCdCommand(t *testing.T) {
	// Test that it returns an error without shell integration
	app := &cli.Command{
		Commands: []*cli.Command{
			NewCdCommand(),
		},
	}

	var buf bytes.Buffer
	app.Writer = &buf
	err := app.Run(context.Background(), []string{"wtp", "cd", "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cd command requires shell integration")

	// Test cd without arguments - it will still show shell integration error
	buf.Reset()
	err = app.Run(context.Background(), []string{"wtp", "cd"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cd command requires shell integration")
}

func TestCdCommandDescription(t *testing.T) {
	cmd := NewCdCommand()
	assert.Equal(t, "cd", cmd.Name)
	assert.Equal(t, "Change directory to worktree (requires shell integration)", cmd.Usage)
	assert.Contains(t, cmd.Description, "shell integration")
	assert.Contains(t, cmd.Description, "Bash:")
	assert.Contains(t, cmd.Description, "Zsh:")
	assert.Contains(t, cmd.Description, "Fish:")
}

func TestShellInitWithCdFlag(t *testing.T) {
	// Save original environment
	originalShell := os.Getenv("SHELL")
	defer func() {
		os.Setenv("SHELL", originalShell)
	}()

	// Test with bash
	os.Setenv("SHELL", "/bin/bash")
	app := &cli.Command{
		Commands: []*cli.Command{
			NewShellInitCommand(),
		},
	}

	var buf bytes.Buffer
	app.Writer = &buf
	err := app.Run(context.Background(), []string{"wtp", "shell-init", "--cd"})
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "wtp cd command integration for bash")
	assert.Contains(t, output, "WTP_SHELL_INTEGRATION=1")
	assert.Contains(t, output, "complete -F _wtp_completion wtp")

	// Test with zsh
	os.Setenv("SHELL", "/bin/zsh")
	buf.Reset()
	err = app.Run(context.Background(), []string{"wtp", "shell-init", "--cd"})
	assert.NoError(t, err)
	output = buf.String()
	assert.Contains(t, output, "wtp cd command integration for zsh")
	assert.Contains(t, output, "WTP_SHELL_INTEGRATION=1")

	// Test with fish
	os.Setenv("SHELL", "/usr/bin/fish")
	buf.Reset()
	err = app.Run(context.Background(), []string{"wtp", "shell-init", "--cd"})
	assert.NoError(t, err)
	output = buf.String()
	assert.Contains(t, output, "wtp cd command integration for fish")
	assert.Contains(t, output, "WTP_SHELL_INTEGRATION=1")
}
