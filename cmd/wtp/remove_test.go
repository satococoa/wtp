package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/satococoa/wtp/internal/command"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

// ===== Simple Unit Tests (What testing) =====

func TestNewRemoveCommand(t *testing.T) {
	cmd := NewRemoveCommand()

	assert.NotNil(t, cmd)
	assert.Equal(t, "remove", cmd.Name)
	assert.Equal(t, "Remove a worktree", cmd.Usage)
	assert.Equal(t, "wtp remove <worktree-name>", cmd.UsageText)
	assert.NotEmpty(t, cmd.Description)
	assert.NotNil(t, cmd.Action)
	assert.NotNil(t, cmd.ShellComplete)

	// Verify flags
	flagNames := make(map[string]bool)
	for _, flag := range cmd.Flags {
		flagNames[flag.Names()[0]] = true
	}

	assert.True(t, flagNames["force"], "force flag should exist")
	assert.True(t, flagNames["with-branch"], "with-branch flag should exist")
	assert.True(t, flagNames["force-branch"], "force-branch flag should exist")
}

func TestRemoveCommand_NoBranchName(t *testing.T) {
	app := &cli.Command{
		Commands: []*cli.Command{
			NewRemoveCommand(),
		},
	}

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "remove"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch name is required")
}

func TestRemoveCommand_ForceBranchWithoutWithBranch(t *testing.T) {
	app := &cli.Command{
		Commands: []*cli.Command{
			NewRemoveCommand(),
		},
	}

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "remove", "--force-branch", "feature/test"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "--force-branch requires --with-branch")
}

