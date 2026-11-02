package main

import (
	"bytes"
	"io"
	"os"
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

func TestMaybeCompleteFlagSuggestions_IgnoresSentinelInPrevious(t *testing.T) {
	cmd := &cli.Command{
		Writer: io.Discard,
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "force"},
			cli.GenerateShellCompletionFlag,
		},
	}

	require.False(t, maybeCompleteFlagSuggestions(cmd, "feature", []string{"-"}))
	require.False(t, maybeCompleteFlagSuggestions(cmd, "", []string{"-"}))
}

func TestFlagCandidateFromOSArgsSentinel(t *testing.T) {
	original := os.Args
	t.Cleanup(func() { os.Args = original })

	os.Args = []string{"wtp", "remove", "target", "-", "--generate-shell-completion"}
	candidate, ok := flagCandidateFromOSArgs()
	require.True(t, ok)
	require.Equal(t, "target", candidate)

	os.Args = []string{"wtp", "remove", "target", "--", "--generate-shell-completion"}
	candidate, ok = flagCandidateFromOSArgs()
	require.True(t, ok)
	require.Equal(t, "target", candidate)

	os.Args = []string{"wtp", "remove", "target", "-", "--generate-shell-completion", "--generate-shell-completion"}
	candidate, ok = flagCandidateFromOSArgs()
	require.True(t, ok)
	require.Equal(t, "target", candidate)
}

func TestMaybeCompleteFlagSuggestions_UsesOSArgsWhenCurrentEmpty(t *testing.T) {
	original := os.Args
	t.Cleanup(func() { os.Args = original })

	os.Args = []string{"wtp", "remove", "--w", "--generate-shell-completion"}

	var buf bytes.Buffer
	cmd := &cli.Command{
		Writer: &buf,
		Flags: []cli.Flag{
			&cli.BoolFlag{Name: "with-branch"},
			&cli.BoolFlag{Name: "force"},
			cli.GenerateShellCompletionFlag,
		},
	}

	require.True(t, maybeCompleteFlagSuggestions(cmd, "", nil))
	require.Contains(t, buf.String(), "--with-branch")
}

func TestIsCompleteFlagName_ShortAlias(t *testing.T) {
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "branch", Aliases: []string{"b"}},
			&cli.BoolFlag{Name: "quiet", Aliases: []string{"q"}},
			&cli.StringFlag{Name: "exec"},
		},
	}

	require.True(t, isCompleteFlagName(cmd, "-b"))
	require.True(t, isCompleteFlagName(cmd, "-q"))
}

func TestIsCompleteFlagName_LongName(t *testing.T) {
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "branch", Aliases: []string{"b"}},
			&cli.BoolFlag{Name: "quiet", Aliases: []string{"q"}},
			&cli.StringFlag{Name: "exec"},
		},
	}

	require.True(t, isCompleteFlagName(cmd, "--branch"))
	require.True(t, isCompleteFlagName(cmd, "--quiet"))
	require.True(t, isCompleteFlagName(cmd, "--exec"))
}

func TestIsCompleteFlagName_DoubleDashSkipsSingleCharAlias(t *testing.T) {
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "branch", Aliases: []string{"b"}},
			&cli.BoolFlag{Name: "quiet", Aliases: []string{"q"}},
		},
	}

	require.False(t, isCompleteFlagName(cmd, "--b"))
	require.False(t, isCompleteFlagName(cmd, "--q"))
}

func TestIsCompleteFlagName_PartialName(t *testing.T) {
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "branch", Aliases: []string{"b"}},
			&cli.BoolFlag{Name: "force", Aliases: []string{"f"}},
		},
	}

	require.False(t, isCompleteFlagName(cmd, "--br"))
	require.False(t, isCompleteFlagName(cmd, "--fo"))
	require.False(t, isCompleteFlagName(cmd, "-fo"))
}

