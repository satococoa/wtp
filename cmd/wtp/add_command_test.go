package main

import (
	"bytes"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/satococoa/wtp/internal/command"
	"github.com/satococoa/wtp/internal/config"
)

// Test the new command-based add implementation
func TestAddCommand_UsingCommandExecutor(t *testing.T) {
	tests := []struct {
		name             string
		flags            map[string]interface{}
		args             []string
		expectedCommands []command.Command
		expectError      bool
	}{
		{
			name:  "basic worktree creation",
			flags: map[string]interface{}{},
			args:  []string{"feature/test"},
			expectedCommands: []command.Command{
				{
					Name: "git",
					Args: []string{"worktree", "add", "/test/worktrees/feature/test", "feature/test"},
				},
			},
			expectError: false,
		},
		{
			name: "worktree creation with force",
			flags: map[string]interface{}{
				"force": true,
			},
			args: []string{"feature/test"},
			expectedCommands: []command.Command{
				{
					Name: "git",
					Args: []string{"worktree", "add", "--force", "/test/worktrees/feature/test", "feature/test"},
				},
			},
			expectError: false,
		},
		{
			name: "new branch creation",
			flags: map[string]interface{}{
				"branch": "new-feature",
			},
			args: []string{"main"},
			expectedCommands: []command.Command{
				{
					Name: "git",
					Args: []string{"worktree", "add", "-b", "new-feature", "/test/worktrees/new-feature"},
				},
			},
			expectError: false,
		},
		{
			name: "detached HEAD",
			flags: map[string]interface{}{
				"detach": true,
			},
			args: []string{"abc1234"},
			expectedCommands: []command.Command{
				{
					Name: "git",
					Args: []string{"worktree", "add", "--detach", "/test/worktrees/abc1234", "abc1234"},
				},
			},
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test setup
			cmd := createTestCommand(tt.flags, tt.args)
			var buf bytes.Buffer
			mockCommandExecutor := &mockCommandExecutor{}

			cfg := &config.Config{
				Defaults: config.Defaults{
					BaseDir: "/test/worktrees",
				},
			}

			// Execute the function
			err := addCommandWithCommandExecutor(cmd, &buf, mockCommandExecutor, cfg, "/test/repo")

			// Check error expectation
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify commands were executed correctly
			assert.Equal(t, tt.expectedCommands, mockCommandExecutor.executedCommands)
		})
	}
}

// Mock CommandExecutor for testing
type mockCommandExecutor struct {
	executedCommands []command.Command
	shouldFail       bool
	results          *command.ExecutionResult
}

func (m *mockCommandExecutor) Execute(commands []command.Command) (*command.ExecutionResult, error) {
	m.executedCommands = commands

	if m.shouldFail {
		return nil, &mockCommandError{msg: "command execution failed"}
	}

	if m.results != nil {
		return m.results, nil
	}

	// Default successful result
	results := make([]command.Result, len(commands))
	for i, cmd := range commands {
		results[i] = command.Result{
			Command: cmd,
			Output:  "success",
			Error:   nil,
		}
	}

	return &command.ExecutionResult{Results: results}, nil
}

type mockCommandError struct {
	msg string
}

func (e *mockCommandError) Error() string {
	return e.msg
}
