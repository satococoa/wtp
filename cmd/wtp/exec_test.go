package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"

	"github.com/satococoa/wtp/v2/internal/command"
)

func TestNewExecCommand(t *testing.T) {
	cmd := NewExecCommand()
	assert.Equal(t, "exec", cmd.Name)
	assert.NotNil(t, cmd.Action)
	assert.NotNil(t, cmd.ShellComplete)
}

func TestParseExecInput(t *testing.T) {
	t.Run("valid input", func(t *testing.T) {
		worktree, name, args, err := parseExecInput([]string{"feature/auth", "--", "go", "test", "./..."})
		require.NoError(t, err)
		assert.Equal(t, "feature/auth", worktree)
		assert.Equal(t, "go", name)
		assert.Equal(t, []string{"test", "./..."}, args)
	})

	t.Run("without separator should still parse", func(t *testing.T) {
		worktree, name, args, err := parseExecInput([]string{"feature/auth", "go", "test"})
		require.NoError(t, err)
		assert.Equal(t, "feature/auth", worktree)
		assert.Equal(t, "go", name)
		assert.Equal(t, []string{"test"}, args)
	})

	t.Run("missing command", func(t *testing.T) {
		_, _, _, err := parseExecInput([]string{"feature/auth", "--"})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "command is required")
	})

	t.Run("missing worktree", func(t *testing.T) {
		_, _, _, err := parseExecInput([]string{})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "worktree name is required")
	})
}

func TestExecCommandWithCommandExecutor(t *testing.T) {
	t.Run("execute command in resolved worktree", func(t *testing.T) {
		cmd := createExecTestCLICommand([]string{"feature/auth", "--", "pwd"})
		var buf bytes.Buffer
		mock := &mockExecCommandExecutor{
			results: []*command.ExecutionResult{
				{
					Results: []command.Result{{
						Output: `worktree /repo/main
HEAD abc
branch refs/heads/main

worktree /repo/worktrees/feature/auth
HEAD def
branch refs/heads/feature/auth
`,
					}},
				},
				{
					Results: []command.Result{{
						Output: "/repo/worktrees/feature/auth",
					}},
				},
			},
		}

		err := execCommandWithCommandExecutor(cmd, &buf, mock)
		require.NoError(t, err)
		require.Len(t, mock.executed, 2)
		require.Len(t, mock.executed[1], 1)
		assert.Equal(t, "pwd", mock.executed[1][0].Name)
		assert.Equal(t, "/repo/worktrees/feature/auth", mock.executed[1][0].WorkDir)
		assert.True(t, mock.executed[1][0].Interactive)
		assert.Contains(t, buf.String(), "/repo/worktrees/feature/auth")
	})

	t.Run("command failure returns error", func(t *testing.T) {
		cmd := createExecTestCLICommand([]string{"@", "--", "false"})
		mock := &mockExecCommandExecutor{
			results: []*command.ExecutionResult{
				{
					Results: []command.Result{{
						Output: "worktree /repo/main\nHEAD abc\nbranch refs/heads/main\n",
					}},
				},
				{
					Results: []command.Result{{
						Error: assert.AnError,
					}},
				},
			},
		}

		err := execCommandWithCommandExecutor(cmd, &bytes.Buffer{}, mock)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "command failed in worktree")
	})

	t.Run("git worktree list with empty result returns git error", func(t *testing.T) {
		cmd := createExecTestCLICommand([]string{"@", "--", "pwd"})
		mock := &mockExecCommandExecutor{
			results: []*command.ExecutionResult{{}},
		}

		err := execCommandWithCommandExecutor(cmd, &bytes.Buffer{}, mock)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "git command failed: git worktree list")
		assert.Contains(t, err.Error(), "no command results")
	})

	t.Run("git worktree list result error returns git error", func(t *testing.T) {
		cmd := createExecTestCLICommand([]string{"@", "--", "pwd"})
		mock := &mockExecCommandExecutor{
			results: []*command.ExecutionResult{
				{
					Results: []command.Result{{
						Output: "fatal output",
						Error:  assert.AnError,
					}},
				},
			},
		}

		err := execCommandWithCommandExecutor(cmd, &bytes.Buffer{}, mock)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "git command failed: git worktree list")
		assert.Contains(t, err.Error(), "fatal output")
	})
}

func TestCompleteWorktreesForExec(t *testing.T) {
	t.Run("should not panic when command args already started", func(t *testing.T) {
		cmd := createExecTestCLICommand([]string{"feature/auth", "go"})

		assert.NotPanics(t, func() {
			restore := silenceStdout(t)
			defer restore()

			completeWorktreesForExec(context.Background(), cmd)
		})
	})
}

func createExecTestCLICommand(args []string) *cli.Command {
	app := &cli.Command{
		Name: "test",
		Commands: []*cli.Command{
			{
				Name:   "exec",
				Action: func(_ context.Context, _ *cli.Command) error { return nil },
			},
		},
	}

	cmdArgs := append([]string{"test", "exec"}, args...)
	_ = app.Run(context.Background(), cmdArgs)
	return app.Commands[0]
}

type mockExecCommandExecutor struct {
	executed [][]command.Command
	results  []*command.ExecutionResult
}

func (m *mockExecCommandExecutor) Execute(commands []command.Command) (*command.ExecutionResult, error) {
	m.executed = append(m.executed, commands)

	idx := len(m.executed) - 1
	if idx < len(m.results) {
		return m.results[idx], nil
	}

	return &command.ExecutionResult{}, nil
}
