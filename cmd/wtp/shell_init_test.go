package main

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

// Test the core value proposition: shell-init combines completion and hook
func TestShellInit_CombinesBothFeatures(t *testing.T) {
	tests := []struct {
		shell             string
		completionMarkers []string
		hookMarkers       []string
	}{
		{
			shell:             "bash",
			completionMarkers: []string{"#!/bin/bash", "_wtp_completion", "complete -F"},
			hookMarkers:       []string{"# wtp cd command hook for bash", "wtp()", "command wtp cd"},
		},
		{
			shell:             "zsh",
			completionMarkers: []string{"#compdef wtp", "_wtp", "compdef _wtp wtp"},
			hookMarkers:       []string{"# wtp cd command hook for zsh", "wtp()", "command wtp cd"},
		},
		{
			shell:             "fish",
			completionMarkers: []string{"complete", "wtp"},
			hookMarkers:       []string{"# wtp cd command hook for fish", "function wtp", "command wtp cd"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.shell, func(t *testing.T) {
			app := &cli.Command{
				Commands: []*cli.Command{NewShellInitCommand()},
			}

			var buf bytes.Buffer
			app.Writer = &buf

			err := app.Run(context.Background(), []string{"wtp", "shell-init", tt.shell})
			assert.NoError(t, err)

			output := buf.String()
			assert.NotEmpty(t, output, "Shell init should produce output")

			// Verify completion functionality is present
			for _, marker := range tt.completionMarkers {
				assert.Contains(t, output, marker, "Must include completion functionality")
			}

			// Verify hook functionality is present
			for _, marker := range tt.hookMarkers {
				assert.Contains(t, output, marker, "Must include hook functionality")
			}

			// Critical: No legacy dependencies
			assert.NotContains(t, output, "WTP_SHELL_INTEGRATION",
				"Shell init must not depend on legacy environment variables")
		})
	}
}

// Test the integration contract: hook should appear after completion
func TestShellInit_CorrectOrderAndSeparation(t *testing.T) {
	app := &cli.Command{
		Commands: []*cli.Command{NewShellInitCommand()},
	}

	var buf bytes.Buffer
	app.Writer = &buf

	err := app.Run(context.Background(), []string{"wtp", "shell-init", "bash"})
	assert.NoError(t, err)

	output := buf.String()

	// Find positions of key markers
	completionPos := strings.Index(output, "_wtp_completion")
	hookPos := strings.Index(output, "# wtp cd command hook")

	assert.Greater(t, completionPos, -1, "Completion section must be present")
	assert.Greater(t, hookPos, -1, "Hook section must be present")
	assert.Less(t, completionPos, hookPos, "Completion must come before hook")

	// Verify clear separation with newline
	lines := strings.Split(output, "\n")
	assert.Greater(t, len(lines), 10, "Should have substantial content")
}

// Minimal structure test - only test what affects user behavior
func TestShellInitCommand_SupportedShells(t *testing.T) {
	cmd := NewShellInitCommand()
	assert.Equal(t, "shell-init", cmd.Name)

	// Only test what matters: supported shells
	supportedShells := []string{"bash", "zsh", "fish"}
	for _, shell := range supportedShells {
		subCmd := findSubcommand(cmd, shell)
		assert.NotNil(t, subCmd, "Shell-init must support %s", shell)
	}
}
