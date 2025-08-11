package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

func TestShellBash(t *testing.T) {
	app := &cli.Command{
		Commands: []*cli.Command{
			NewShellCommand(),
		},
	}

	var buf bytes.Buffer
	app.Writer = &buf

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "shell", "bash"})
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "#!/bin/bash")
	assert.Contains(t, output, "_wtp_completion")
	assert.Contains(t, output, "complete -F _wtp_completion wtp")
	assert.Contains(t, output, "wtp shell integration")
	assert.Contains(t, output, "wtp() {")
	assert.Contains(t, output, "WTP_CD_FILE")
}

func TestShellZsh(t *testing.T) {
	app := &cli.Command{
		Commands: []*cli.Command{
			NewShellCommand(),
		},
	}

	var buf bytes.Buffer
	app.Writer = &buf

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "shell", "zsh"})
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "#compdef wtp")
	assert.Contains(t, output, "_wtp()")
	assert.Contains(t, output, "compdef _wtp wtp")
	assert.Contains(t, output, "wtp shell integration")
	assert.Contains(t, output, "wtp() {")
	assert.Contains(t, output, "WTP_CD_FILE")
}

func TestShellFish(t *testing.T) {
	app := &cli.Command{
		Commands: []*cli.Command{
			NewShellCommand(),
		},
	}

	var buf bytes.Buffer
	app.Writer = &buf

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "shell", "fish"})
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "wtp")
	assert.Contains(t, output, "complete")
	assert.Contains(t, output, "wtp shell integration")
	assert.Contains(t, output, "function wtp")
	assert.Contains(t, output, "WTP_CD_FILE")
}

func TestShellHelp(t *testing.T) {
	app := &cli.Command{
		Commands: []*cli.Command{
			NewShellCommand(),
		},
	}

	var buf bytes.Buffer
	app.Writer = &buf

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "shell", "--help"})
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "shell integration script")
	assert.Contains(t, output, "tab completion and cd functionality")
	assert.Contains(t, output, "bash")
	assert.Contains(t, output, "zsh")
	assert.Contains(t, output, "fish")
}