func TestIsCompleteFlagName_NonFlagOrEdgeCases(t *testing.T) {
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "branch", Aliases: []string{"b"}},
		},
	}

	require.False(t, isCompleteFlagName(cmd, "foo"))
	require.False(t, isCompleteFlagName(cmd, "-"))
	require.False(t, isCompleteFlagName(cmd, "--"))
	require.False(t, isCompleteFlagName(cmd, ""))
	require.False(t, isCompleteFlagName(cmd, "--unknown"))
}

func TestIsCompleteFlagName_EqualsFormat(t *testing.T) {
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "branch", Aliases: []string{"b"}},
		},
	}

	require.True(t, isCompleteFlagName(cmd, "--branch=value"))
	require.True(t, isCompleteFlagName(cmd, "-b=value"))
}

func TestIsCompleteFlagName_SingleDashFullName(t *testing.T) {
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "branch", Aliases: []string{"b"}},
		},
	}

	require.True(t, isCompleteFlagName(cmd, "-branch"))
}

func TestMaybeCompleteFlagSuggestions_SkipsCompleteShortFlag(t *testing.T) {
	original := os.Args
	t.Cleanup(func() { os.Args = original })

	os.Args = []string{"wtp", "add", "-b", "--generate-shell-completion"}

	var buf bytes.Buffer
	cmd := &cli.Command{
		Writer: &buf,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "branch", Aliases: []string{"b"}},
			&cli.BoolFlag{Name: "quiet", Aliases: []string{"q"}},
			&cli.StringFlag{Name: "exec"},
			cli.GenerateShellCompletionFlag,
		},
	}

	require.False(t, maybeCompleteFlagSuggestions(cmd, "", nil))
	require.Empty(t, buf.String())
}

func TestMaybeCompleteFlagSuggestions_SkipsCompleteLongFlag(t *testing.T) {
	original := os.Args
	t.Cleanup(func() { os.Args = original })

	os.Args = []string{"wtp", "add", "--branch", "--generate-shell-completion"}

	var buf bytes.Buffer
	cmd := &cli.Command{
		Writer: &buf,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "branch", Aliases: []string{"b"}},
			cli.GenerateShellCompletionFlag,
		},
	}

	require.False(t, maybeCompleteFlagSuggestions(cmd, "", nil))
	require.Empty(t, buf.String())
}

func TestMaybeCompleteFlagSuggestions_SkipsCompleteBoolFlag(t *testing.T) {
	original := os.Args
	t.Cleanup(func() { os.Args = original })

	os.Args = []string{"wtp", "add", "-q", "--generate-shell-completion"}

	var buf bytes.Buffer
	cmd := &cli.Command{
		Writer: &buf,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "branch", Aliases: []string{"b"}},
			&cli.BoolFlag{Name: "quiet", Aliases: []string{"q"}},
			cli.GenerateShellCompletionFlag,
		},
	}

	require.False(t, maybeCompleteFlagSuggestions(cmd, "", nil))
	require.Empty(t, buf.String())
}

func TestMaybeCompleteFlagSuggestions_PartialFlagStillCompletes(t *testing.T) {
	original := os.Args
	t.Cleanup(func() { os.Args = original })

	os.Args = []string{"wtp", "add", "--b", "--generate-shell-completion"}

	var buf bytes.Buffer
	cmd := &cli.Command{
		Writer: &buf,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "branch", Aliases: []string{"b"}},
			cli.GenerateShellCompletionFlag,
		},
	}

	require.True(t, maybeCompleteFlagSuggestions(cmd, "", nil))
	require.Contains(t, buf.String(), "--branch")
}

func TestMaybeCompleteFlagSuggestions_CompleteFlagWithCurrentNotEmpty(t *testing.T) {
	original := os.Args
	t.Cleanup(func() { os.Args = original })

	os.Args = []string{"wtp", "add", "foo", "-b", "--generate-shell-completion"}

	var buf bytes.Buffer
	cmd := &cli.Command{
		Writer: &buf,
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "branch", Aliases: []string{"b"}},
			cli.GenerateShellCompletionFlag,
		},
	}

	require.True(t, maybeCompleteFlagSuggestions(cmd, "foo", nil))
	require.Contains(t, buf.String(), "--branch")
}
