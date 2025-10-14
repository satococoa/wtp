package main

import (
	"bytes"
	"io"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestCompleteFlagSuggestions_MatchesLongFlag(t *testing.T) {
	var buf bytes.Buffer

	cmd := &cli.Command{
		Writer: &buf,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name: "with-branch",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
			},
			cli.GenerateShellCompletionFlag,
		},
	}

	require.True(t, completeFlagSuggestions(cmd, "--w"))

	require.Contains(t, buf.String(), "--with-branch")
	require.NotContains(t, buf.String(), "--generate-shell-completion")
}

func TestCompleteFlagSuggestions_ShowsAllForSingleHyphen(t *testing.T) {
	var buf bytes.Buffer

	cmd := &cli.Command{
		Writer: &buf,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name: "with-branch",
			},
			&cli.BoolFlag{
				Name:    "force",
				Aliases: []string{"f"},
			},
		},
	}

	require.True(t, completeFlagSuggestions(cmd, "-"))

	output := buf.String()
	require.True(t, strings.Contains(output, "--with-branch") || strings.Contains(output, "-with-branch"))
	require.True(t, strings.Contains(output, "--force") || strings.Contains(output, "-force"))
}

func TestMaybeCompleteFlagSuggestions_IgnoresPreviousWhenCurrentEmpty(t *testing.T) {
	cmd := &cli.Command{
		Writer: io.Discard,
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "force"},
			cli.GenerateShellCompletionFlag,
		},
	}

	require.False(t, maybeCompleteFlagSuggestions(cmd, "", []string{"--force"}))
}
