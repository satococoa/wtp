package main

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

func TestNewRemoveCommand(t *testing.T) {
	cmd := NewRemoveCommand()
	
	assert.NotNil(t, cmd)
	assert.Equal(t, "remove", cmd.Name)
	assert.Equal(t, "Remove a worktree", cmd.Usage)
	assert.Equal(t, "wtp remove <branch-name>", cmd.UsageText)
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
	defer os.Chdir(oldDir)
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
	defer os.Chdir(oldDir)
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