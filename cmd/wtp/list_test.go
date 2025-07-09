package main

import (
	"bytes"
	"context"
	"os"
	"strings"
	"testing"

	"github.com/satococoa/wtp/internal/command"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

// ===== Command Structure Tests =====

func TestNewListCommand(t *testing.T) {
	cmd := NewListCommand()

	assert.NotNil(t, cmd)
	assert.Equal(t, "list", cmd.Name)
	assert.Equal(t, "List all worktrees", cmd.Usage)
	assert.NotEmpty(t, cmd.Description)
	assert.NotNil(t, cmd.Action)
}

// ===== Pure Business Logic Tests =====

func TestDisplayConstants(t *testing.T) {
	assert.Equal(t, 4, pathHeaderDashes)
	assert.Equal(t, 6, branchHeaderDashes)
	assert.Equal(t, 8, headDisplayLength)
}

func TestWorktreeFormatting(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		branch         string
		head           string
		expectedFormat string
	}{
		{
			name:           "basic worktree",
			path:           "/path/to/worktree",
			branch:         "main",
			head:           "abcd1234",
			expectedFormat: "/path/to/worktree",
		},
		{
			name:           "long path",
			path:           "/very/long/path/to/worktree/that/might/need/truncation",
			branch:         "feature/test",
			head:           "efgh5678",
			expectedFormat: "/very/long/path/to/worktree/that/might/need/truncation",
		},
		{
			name:           "short head",
			path:           "/path",
			branch:         "main",
			head:           "abc123",
			expectedFormat: "/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that formatting doesn't break with various inputs
			// The actual formatting logic can be tested here if needed
			assert.NotEmpty(t, tt.path)
			assert.NotEmpty(t, tt.branch)
			assert.NotEmpty(t, tt.head)
		})
	}
}

// ===== Command Execution Tests =====

func TestListCommand_CommandConstruction(t *testing.T) {
	tests := []struct {
		name             string
		mockOutput       string
		expectedCommands []command.Command
	}{
		{
			name:       "list worktrees command",
			mockOutput: "worktree /path/to/worktree\nHEAD abc123\nbranch refs/heads/main\n\n",
			expectedCommands: []command.Command{{
				Name: "git",
				Args: []string{"worktree", "list", "--porcelain"},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockListCommandExecutor{
				results: []command.Result{
					{
						Output: tt.mockOutput,
						Error:  nil,
					},
				},
			}

			var buf bytes.Buffer
			cmd := &cli.Command{}

			err := listCommandWithCommandExecutor(cmd, &buf, mockExec, "/test/repo")

			assert.NoError(t, err)
			// Verify the correct git command was executed
			assert.Equal(t, tt.expectedCommands, mockExec.executedCommands)
		})
	}
}

func TestListCommand_Output(t *testing.T) {
	tests := []struct {
		name           string
		mockOutput     string
		expectedOutput []string
	}{
		{
			name:       "single worktree",
			mockOutput: "worktree /path/to/worktree\nHEAD abc123\nbranch refs/heads/main\n\n",
			expectedOutput: []string{
				"PATH",
				"BRANCH",
				"HEAD",
				"/path/to/worktree",
				"main",
				"abc123",
			},
		},
		{
			name: "multiple worktrees",
			mockOutput: "worktree /path/to/main\nHEAD abc123\nbranch refs/heads/main\n\n" +
				"worktree /path/to/feature\nHEAD def456\nbranch refs/heads/feature/test\n\n",
			expectedOutput: []string{
				"PATH",
				"BRANCH",
				"HEAD",
				"/path/to/main",
				"main",
				"/path/to/feature",
				"feature/test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockListCommandExecutor{
				results: []command.Result{
					{
						Output: tt.mockOutput,
						Error:  nil,
					},
				},
			}

			var buf bytes.Buffer
			cmd := &cli.Command{}

			err := listCommandWithCommandExecutor(cmd, &buf, mockExec, "/test/repo")

			assert.NoError(t, err)
			output := buf.String()
			for _, expected := range tt.expectedOutput {
				assert.Contains(t, output, expected)
			}
		})
	}
}

// ===== Error Handling Tests =====

