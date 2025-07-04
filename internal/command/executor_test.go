package command

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// Test that defines what we want from CommandExecutor
func TestCommandExecutor_Interface(t *testing.T) {
	t.Run("should execute single command", func(t *testing.T) {
		// Given: a command executor with mock shell executor
		mockShell := &mockShellExecutor{}
		executor := NewCommandExecutor(mockShell)

		// When: executing a single git command
		cmd := Command{
			Name: "git",
			Args: []string{"worktree", "add", "../worktrees/feature", "feature"},
		}
		result, err := executor.Execute([]Command{cmd})

		// Then: command should be executed successfully
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Results, 1)
		assert.Equal(t, cmd, result.Results[0].Command)
		assert.Empty(t, result.Results[0].Error)
	})

	t.Run("should execute multiple commands in sequence", func(t *testing.T) {
		// Given: a command executor
		mockShell := &mockShellExecutor{}
		executor := NewCommandExecutor(mockShell)

		// When: executing multiple commands
		commands := []Command{
			{Name: "git", Args: []string{"worktree", "add", "../worktrees/feature", "feature"}},
			{Name: "git", Args: []string{"branch", "-D", "old-feature"}},
		}
		result, err := executor.Execute(commands)

		// Then: all commands should be executed in order
		assert.NoError(t, err)
		assert.Len(t, result.Results, 2)
		assert.Equal(t, commands[0], result.Results[0].Command)
		assert.Equal(t, commands[1], result.Results[1].Command)
	})

	t.Run("should handle command failure", func(t *testing.T) {
		// Given: a command executor that will fail
		mockShell := &mockShellExecutor{
			shouldFail: true,
			failOutput: "fatal: branch not found",
		}
		executor := NewCommandExecutor(mockShell)

		// When: executing a command that fails
		cmd := Command{
			Name: "git",
			Args: []string{"branch", "-D", "nonexistent"},
		}
		result, err := executor.Execute([]Command{cmd})

		// Then: error should be captured but not returned
		assert.NoError(t, err) // Execute itself doesn't fail
		assert.Len(t, result.Results, 1)
		assert.NotNil(t, result.Results[0].Error)
		assert.Contains(t, result.Results[0].Output, "fatal: branch not found")
	})

	t.Run("should support working directory", func(t *testing.T) {
		// Given: a command with specific working directory
		mockShell := &mockShellExecutor{}
		executor := NewCommandExecutor(mockShell)

		// When: executing command with WorkDir
		cmd := Command{
			Name:    "git",
			Args:    []string{"status"},
			WorkDir: "/path/to/repo",
		}
		result, err := executor.Execute([]Command{cmd})

		// Then: command should be executed in specified directory
		assert.NoError(t, err)
		assert.Len(t, result.Results, 1)
		assert.Equal(t, "/path/to/repo", mockShell.lastWorkDir)
	})

	t.Run("should handle empty command list", func(t *testing.T) {
		// Given: a command executor
		mockShell := &mockShellExecutor{}
		executor := NewCommandExecutor(mockShell)

		// When: executing empty command list
		result, err := executor.Execute([]Command{})

		// Then: should return empty result without error
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Results)
	})
}

// Test command builder functions
func TestCommandBuilder(t *testing.T) {
	t.Run("should build git worktree add command", func(t *testing.T) {
		// When: building a worktree add command
		cmd := GitWorktreeAdd("../worktrees/feature", "feature", GitWorktreeAddOptions{
			Force:  true,
			Branch: "new-feature",
		})

		// Then: command should have correct structure
		assert.Equal(t, "git", cmd.Name)
		assert.Equal(t, []string{"worktree", "add", "--force", "-b", "new-feature", "../worktrees/feature"}, cmd.Args)
	})

	t.Run("should build git branch delete command", func(t *testing.T) {
		// When: building a branch delete command
		cmd := GitBranchDelete("old-feature", false)

		// Then: command should have correct structure
		assert.Equal(t, "git", cmd.Name)
		assert.Equal(t, []string{"branch", "-d", "old-feature"}, cmd.Args)
	})

	t.Run("should build forced branch delete command", func(t *testing.T) {
		// When: building a forced branch delete command
		cmd := GitBranchDelete("old-feature", true)

		// Then: command should have correct structure
		assert.Equal(t, "git", cmd.Name)
		assert.Equal(t, []string{"branch", "-D", "old-feature"}, cmd.Args)
	})
}

// Mock implementation for testing
type mockShellExecutor struct {
	executedCommands []executedCommand
	shouldFail       bool
	failOutput       string
	lastWorkDir      string
}

type executedCommand struct {
	name    string
	args    []string
	workDir string
}

func (m *mockShellExecutor) Execute(name string, args []string, workDir string) (string, error) {
	m.executedCommands = append(m.executedCommands, executedCommand{
		name:    name,
		args:    args,
		workDir: workDir,
	})
	m.lastWorkDir = workDir

	if m.shouldFail {
		return m.failOutput, &mockError{msg: "command failed"}
	}
	return "success", nil
}

type mockError struct {
	msg string
}

func (e *mockError) Error() string {
	return e.msg
}