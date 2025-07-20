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

// ===== Command Structure Tests =====

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

// ===== Pure Business Logic Tests =====

func TestCdCommand_WorktreePathResolution(t *testing.T) {
	tests := []struct {
		name         string
		worktreeName string
		worktreeList string
		expectedPath string
		shouldFind   bool
	}{
		{
			name:         "exact match",
			worktreeName: "feature-branch",
			worktreeList: "worktree /path/to/worktrees/feature-branch\nHEAD abc123\nbranch refs/heads/feature-branch\n\n",
			expectedPath: "/path/to/worktrees/feature-branch",
			shouldFind:   true,
		},
		{
			name:         "no match",
			worktreeName: "nonexistent",
			worktreeList: "worktree /path/to/worktrees/main\nHEAD abc123\nbranch refs/heads/main\n\n",
			expectedPath: "",
			shouldFind:   false,
		},
		{
			name:         "multiple worktrees",
			worktreeName: "test",
			worktreeList: "worktree /path/to/worktrees/feature/test\nHEAD abc123\nbranch refs/heads/feature/test\n\n" +
				"worktree /path/to/worktrees/bugfix\nHEAD def456\nbranch refs/heads/bugfix\n\n",
			expectedPath: "/path/to/worktrees/feature/test",
			shouldFind:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worktrees := parseWorktreesFromOutput(tt.worktreeList)

			var targetPath string
			for _, wt := range worktrees {
				if filepath.Base(wt.Path) == tt.worktreeName {
					targetPath = wt.Path
					break
				}
			}

			if tt.shouldFind {
				assert.Equal(t, tt.expectedPath, targetPath)
			} else {
				assert.Empty(t, targetPath)
			}
		})
	}
}

