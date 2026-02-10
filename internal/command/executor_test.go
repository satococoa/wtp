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
		executor := NewExecutor(mockShell)

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
		executor := NewExecutor(mockShell)

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
		executor := NewExecutor(mockShell)

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
		executor := NewExecutor(mockShell)

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

	t.Run("should pass interactive mode to shell executor", func(t *testing.T) {
		mockShell := &mockShellExecutor{}
		executor := NewExecutor(mockShell)

		cmd := Command{
			Name:        "fzf",
			Interactive: true,
		}
		result, err := executor.Execute([]Command{cmd})

		assert.NoError(t, err)
		assert.Len(t, result.Results, 1)
		assert.True(t, mockShell.lastInteractive)
	})

	t.Run("should handle empty command list", func(t *testing.T) {
		// Given: a command executor
		mockShell := &mockShellExecutor{}
		executor := NewExecutor(mockShell)

		// When: executing empty command list
		result, err := executor.Execute([]Command{})

		// Then: should return empty result without error
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result.Results)
	})
}

// Test helper functions
func TestExtractBranchName(t *testing.T) {
	t.Run("should extract branch name from remote reference", func(t *testing.T) {
		// Given: a remote reference like "origin/feature"
		ref := "origin/feature"

		// When: extracting the branch name
		result := extractBranchName(ref)

		// Then: should return the branch part only
		assert.Equal(t, "feature", result)
	})

	t.Run("should extract branch name from nested reference", func(t *testing.T) {
		// Given: a nested remote reference
		ref := "origin/feature/awesome-feature"

		// When: extracting the branch name
		result := extractBranchName(ref)

		// Then: should return everything after the last slash
		assert.Equal(t, "awesome-feature", result)
	})

	t.Run("should return unchanged when no slash exists", func(t *testing.T) {
		// Given: a simple branch name without slash
		ref := "main"

		// When: extracting the branch name
		result := extractBranchName(ref)

		// Then: should return the original name
		assert.Equal(t, "main", result)
	})

	t.Run("should handle empty string", func(t *testing.T) {
		// Given: an empty string
		ref := ""

		// When: extracting the branch name
		result := extractBranchName(ref)

		// Then: should return empty string
		assert.Equal(t, "", result)
	})

	t.Run("should handle reference ending with slash", func(t *testing.T) {
		// Given: a reference ending with slash
		ref := "origin/"

		// When: extracting the branch name
		result := extractBranchName(ref)

		// Then: should return empty string after slash
		assert.Equal(t, "", result)
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
		assert.Equal(t,
			[]string{"worktree", "add", "--force", "-b", "new-feature", "../worktrees/feature", "feature"},
			cmd.Args)
	})

	t.Run("should build git worktree remove command", func(t *testing.T) {
		// Given: a worktree path to remove
		path := "../worktrees/feature"

		// When: building a worktree remove command
		cmd := GitWorktreeRemove(path, false)

		// Then: command should have correct structure
		assert.Equal(t, "git", cmd.Name)
		assert.Equal(t, []string{"worktree", "remove", path}, cmd.Args)
	})

	t.Run("should build forced git worktree remove command", func(t *testing.T) {
		// Given: a worktree path to remove forcefully
		path := "../worktrees/feature"

		// When: building a forced worktree remove command
		cmd := GitWorktreeRemove(path, true)

		// Then: command should include force flag
		assert.Equal(t, "git", cmd.Name)
		assert.Equal(t, []string{"worktree", "remove", "--force", path}, cmd.Args)
	})

	t.Run("should build git worktree list command", func(t *testing.T) {
		// When: building a worktree list command
		cmd := GitWorktreeList()

		// Then: command should have correct structure
		assert.Equal(t, "git", cmd.Name)
		assert.Equal(t, []string{"worktree", "list", "--porcelain"}, cmd.Args)
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

// Test real executor functions
func TestRealExecutor(t *testing.T) {
	t.Run("should create real executor", func(t *testing.T) {
		// When: creating a real executor
		executor := NewRealExecutor()

		// Then: should return a valid executor
		assert.NotNil(t, executor)
		assert.Implements(t, (*Executor)(nil), executor)
	})

	t.Run("should execute simple command successfully", func(t *testing.T) {
		// Given: a real executor
		executor := NewRealExecutor()

		// When: executing a simple echo command
		cmd := Command{
			Name: "echo",
			Args: []string{"hello"},
		}
		result, err := executor.Execute([]Command{cmd})

		// Then: should execute successfully
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Len(t, result.Results, 1)
		assert.Equal(t, "hello", result.Results[0].Output)
		assert.Nil(t, result.Results[0].Error)
	})

	t.Run("should handle command failure gracefully", func(t *testing.T) {
		// Given: a real executor
		executor := NewRealExecutor()

		// When: executing a command that will fail
		cmd := Command{
			Name: "false", // 'false' command always returns exit code 1
			Args: []string{},
		}
		result, err := executor.Execute([]Command{cmd})

		// Then: Execute should not return error, but Result should contain error
		assert.NoError(t, err) // Execute itself doesn't fail
		assert.NotNil(t, result)
		assert.Len(t, result.Results, 1)
		assert.NotNil(t, result.Results[0].Error)
	})
}

// Test real shell executor functions
func TestRealShellExecutor(t *testing.T) {
	t.Run("should create real shell executor", func(t *testing.T) {
		// When: creating a real shell executor
		shell := NewRealShellExecutor()

		// Then: should return a valid shell executor
		assert.NotNil(t, shell)
		assert.Implements(t, (*ShellExecutor)(nil), shell)
	})

	t.Run("should execute command and return output", func(t *testing.T) {
		// Given: a real shell executor
		shell := NewRealShellExecutor()

		// When: executing a simple command
		output, err := shell.Execute("echo", []string{"test output"}, "", false)

		// Then: should return correct output
		assert.NoError(t, err)
		assert.Equal(t, "test output", output)
	})

	t.Run("should handle command with working directory", func(t *testing.T) {
		// Given: a real shell executor
		shell := NewRealShellExecutor()

		// When: executing pwd command in /tmp directory
		output, err := shell.Execute("pwd", []string{}, "/tmp", false)

		// Then: should return /tmp as output
		assert.NoError(t, err)
		assert.Contains(t, output, "tmp")
	})

	t.Run("should handle command failure", func(t *testing.T) {
		// Given: a real shell executor
		shell := NewRealShellExecutor()

		// When: executing a command that doesn't exist
		_, err := shell.Execute("nonexistent-command-xyz", []string{}, "", false)

		// Then: should return error
		assert.Error(t, err)
		// Note: output can be empty or contain error message depending on system
	})

	t.Run("should trim whitespace from output", func(t *testing.T) {
		// Given: a real shell executor
		shell := NewRealShellExecutor()

		// When: executing command that produces output with trailing newline
		output, err := shell.Execute("printf", []string{"test\n"}, "", false)

		// Then: output should be trimmed (strings.TrimSpace removes leading/trailing whitespace)
		assert.NoError(t, err)
		assert.Equal(t, "test", output) // TrimSpace removes newlines and spaces
	})
}

// Mock implementation for testing
type mockShellExecutor struct {
	executedCommands []executedCommand
	shouldFail       bool
	failOutput       string
	lastWorkDir      string
	lastInteractive  bool
}

type executedCommand struct {
	name        string
	args        []string
	workDir     string
	interactive bool
}

func (m *mockShellExecutor) Execute(name string, args []string, workDir string, interactive bool) (string, error) {
	m.executedCommands = append(m.executedCommands, executedCommand{
		name:        name,
		args:        args,
		workDir:     workDir,
		interactive: interactive,
	})
	m.lastWorkDir = workDir
	m.lastInteractive = interactive

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