func TestRemoveCommand_NotInGitRepo(t *testing.T) {
	// Create a temporary directory that is not a git repo
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	err := os.Chdir(tempDir)
	assert.NoError(t, err)

	app := &cli.Command{
		Commands: []*cli.Command{
			NewRemoveCommand(),
		},
	}

	ctx := context.Background()
	err = app.Run(ctx, []string{"wtp", "remove", "feature/test"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in a git repository")
}

func TestRemoveCommand_DirectoryAccessError(t *testing.T) {
	// Save original removeGetwd to restore later
	originalGetwd := removeGetwd
	defer func() { removeGetwd = originalGetwd }()

	// Mock removeGetwd to return an error
	removeGetwd = func() (string, error) {
		return "", assert.AnError
	}

	app := &cli.Command{
		Commands: []*cli.Command{
			NewRemoveCommand(),
		},
	}

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "remove", "feature/test"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to access")
}

func TestRemoveCommand_WorktreeNotFound(t *testing.T) {
	// Create a temporary git repository
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	err := os.MkdirAll(gitDir, 0755)
	assert.NoError(t, err)

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	err = os.Chdir(tempDir)
	assert.NoError(t, err)

	app := &cli.Command{
		Commands: []*cli.Command{
			NewRemoveCommand(),
		},
	}

	ctx := context.Background()
	err = app.Run(ctx, []string{"wtp", "remove", "non-existent-branch"})

	// Either error about git not being initialized or worktree not found
	assert.Error(t, err)
}

func TestRemoveCommand_Flags(t *testing.T) {
	cmd := NewRemoveCommand()

	// Test force flag
	forceFlag := findFlag(cmd, "force")
	assert.NotNil(t, forceFlag)
	boolFlag, ok := forceFlag.(*cli.BoolFlag)
	assert.True(t, ok)
	assert.Contains(t, boolFlag.Aliases, "f")

	// Test with-branch flag
	withBranchFlag := findFlag(cmd, "with-branch")
	assert.NotNil(t, withBranchFlag)

	// Test force-branch flag
	forceBranchFlag := findFlag(cmd, "force-branch")
	assert.NotNil(t, forceBranchFlag)
}

// Helper function to find a flag by name
func findFlag(cmd *cli.Command, name string) cli.Flag {
	for _, flag := range cmd.Flags {
		if flag.Names()[0] == name {
			return flag
		}
	}
	return nil
}

func TestRemoveCommand_ShellComplete(t *testing.T) {
	cmd := NewRemoveCommand()
	assert.NotNil(t, cmd.ShellComplete)

	// Test that shell complete function exists and can be called
	ctx := context.Background()
	cliCmd := &cli.Command{}

	// ShellComplete returns nothing, just test it doesn't panic
	assert.NotPanics(t, func() {
		cmd.ShellComplete(ctx, cliCmd)
	})
}

func TestRemoveCommand_Success(t *testing.T) {
	mockExec := &mockRemoveCommandExecutor{
		results: []command.Result{
			{
				Output: "worktree /path/to/worktrees/feature-auth\nHEAD abc123\nbranch refs/heads/feature-auth\n\n",
				Error:  nil,
			},
			{
				Output: "",
				Error:  nil,
			},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	err := removeCommandWithCommandExecutor(cmd, &buf, mockExec, "/repo", "feature-auth", false, false, false)

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Removed worktree 'feature-auth'")
	assert.Len(t, mockExec.executedCommands, 2)
	assert.Equal(t, []string{"worktree", "list", "--porcelain"}, mockExec.executedCommands[0].Args)
	assert.Equal(t, []string{"worktree", "remove", "/path/to/worktrees/feature-auth"}, mockExec.executedCommands[1].Args)
}

// ===== Mock Implementations =====

type mockRemoveCommandExecutor struct {
	executedCommands []command.Command
	results          []command.Result
}

func (m *mockRemoveCommandExecutor) Execute(commands []command.Command) (*command.ExecutionResult, error) {
	m.executedCommands = append(m.executedCommands, commands...)

	results := make([]command.Result, len(commands))
	for i, cmd := range commands {
		if i < len(m.results) {
			results[i] = m.results[i]
		} else {
			results[i] = command.Result{
				Command: cmd,
				Output:  "",
				Error:   nil,
			}
		}
	}

	return &command.ExecutionResult{Results: results}, nil
}

func TestRemoveCommand_WithBranch(t *testing.T) {
	mockExec := &mockRemoveCommandExecutor{
		results: []command.Result{
			{
				Output: "worktree /path/to/worktrees/feature-auth\nHEAD abc123\nbranch refs/heads/feature-auth\n\n",
				Error:  nil,
			},
			{
				Output: "",
				Error:  nil,
			},
			{
				Output: "",
				Error:  nil,
			},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	err := removeCommandWithCommandExecutor(cmd, &buf, mockExec, "/repo", "feature-auth", false, true, false)

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Removed worktree 'feature-auth'")
	assert.Contains(t, buf.String(), "Removed branch 'feature-auth'")
	assert.Len(t, mockExec.executedCommands, 3)
	assert.Equal(t, []string{"branch", "-d", "feature-auth"}, mockExec.executedCommands[2].Args)
}

func TestRemoveCommand_ForceBranch(t *testing.T) {
	mockExec := &mockRemoveCommandExecutor{
		results: []command.Result{
			{
				Output: "worktree /path/to/worktrees/feature-auth\nHEAD abc123\nbranch refs/heads/feature-auth\n\n",
				Error:  nil,
			},
			{
				Output: "",
				Error:  nil,
			},
			{
				Output: "",
				Error:  nil,
			},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	err := removeCommandWithCommandExecutor(cmd, &buf, mockExec, "/repo", "feature-auth", false, true, true)

	assert.NoError(t, err)
	assert.Equal(t, []string{"branch", "-D", "feature-auth"}, mockExec.executedCommands[2].Args)
}

func TestRemoveCommand_NotFound(t *testing.T) {
	mockExec := &mockRemoveCommandExecutor{
		results: []command.Result{
			{
				Output: "worktree /path/to/worktrees/main\nHEAD abc123\nbranch refs/heads/main\n\n",
				Error:  nil,
			},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	err := removeCommandWithCommandExecutor(cmd, &buf, mockExec, "/repo", "feature-auth", false, false, false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worktree 'feature-auth' not found")
}