func TestCdCommand_BranchNameResolution(t *testing.T) {
	// Test that cd command can resolve worktrees by branch name (with prefixes)
	// This is needed for the new completion system that shows full branch names
	tests := []struct {
		name         string
		worktreeName string
		worktreeList string
		expectedPath string
		shouldFind   bool
	}{
		{
			name:         "feature branch with prefix",
			worktreeName: "feature/awesome",
			worktreeList: "worktree /path/to/worktrees/feature-awesome\nHEAD abc123\nbranch refs/heads/feature/awesome\n\n",
			expectedPath: "/path/to/worktrees/feature-awesome",
			shouldFind:   true,
		},
		{
			name:         "fix branch with nested prefix",
			worktreeName: "fix/123/fix-login",
			worktreeList: "worktree /path/to/worktrees/fix-123-fix-login\nHEAD def456\nbranch refs/heads/fix/123/fix-login\n\n",
			expectedPath: "/path/to/worktrees/fix-123-fix-login",
			shouldFind:   true,
		},
		{
			name:         "root worktree by alias",
			worktreeName: "root",
			worktreeList: "worktree /path/to/main\nHEAD ghi789\nbranch refs/heads/main\n\n",
			expectedPath: "/path/to/main",
			shouldFind:   true,
		},
		{
			name:         "main worktree by @ symbol",
			worktreeName: "@",
			worktreeList: "worktree /path/to/main\nHEAD ghi789\nbranch refs/heads/main\n\n",
			expectedPath: "/path/to/main",
			shouldFind:   true,
		},
		{
			name:         "@ symbol with asterisk from completion",
			worktreeName: "@*",
			worktreeList: "worktree /path/to/main\nHEAD ghi789\nbranch refs/heads/main\n\n",
			expectedPath: "/path/to/main",
			shouldFind:   true,
		},
		{
			name:         "worktree name with asterisk from completion",
			worktreeName: "feature*",
			worktreeList: "worktree /path/to/worktrees/feature\nHEAD abc123\nbranch refs/heads/feature\n\n",
			expectedPath: "/path/to/worktrees/feature",
			shouldFind:   true,
		},
		{
			name:         "root worktree by repo name",
			worktreeName: "wtp(root worktree)", // This is shown in completion
			worktreeList: "worktree /path/to/wtp\nHEAD ghi789\nbranch refs/heads/main\n\n",
			expectedPath: "/path/to/wtp",
			shouldFind:   true,
		},
		{
			name:         "root worktree by completion format",
			worktreeName: "giselle@fix-nodes(root worktree)", // Actual completion format
			worktreeList: "worktree /path/to/giselle\nHEAD ghi789\nbranch refs/heads/fix-nodes\n\n",
			expectedPath: "/path/to/giselle",
			shouldFind:   true,
		},
		{
			name:         "simple branch name still works",
			worktreeName: "develop",
			worktreeList: "worktree /path/to/worktrees/develop\nHEAD jkl012\nbranch refs/heads/develop\n\n",
			expectedPath: "/path/to/worktrees/develop",
			shouldFind:   true,
		},
		{
			name:         "root worktree with develop branch by alias",
			worktreeName: "root",
			worktreeList: "worktree /path/to/myproject\nHEAD abc123\nbranch refs/heads/develop\n\n",
			expectedPath: "/path/to/myproject",
			shouldFind:   true,
		},
		{
			name:         "root worktree with custom branch by repo name",
			worktreeName: "myproject(root worktree)",
			worktreeList: "worktree /path/to/myproject\nHEAD def456\nbranch refs/heads/custom-default\n\n",
			expectedPath: "/path/to/myproject",
			shouldFind:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create mock executor
			mockExec := &mockCdCommandExecutor{
				results: []command.Result{
					{Output: tt.worktreeList, Error: nil},
				},
			}

			// Create buffer to capture output
			var buf bytes.Buffer

			// Call the function
			err := cdCommandWithCommandExecutor(nil, &buf, mockExec, "/some/path", tt.worktreeName)

			if tt.shouldFind {
				assert.NoError(t, err)
				output := buf.String()
				assert.Contains(t, output, tt.expectedPath)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestCdCommand_NewInputAcceptanceRequirements(t *testing.T) {
	// Test the new requirements: cd should accept worktree name (with prefix), branch name, and "root"
	tests := []struct {
		name             string
		worktreeName     string
		worktrees        []ParsedWorktree
		mainWorktreePath string
		expected         string
		shouldFind       bool
	}{
		{
			name:         "accept branch name input for root worktree",
			worktreeName: "fix-nodes",
			worktrees: []ParsedWorktree{
				{Path: "/Users/user/repos/giselle", Branch: "fix-nodes"},
				{Path: "/Users/user/repos/giselle/worktrees/feature-awesome", Branch: "feature/awesome"},
			},
			mainWorktreePath: "/Users/user/repos/giselle",
			expected:         "/Users/user/repos/giselle",
			shouldFind:       true,
		},
		{
			name:         "accept repo name input for root worktree",
			worktreeName: "giselle",
			worktrees: []ParsedWorktree{
				{Path: "/Users/user/repos/giselle", Branch: "fix-nodes"},
				{Path: "/Users/user/repos/giselle/worktrees/feature-awesome", Branch: "feature/awesome"},
			},
			mainWorktreePath: "/Users/user/repos/giselle",
			expected:         "/Users/user/repos/giselle",
			shouldFind:       true, // Now this should work
		},
		{
			name:         "accept branch name input for regular worktree",
			worktreeName: "feature/awesome",
			worktrees: []ParsedWorktree{
				{Path: "/Users/user/repos/giselle", Branch: "fix-nodes"},
				{Path: "/Users/user/repos/worktrees/feature/awesome", Branch: "feature/awesome"},
			},
			mainWorktreePath: "/Users/user/repos/giselle",
			expected:         "/Users/user/repos/worktrees/feature/awesome",
			shouldFind:       true,
		},
		{
			name:         "accept worktree directory name input for regular worktree",
			worktreeName: "awesome",
			worktrees: []ParsedWorktree{
				{Path: "/Users/user/repos/giselle", Branch: "fix-nodes"},
				{Path: "/Users/user/repos/worktrees/feature/awesome", Branch: "feature/awesome"},
			},
			mainWorktreePath: "/Users/user/repos/giselle",
			expected:         "/Users/user/repos/worktrees/feature/awesome",
			shouldFind:       true,
		},
		{
			name:         "root alias should work for any branch",
			worktreeName: "root",
			worktrees: []ParsedWorktree{
				{Path: "/Users/user/repos/giselle", Branch: "fix-nodes"},
				{Path: "/Users/user/repos/worktrees/feature/awesome", Branch: "feature/awesome"},
			},
			mainWorktreePath: "/Users/user/repos/giselle",
			expected:         "/Users/user/repos/giselle",
			shouldFind:       true,
		},
		{
			name:         "should fail for non-existent input",
			worktreeName: "non-existent",
			worktrees: []ParsedWorktree{
				{Path: "/Users/user/repos/giselle", Branch: "fix-nodes"},
				{Path: "/Users/user/repos/worktrees/feature/awesome", Branch: "feature/awesome"},
			},
			mainWorktreePath: "/Users/user/repos/giselle",
			expected:         "",
			shouldFind:       false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Convert test data to worktree list output
			var worktreeListOutput string
			for _, wt := range tt.worktrees {
				worktreeListOutput += fmt.Sprintf("worktree %s\nHEAD abc123\nbranch refs/heads/%s\n\n", wt.Path, wt.Branch)
			}

			// Create mock executor
			mockExec := &mockCdCommandExecutor{
				results: []command.Result{
					{Output: worktreeListOutput, Error: nil},
				},
			}

			// Create buffer to capture output
			var buf bytes.Buffer

			// Call the function
			err := cdCommandWithCommandExecutor(nil, &buf, mockExec, "/some/path", tt.worktreeName)

			if tt.shouldFind {
				assert.NoError(t, err, "Expected to find worktree for input: %s", tt.worktreeName)
				output := buf.String()
				assert.Contains(t, output, tt.expected, "Expected output to contain path: %s", tt.expected)
			} else {
				assert.Error(t, err, "Expected error for non-existent worktree: %s", tt.worktreeName)
			}
		})
	}
}

// ParsedWorktree represents a worktree for testing
type ParsedWorktree struct {
	Path   string
	Branch string
}

// ===== Command Execution Tests =====

func TestCdCommand_SuccessfulExecution(t *testing.T) {
	tests := []struct {
		name         string
		worktreeName string
		mockOutput   string
		expectedPath string
	}{
		{
			name:         "find worktree by name",
			worktreeName: "feature-branch",
			mockOutput:   "worktree /path/to/worktrees/feature-branch\nHEAD abc123\nbranch refs/heads/feature-branch\n\n",
			expectedPath: "/path/to/worktrees/feature-branch",
		},
		{
			name:         "find nested worktree",
			worktreeName: "auth",
			mockOutput:   "worktree /path/to/worktrees/feature/auth\nHEAD abc123\nbranch refs/heads/feature/auth\n\n",
			expectedPath: "/path/to/worktrees/feature/auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockCdCommandExecutor{
				results: []command.Result{
					{
						Output: tt.mockOutput,
						Error:  nil,
					},
				},
			}

			var buf bytes.Buffer
			cmd := &cli.Command{}

			err := cdCommandWithCommandExecutor(cmd, &buf, mockExec, "/repo", tt.worktreeName)

			assert.NoError(t, err)
			assert.Equal(t, tt.expectedPath+"\n", buf.String())
			assert.Len(t, mockExec.executedCommands, 1)
			assert.Equal(t, []string{"worktree", "list", "--porcelain"}, mockExec.executedCommands[0].Args)
		})
	}
}

func TestCdCommand_GitCommandConstruction(t *testing.T) {
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
	// Verify the correct git command was executed
	assert.Len(t, mockExec.executedCommands, 1)
	expectedCmd := command.Command{
		Name: "git",
		Args: []string{"worktree", "list", "--porcelain"},
	}
	assert.Equal(t, expectedCmd.Name, mockExec.executedCommands[0].Name)
	assert.Equal(t, expectedCmd.Args, mockExec.executedCommands[0].Args)
}

// ===== Error Handling Tests =====

func TestCdToWorktree_ValidationErrors(t *testing.T) {
	tests := []struct {
		name             string
		shellIntegration bool
		args             []string
		expectedError    string
	}{
		{
			name:             "no shell integration",
			shellIntegration: false,
			args:             []string{"test"},
			expectedError:    "cd command requires shell integration",
		},
		{
			name:             "no arguments",
			shellIntegration: true,
			args:             []string{},
			expectedError:    "worktree name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Set up environment
			if tt.shellIntegration {
				os.Setenv("WTP_SHELL_INTEGRATION", "1")
				defer os.Unsetenv("WTP_SHELL_INTEGRATION")
			} else {
				os.Unsetenv("WTP_SHELL_INTEGRATION")
			}

			app := &cli.Command{
				Commands: []*cli.Command{
					NewCdCommand(),
				},
			}

			var buf bytes.Buffer
			app.Writer = &buf

			ctx := context.Background()
			cmdArgs := []string{"wtp", "cd"}
			cmdArgs = append(cmdArgs, tt.args...)

			err := app.Run(ctx, cmdArgs)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestCdToWorktree_NotInGitRepo(t *testing.T) {
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

func TestCdCommand_WorktreeNotFound(t *testing.T) {
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

func TestCdCommand_WorktreeNotFound_ShowsConsistentNames(t *testing.T) {
	mockExec := &mockCdCommandExecutor{
		results: []command.Result{
			{
				Output: "worktree /repo\nHEAD abc123\nbranch refs/heads/main\n\n" +
					"worktree /repo/.worktrees/feat/hogehoge\nHEAD def456\nbranch refs/heads/feat/hogehoge\n\n",
				Error: nil,
			},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	err := cdCommandWithCommandExecutor(cmd, &buf, mockExec, "/repo", "feature-branch")

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worktree 'feature-branch' not found")
	// Should show worktree names (main worktree should always be shown)
	assert.Contains(t, err.Error(), "@")
}

// ===== Edge Cases Tests =====

func TestCdCommand_InternationalCharacters(t *testing.T) {
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
		{
			name:         "Arabic characters",
			branchName:   "ميزة/تسجيل-الدخول",
			worktreePath: "/path/to/worktrees/ميزة/تسجيل-الدخول",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := fmt.Sprintf("worktree %s\nHEAD abc123\nbranch refs/heads/%s\n\n",
				tt.worktreePath, tt.branchName)

			mockExec := &mockCdCommandExecutor{
				results: []command.Result{
					{Output: output, Error: nil},
				},
			}

			var buf bytes.Buffer
			cmd := &cli.Command{}

			err := cdCommandWithCommandExecutor(cmd, &buf, mockExec, "/repo", filepath.Base(tt.worktreePath))

			assert.NoError(t, err)
			assert.Equal(t, tt.worktreePath+"\n", buf.String())
		})
	}
}

func TestCdCommand_PathResolution(t *testing.T) {
	tests := []struct {
		name           string
		branchName     string
		worktreePath   string
		expectedOutput string
		description    string
	}{
		{
			name:           "spaces in path",
			branchName:     "feature/with spaces",
			worktreePath:   "/path/to/worktrees/feature/with spaces",
			expectedOutput: "/path/to/worktrees/feature/with spaces\n",
			description:    "Should handle spaces in paths correctly",
		},
		{
			name:           "relative paths",
			branchName:     "feature/test",
			worktreePath:   "../worktrees/feature/test",
			expectedOutput: "../worktrees/feature/test\n",
			description:    "Should handle relative paths",
		},
		{
			name:           "deeply nested path",
			branchName:     "team/backend/feature/auth",
			worktreePath:   "/path/to/worktrees/team/backend/feature/auth",
			expectedOutput: "/path/to/worktrees/team/backend/feature/auth\n",
			description:    "Should handle deeply nested paths",
		},
		{
			name:           "main worktree",
			branchName:     "main",
			worktreePath:   ".",
			expectedOutput: ".\n",
			description:    "Should handle main worktree path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			output := fmt.Sprintf("worktree %s\nHEAD abc123\nbranch refs/heads/%s\n\n",
				tt.worktreePath, tt.branchName)

			mockExec := &mockCdCommandExecutor{
				results: []command.Result{
					{Output: output, Error: nil},
				},
			}

			var buf bytes.Buffer
			cmd := &cli.Command{}

			err := cdCommandWithCommandExecutor(cmd, &buf, mockExec, "/repo", filepath.Base(tt.worktreePath))

			assert.NoError(t, err, tt.description)
			assert.Equal(t, tt.expectedOutput, buf.String())
		})
	}
}

func TestCdCommand_AmbiguousBranches(t *testing.T) {
	// Test case where multiple worktrees might have similar names
	mockOutput := `worktree /repo
HEAD abc123
branch refs/heads/main

worktree /worktrees/feature/test
HEAD abc123
branch refs/heads/feature/test

worktree /worktrees/feature/test-2
HEAD def456
branch refs/heads/feature/test-2

worktree /worktrees/bugfix/auth
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
		{"test", "/worktrees/feature/test"},
		{"test-2", "/worktrees/feature/test-2"},
		{"auth", "/worktrees/bugfix/auth"},
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
