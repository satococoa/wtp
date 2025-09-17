package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

func TestNewShellInitCommand(t *testing.T) {
	cmd := NewShellInitCommand()

	assert.NotNil(t, cmd)
	assert.Equal(t, "shell-init", cmd.Name)
	assert.Equal(t, "Initialize shell with completion and cd functionality", cmd.Usage)
	assert.NotEmpty(t, cmd.Description)

	// Check subcommands
	subcommands := make(map[string]*cli.Command)
	for _, sub := range cmd.Commands {
		subcommands[sub.Name] = sub
	}

	// Verify required shells are supported
	supportedShells := []string{"bash", "zsh", "fish"}
	for _, shell := range supportedShells {
		assert.Contains(t, subcommands, shell, "Shell-init command must support %s", shell)
		assert.NotNil(t, subcommands[shell].Action)
	}
}

func TestShellInitCommand_OutputsValidScripts(t *testing.T) {
	// Note: These tests can't easily verify the actual output without
	// executing the wtp binary, which would create a circular dependency.
	oldRunCompletion := runCompletionCommand
	runCompletionCommand = func(shell string) ([]byte, error) {
		return []byte("completion-" + shell), nil
	}
	t.Cleanup(func() {
		runCompletionCommand = oldRunCompletion
	})

	tests := []struct {
		name  string
		shell string
	}{
		{
			name:  "bash generates without error",
			shell: "bash",
		},
		{
			name:  "zsh generates without error",
			shell: "zsh",
		},
		{
			name:  "fish generates without error",
			shell: "fish",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			app := &cli.Command{
				Commands: []*cli.Command{
					NewShellInitCommand(),
				},
				Writer: &buf,
			}

			ctx := context.Background()
			err := app.Run(ctx, []string{"wtp", "shell-init", tt.shell})
			assert.NoError(t, err)
			assert.Contains(t, buf.String(), "completion-"+tt.shell)
		})
	}
}
