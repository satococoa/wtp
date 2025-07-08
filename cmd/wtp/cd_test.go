package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/satococoa/wtp/internal/command"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

// ===== Simple Unit Tests (What testing) =====

func TestNewCdCommand(t *testing.T) {
	cmd := NewCdCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "cd", cmd.Name)
	assert.Equal(t, "Change directory to worktree (requires shell integration)", cmd.Usage)
	assert.NotEmpty(t, cmd.Description)
	assert.Contains(t, cmd.Description, "shell integration")
	assert.Contains(t, cmd.Description, "Bash:")
	assert.Contains(t, cmd.Description, "Zsh:")
	assert.Contains(t, cmd.Description, "Fish:")
	assert.NotNil(t, cmd.Action)
	assert.Equal(t, "<worktree-name>", cmd.ArgsUsage)
}

func TestCdToWorktree_NoShellIntegration(t *testing.T) {
	// Ensure WTP_SHELL_INTEGRATION is not set
	os.Unsetenv("WTP_SHELL_INTEGRATION")

	app := &cli.Command{
		Commands: []*cli.Command{
			NewCdCommand(),
		},
	}

	var buf bytes.Buffer
	app.Writer = &buf

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "cd", "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cd command requires shell integration")
}

func TestCdToWorktree_NoArguments(t *testing.T) {
	// Set shell integration
	os.Setenv("WTP_SHELL_INTEGRATION", "1")
	defer os.Unsetenv("WTP_SHELL_INTEGRATION")

	app := &cli.Command{
		Commands: []*cli.Command{
			NewCdCommand(),
		},
	}

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "cd"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worktree name is required")
}

