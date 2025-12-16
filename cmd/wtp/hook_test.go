package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

// Helper function
func findSubcommand(cmd *cli.Command, name string) *cli.Command {
	for _, sub := range cmd.Commands {
		if sub.Name == name {
			return sub
		}
	}
	return nil
}

// Focus on what matters: command behavior, not structure
func TestNewHookCommand_SupportedShells(t *testing.T) {
	cmd := NewHookCommand()
	assert.Equal(t, "hook", cmd.Name)

	// What matters: all required shells are supported
	supportedShells := []string{"bash", "zsh", "fish"}
	for _, shell := range supportedShells {
		subCmd := findSubcommand(cmd, shell)
		assert.NotNil(t, subCmd, "Hook command must support %s", shell)
	}
}

func TestHookCommand_GeneratesValidShellScripts(t *testing.T) {
	tests := []struct {
		name     string
		shell    string
		contains []string
	}{
		{
			name:  "bash generates valid hook",
			shell: "bash",
			contains: []string{
				"wtp()",
				"if [[ \"$1\" == \"cd\" ]]",
				"command wtp cd",
				"cd \"$target_dir\"",
			},
		},
		{
			name:  "zsh generates valid hook",
			shell: "zsh",
			contains: []string{
				"wtp()",
				"if [[ \"$1\" == \"cd\" ]]",
				"command wtp cd",
				"cd \"$target_dir\"",
			},
		},
		{
			name:  "fish generates valid hook",
			shell: "fish",
			contains: []string{
				"function wtp",
				"if test \"$argv[1]\" = \"cd\"",
				"command wtp cd",
				"cd \"$target_dir\"",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &cli.Command{
				Commands: []*cli.Command{
					NewHookCommand(),
				},
			}

			var buf bytes.Buffer
			app.Writer = &buf

			ctx := context.Background()
			err := app.Run(ctx, []string{"wtp", "hook", tt.shell})
			assert.NoError(t, err)

			output := buf.String()
			assert.NotEmpty(t, output, "Hook script should not be empty")

			// Essential behavior: script contains required elements
			for _, expected := range tt.contains {
				assert.Contains(t, output, expected)
			}

			// Essential behavior: no legacy environment variable dependency
			assert.NotContains(t, output, "WTP_SHELL_INTEGRATION")
		})
	}
}

// Test the core business logic that matters most
func TestHookScripts_HandleEdgeCases(t *testing.T) {
	tests := []struct {
		name          string
		shell         string
		requiredLogic []string
		notContains   []string
	}{
		{
			name:  "bash hook supports no-arg cd",
			shell: "bash",
			requiredLogic: []string{
				"if [[ -z \"$2\" ]]",              // No-arg branch
				"target_dir=$(command wtp cd",      // Uses `wtp cd` default behavior
				"target_dir=$(command wtp cd \"$2", // Uses explicit worktree name when present
			},
			notContains: []string{
				"Usage: wtp cd <worktree>",
				"echo \"Usage:",
			},
		},
		{
			name:  "fish hook supports no-arg cd",
			shell: "fish",
			requiredLogic: []string{
				"if test -z \"$argv[2]\"",           // No-arg branch
				"set -l target_dir (command wtp cd",  // Uses `wtp cd` default behavior
				"command wtp cd $argv[2]",            // Uses explicit worktree name when present
				"cd \"$target_dir\"",                 // Handles spaces safely
			},
			notContains: []string{
				"Usage: wtp cd <worktree>",
				"echo \"Usage:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer

			switch tt.shell {
			case "bash":
				require.NoError(t, printBashHook(&buf))
			case "fish":
				require.NoError(t, printFishHook(&buf))
			}

			output := buf.String()
			for _, logic := range tt.requiredLogic {
				assert.Contains(t, output, logic, "Hook must handle edge cases properly")
			}
			for _, unexpected := range tt.notContains {
				assert.NotContains(t, output, unexpected)
			}
		})
	}
}