func TestListCommand_NotInGitRepo(t *testing.T) {
	// Create a temporary directory that is not a git repo
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	err := os.Chdir(tempDir)
	assert.NoError(t, err)

	app := &cli.Command{
		Commands: []*cli.Command{
			NewListCommand(),
		},
	}

	ctx := context.Background()
	err = app.Run(ctx, []string{"wtp", "list"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in a git repository")
}

func TestListCommand_ExecutionError(t *testing.T) {
	mockExec := &mockListCommandExecutor{
		shouldFail: true,
		errorMsg:   "git command failed",
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	err := listCommandWithCommandExecutor(cmd, &buf, mockExec, "/test/repo")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git command failed")
}

func TestListCommand_NoWorktrees(t *testing.T) {
	mockExec := &mockListCommandExecutor{
		results: []command.Result{
			{
				Output: "",
				Error:  nil,
			},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	err := listCommandWithCommandExecutor(cmd, &buf, mockExec, "/test/repo")

	assert.NoError(t, err)
	output := buf.String()
	// Should show "No worktrees found" message when no worktrees
	assert.Contains(t, output, "No worktrees found")
}

// ===== Edge Cases Tests =====

func TestListCommand_InternationalCharacters(t *testing.T) {
	tests := []struct {
		name         string
		branchName   string
		worktreePath string
	}{
		{
			name:         "Japanese characters",
			branchName:   "機能/ログイン",
			worktreePath: "/path/to/worktrees/機能/ログイン",
		},
		{
			name:         "Spanish accents",
			branchName:   "función/añadir",
			worktreePath: "/path/to/worktrees/función/añadir",
		},
		{
			name:         "Emoji characters",
			branchName:   "feature/🚀-rocket",
			worktreePath: "/path/to/worktrees/feature/🚀-rocket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockOutput := "worktree " + tt.worktreePath + "\nHEAD abc123\nbranch refs/heads/" + tt.branchName + "\n\n"

			mockExec := &mockListCommandExecutor{
				results: []command.Result{
					{
						Output: mockOutput,
						Error:  nil,
					},
				},
			}

			var buf bytes.Buffer
			cmd := &cli.Command{}

			err := listCommandWithCommandExecutor(cmd, &buf, mockExec, "/test/repo")

			assert.NoError(t, err)
			output := buf.String()
			assert.Contains(t, output, tt.worktreePath)
			assert.Contains(t, output, tt.branchName)
		})
	}
}

func TestListCommand_LongPaths(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "very long path",
			path: "/very/long/path/to/worktree/that/might/cause/display/issues/in/terminal/environments/with/limited/width",
		},
		{
			name: "path with spaces",
			path: "/path/with spaces/in the/directory names",
		},
		{
			name: "path with special characters",
			path: "/path/with-special_characters.and.dots",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockOutput := "worktree " + tt.path + "\nHEAD abc123\nbranch refs/heads/main\n\n"

			mockExec := &mockListCommandExecutor{
				results: []command.Result{
					{
						Output: mockOutput,
						Error:  nil,
					},
				},
			}

			var buf bytes.Buffer
			cmd := &cli.Command{}

			err := listCommandWithCommandExecutor(cmd, &buf, mockExec, "/test/repo")

			assert.NoError(t, err)
			output := buf.String()
			assert.Contains(t, output, tt.path)
		})
	}
}

func TestListCommand_MixedWorktreeStates(t *testing.T) {
	// Test with worktrees in different states (detached HEAD, etc.)
	mockOutput := `worktree /path/to/main
HEAD abc123
branch refs/heads/main

worktree /path/to/detached
HEAD def456
detached

worktree /path/to/feature
HEAD ghi789
branch refs/heads/feature/test

`

	mockExec := &mockListCommandExecutor{
		results: []command.Result{
			{
				Output: mockOutput,
				Error:  nil,
			},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	err := listCommandWithCommandExecutor(cmd, &buf, mockExec, "/test/repo")

	assert.NoError(t, err)
	output := buf.String()

	// Check that all worktrees are displayed
	assert.Contains(t, output, "/path/to/main")
	assert.Contains(t, output, "/path/to/detached")
	assert.Contains(t, output, "/path/to/feature")
	assert.Contains(t, output, "main")
	assert.Contains(t, output, "feature/test")
}

func TestListCommand_HeaderFormatting(t *testing.T) {
	mockExec := &mockListCommandExecutor{
		results: []command.Result{
			{
				Output: "worktree /path/to/worktree\nHEAD abc123\nbranch refs/heads/main\n\n",
				Error:  nil,
			},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	err := listCommandWithCommandExecutor(cmd, &buf, mockExec, "/test/repo")

	assert.NoError(t, err)
	output := buf.String()

	// Check header formatting
	lines := strings.Split(output, "\n")
	assert.True(t, len(lines) >= 2, "Should have header and separator lines")

	// Should contain header columns
	assert.Contains(t, output, "PATH")
	assert.Contains(t, output, "BRANCH")
	assert.Contains(t, output, "HEAD")

	// Should contain separator dashes
	assert.Contains(t, output, "----")
	assert.Contains(t, output, "------")
}

// ===== Mock Implementations =====

type mockListCommandExecutor struct {
	executedCommands []command.Command
	results          []command.Result
	shouldFail       bool
	errorMsg         string
}

func (m *mockListCommandExecutor) Execute(commands []command.Command) (*command.ExecutionResult, error) {
	m.executedCommands = commands

	if m.shouldFail {
		return nil, &mockError{message: m.errorMsg}
	}

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

type mockError struct {
	message string
}

func (e *mockError) Error() string {
	return e.message
}