func TestCdToWorktree_NotInGitRepo(t *testing.T) {
	// Set shell integration
	os.Setenv("WTP_SHELL_INTEGRATION", "1")
	defer os.Unsetenv("WTP_SHELL_INTEGRATION")

	// Create a temp dir and cd to it
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	err := os.Chdir(tempDir)
	assert.NoError(t, err)

	app := &cli.Command{
		Commands: []*cli.Command{
			NewCdCommand(),
		},
	}

	ctx := context.Background()
	err = app.Run(ctx, []string{"wtp", "cd", "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in a git repository")
}

func TestCdCommand_Success(t *testing.T) {
	mockExec := &mockCdCommandExecutor{
		results: []command.Result{
			{
				Output: "worktree /path/to/worktrees/feature-branch\nHEAD abc123\nbranch refs/heads/feature-branch\n\n",
				Error:  nil,
			},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	err := cdCommandWithCommandExecutor(cmd, &buf, mockExec, "/repo", "feature-branch")

	assert.NoError(t, err)
	assert.Equal(t, "/path/to/worktrees/feature-branch\n", buf.String())
	assert.Len(t, mockExec.executedCommands, 1)
	assert.Equal(t, []string{"worktree", "list", "--porcelain"}, mockExec.executedCommands[0].Args)
}

func TestCdCommand_NotFound(t *testing.T) {
	mockExec := &mockCdCommandExecutor{
		results: []command.Result{
			{
				Output: "worktree /path/to/worktrees/main\nHEAD abc123\nbranch refs/heads/main\n\n",
				Error:  nil,
			},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	err := cdCommandWithCommandExecutor(cmd, &buf, mockExec, "/repo", "feature-branch")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worktree 'feature-branch' not found")
}

func TestCdCommand_ShellComplete(t *testing.T) {
	cmd := NewCdCommand()
	// cd command doesn't have shell completion
	assert.Nil(t, cmd.ShellComplete)
}

func TestCdCommand_NoWorktrees(t *testing.T) {
	mockExec := &mockCdCommandExecutor{
		results: []command.Result{
			{
				Output: "",
				Error:  nil,
			},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	err := cdCommandWithCommandExecutor(cmd, &buf, mockExec, "/repo", "feature-branch")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worktree 'feature-branch' not found")
}

// ===== Mock Implementations =====

type mockCdCommandExecutor struct {
	executedCommands []command.Command
	results          []command.Result
}

func (m *mockCdCommandExecutor) Execute(commands []command.Command) (*command.ExecutionResult, error) {
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

// ===== Real-World Edge Cases =====

func TestCdCommand_InternationalCharacters(t *testing.T) {
	tests := []struct {
		name         string
		branchName   string
		worktreePath string
		shouldWork   bool
	}{
		{
			name:         "Japanese characters",
			branchName:   "機能/ログイン",
			worktreePath: "/path/to/worktrees/機能/ログイン",
			shouldWork:   true,
		},
		{
			name:         "Spanish accents",
			branchName:   "función/añadir",
			worktreePath: "/path/to/worktrees/función/añadir",
			shouldWork:   true,
		},
		{
			name:         "Emoji characters",
			branchName:   "feature/🚀-rocket",
			worktreePath: "/path/to/worktrees/feature/🚀-rocket",
			shouldWork:   true,
		},
		{
			name:         "Arabic characters",
			branchName:   "ميزة/تسجيل-الدخول",
			worktreePath: "/path/to/worktrees/ميزة/تسجيل-الدخول",
			shouldWork:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output string
			if tt.shouldWork {
				output = fmt.Sprintf("worktree %s\nHEAD abc123\nbranch refs/heads/%s\n\n",
					tt.worktreePath, tt.branchName)
			} else {
				output = "" // No matching worktree
			}

			mockExec := &mockCdCommandExecutor{
				results: []command.Result{
					{Output: output, Error: nil},
				},
			}

			var buf bytes.Buffer
			cmd := &cli.Command{}

			err := cdCommandWithCommandExecutor(cmd, &buf, mockExec, "/repo", filepath.Base(tt.worktreePath))

			if tt.shouldWork {
				assert.NoError(t, err)
				assert.Equal(t, tt.worktreePath+"\n", buf.String())
			} else {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "not found")
			}
		})
	}
}

func TestCdCommand_PathResolution(t *testing.T) {
	tests := []struct {
		name             string
		branchName       string
		worktreePath     string
		expectedOutput   string
		shouldFindBranch bool
		description      string
	}{
		{
			name:             "spaces in path",
			branchName:       "feature/with spaces",
			worktreePath:     "/path/to/worktrees/feature/with spaces",
			expectedOutput:   "/path/to/worktrees/feature/with spaces\n",
			shouldFindBranch: true,
			description:      "Should handle spaces in paths correctly",
		},
		{
			name:             "relative paths",
			branchName:       "feature/test",
			worktreePath:     "../worktrees/feature/test",
			expectedOutput:   "../worktrees/feature/test\n",
			shouldFindBranch: true,
			description:      "Should handle relative paths",
		},
		{
			name:             "deeply nested path",
			branchName:       "team/backend/feature/auth",
			worktreePath:     "/path/to/worktrees/team/backend/feature/auth",
			expectedOutput:   "/path/to/worktrees/team/backend/feature/auth\n",
			shouldFindBranch: true,
			description:      "Should handle deeply nested paths",
		},
		{
			name:             "main worktree",
			branchName:       "main",
			worktreePath:     ".",
			expectedOutput:   ".\n",
			shouldFindBranch: true,
			description:      "Should handle main worktree path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var output string
			if tt.shouldFindBranch {
				output = fmt.Sprintf("worktree %s\nHEAD abc123\nbranch refs/heads/%s\n\n",
					tt.worktreePath, tt.branchName)
			} else {
				output = "worktree /other/path\nHEAD def456\nbranch refs/heads/other\n\n"
			}

			mockExec := &mockCdCommandExecutor{
				results: []command.Result{
					{Output: output, Error: nil},
				},
			}

			var buf bytes.Buffer
			cmd := &cli.Command{}

			err := cdCommandWithCommandExecutor(cmd, &buf, mockExec, "/repo", filepath.Base(tt.worktreePath))

			if tt.shouldFindBranch {
				assert.NoError(t, err, tt.description)
				assert.Equal(t, tt.expectedOutput, buf.String())
			} else {
				assert.Error(t, err, tt.description)
			}
		})
	}
}

func TestCdCommand_AmbiguousBranches(t *testing.T) {
	// Test case where multiple worktrees might have similar names
	mockOutput := `worktree /path/to/worktrees/feature/test
HEAD abc123
branch refs/heads/feature/test

worktree /path/to/worktrees/feature/test-2
HEAD def456
branch refs/heads/feature/test-2

worktree /path/to/worktrees/bugfix/auth
HEAD hij789
branch refs/heads/bugfix/auth

`

	mockExec := &mockCdCommandExecutor{
		results: []command.Result{
			{Output: mockOutput, Error: nil},
		},
	}

	tests := []struct {
		branchName   string
		expectedPath string
	}{
		{"test", "/path/to/worktrees/feature/test"},
		{"test-2", "/path/to/worktrees/feature/test-2"},
		{"auth", "/path/to/worktrees/bugfix/auth"},
	}

	for _, tt := range tests {
		t.Run(tt.branchName, func(t *testing.T) {
			var buf bytes.Buffer
			cmd := &cli.Command{}

			err := cdCommandWithCommandExecutor(cmd, &buf, mockExec, "/repo", filepath.Base(tt.expectedPath))

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPath+"\n", buf.String())
		})
	}
}
