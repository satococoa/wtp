package main

import (
	"bytes"
	"context"
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"

	"github.com/satococoa/wtp/internal/command"
	"github.com/satococoa/wtp/internal/config"
	"github.com/satococoa/wtp/internal/errors"
)

// ===== Living Specifications: User Behavior Tests =====

// User Story: As a developer, I want to create worktrees for different branches
// so that I can work on multiple features simultaneously without switching contexts.

// Test command structure and flags
func TestNewAddCommand(t *testing.T) {
	cmd := NewAddCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "add", cmd.Name)
	assert.Equal(t, "Create a new worktree", cmd.Usage)
	assert.NotEmpty(t, cmd.Description)
	assert.NotNil(t, cmd.Action)
	assert.NotNil(t, cmd.ShellComplete)

	// Check flags exist
	flagNames := []string{"path", "force", "detach", "branch", "track", "cd", "no-cd"}
	for _, name := range flagNames {
		found := false
		for _, flag := range cmd.Flags {
			if flag.Names()[0] == name {
				found = true
				break
			}
		}
		assert.True(t, found, "Flag %s should exist", name)
	}
}

// Test input validation
func TestValidateAddInput(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		branch  string
		wantErr bool
	}{
		{
			name:    "no args and no branch flag",
			args:    []string{},
			branch:  "",
			wantErr: true,
		},
		{
			name:    "with args",
			args:    []string{"feature"},
			branch:  "",
			wantErr: false,
		},
		{
			name:    "with branch flag",
			args:    []string{},
			branch:  "new-feature",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a properly initialized app
			app := &cli.Command{
				Name: "test",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "branch"},
				},
				Action: func(_ context.Context, cmd *cli.Command) error {
					return validateAddInput(cmd)
				},
			}

			// Build args
			args := []string{"test"}
			if tt.branch != "" {
				args = append(args, "--branch", tt.branch)
			}
			args = append(args, tt.args...)

			ctx := context.Background()
			err := app.Run(ctx, args)

			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "branch name is required")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

// Test git command building logic
func TestBuildGitWorktreeArgs(t *testing.T) {
	tests := []struct {
		name         string
		workTreePath string
		branchName   string
		flags        map[string]interface{}
		cliArgs      []string
		want         []string
	}{
		{
			name:         "simple branch",
			workTreePath: "/path/to/worktree",
			branchName:   "feature",
			flags:        map[string]interface{}{},
			want:         []string{"worktree", "add", "/path/to/worktree", "feature"},
		},
		{
			name:         "with force flag",
			workTreePath: "/path/to/worktree",
			branchName:   "feature",
			flags:        map[string]interface{}{"force": true},
			want:         []string{"worktree", "add", "--force", "/path/to/worktree", "feature"},
		},
		{
			name:         "with new branch flag",
			workTreePath: "/path/to/worktree",
			branchName:   "new-feature",
			flags:        map[string]interface{}{"branch": "new-feature"},
			want:         []string{"worktree", "add", "-b", "new-feature", "/path/to/worktree"},
		},
		{
			name:         "detached HEAD",
			workTreePath: "/path/to/worktree",
			branchName:   "",
			flags:        map[string]interface{}{"detach": true},
			cliArgs:      []string{"abc1234"},
			want:         []string{"worktree", "add", "--detach", "/path/to/worktree", "abc1234"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create app with all required flags
			app := &cli.Command{
				Name: "test",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "force"},
					&cli.BoolFlag{Name: "detach"},
					&cli.StringFlag{Name: "branch"},
					&cli.StringFlag{Name: "track"},
					&cli.StringFlag{Name: "path"},
				},
				Action: func(_ context.Context, cmd *cli.Command) error {
					result := buildGitWorktreeArgs(cmd, tt.workTreePath, tt.branchName)
					assert.Equal(t, tt.want, result)
					return nil
				},
			}

			// Build args
			args := []string{"test"}
			for flag, value := range tt.flags {
				switch v := value.(type) {
				case bool:
					if v {
						args = append(args, "--"+flag)
					}
				case string:
					args = append(args, "--"+flag, v)
				}
			}
			args = append(args, tt.cliArgs...)

			ctx := context.Background()
			err := app.Run(ctx, args)
			assert.NoError(t, err)
		})
	}
}

// Test with new CommandExecutor architecture
func TestAddCommand_WithCommandExecutor(t *testing.T) {
	tests := []struct {
		name             string
		flags            map[string]interface{}
		args             []string
		expectedCommands []command.Command
		executorError    error
		expectError      bool
		expectedOutput   string
	}{
		{
			name:  "successful worktree creation",
			flags: map[string]interface{}{},
			args:  []string{"feature/test"},
			expectedCommands: []command.Command{{
				Name: "git",
				Args: []string{"worktree", "add", "/test/worktrees/feature/test", "feature/test"},
			}},
			expectError:    false,
			expectedOutput: "Created worktree 'feature/test'",
		},
		{
			name: "worktree with force flag",
			flags: map[string]interface{}{
				"force": true,
			},
			args: []string{"feature/test"},
			expectedCommands: []command.Command{{
				Name: "git",
				Args: []string{"worktree", "add", "--force", "/test/worktrees/feature/test", "feature/test"},
			}},
			expectError: false,
		},
		{
			name: "new branch creation",
			flags: map[string]interface{}{
				"branch": "new-feature",
			},
			args: []string{"main"},
			expectedCommands: []command.Command{{
				Name: "git",
				Args: []string{"worktree", "add", "-b", "new-feature", "/test/worktrees/new-feature"},
			}},
			expectError:    false,
			expectedOutput: "Created worktree 'new-feature'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			cmd := createTestCLICommand(tt.flags, tt.args)
			var buf bytes.Buffer
			mockExec := &mockCommandExecutor{}

			if tt.executorError != nil {
				mockExec.shouldFail = true
			}

			cfg := &config.Config{
				Defaults: config.Defaults{
					BaseDir: "/test/worktrees",
				},
			}

			// Execute
			err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

			// Verify
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				if tt.expectedOutput != "" {
					assert.Contains(t, buf.String(), tt.expectedOutput)
				}
			}

			assert.Equal(t, tt.expectedCommands, mockExec.executedCommands)
		})
	}
}

// Test with GitExecutor interface (legacy)
func TestAddCommand_WithGitExecutor(t *testing.T) {
	tests := []struct {
		name           string
		flags          map[string]interface{}
		args           []string
		expectedArgs   []string
		resolvedBranch string
		isRemoteBranch bool
		expectError    bool
	}{
		{
			name:         "local branch",
			flags:        map[string]interface{}{},
			args:         []string{"feature/test"},
			expectedArgs: []string{"worktree", "add", "/test/worktrees/feature/test", "feature/test"},
			expectError:  false,
		},
		{
			name:  "remote branch auto-tracking",
			flags: map[string]interface{}{},
			args:  []string{"feature/remote"},
			expectedArgs: []string{
				"worktree", "add", "--track", "-b", "feature/remote",
				"/test/worktrees/feature/remote", "origin/feature/remote",
			},
			resolvedBranch: "origin/feature/remote",
			isRemoteBranch: true,
			expectError:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Setup
			cmd := createTestCLICommand(tt.flags, tt.args)
			var buf bytes.Buffer
			mockExec := newMockGitExecutor()

			if tt.resolvedBranch != "" {
				mockExec.SetResolveBranch(tt.resolvedBranch, tt.isRemoteBranch, nil)
			}

			cfg := &config.Config{
				Defaults: config.Defaults{
					BaseDir: "/test/worktrees",
				},
			}

			// Execute
			err := addCommandWithExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

			// Verify
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			commands := mockExec.GetExecutedCommands()
			if len(tt.expectedArgs) > 0 {
				assert.Len(t, commands, 1)
				assert.Equal(t, tt.expectedArgs, commands[0])
			}
		})
	}
}

// ===== Helper Functions =====

func createTestCLICommand(flags map[string]interface{}, args []string) *cli.Command {
	app := &cli.Command{
		Name: "test",
		Commands: []*cli.Command{
			{
				Name: "add",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "path"},
					&cli.BoolFlag{Name: "force"},
					&cli.BoolFlag{Name: "detach"},
					&cli.StringFlag{Name: "branch"},
					&cli.StringFlag{Name: "track"},
					&cli.BoolFlag{Name: "cd"},
					&cli.BoolFlag{Name: "no-cd"},
				},
				Action: func(_ context.Context, _ *cli.Command) error {
					return nil
				},
			},
		},
	}

	// Build command line args
	cmdArgs := []string{"test", "add"}
	for key, value := range flags {
		switch v := value.(type) {
		case bool:
			if v {
				cmdArgs = append(cmdArgs, "--"+key)
			}
		case string:
			cmdArgs = append(cmdArgs, "--"+key, v)
		}
	}
	cmdArgs = append(cmdArgs, args...)

	// Run the app to populate flags
	ctx := context.Background()
	_ = app.Run(ctx, cmdArgs)

	// Return the add subcommand
	return app.Commands[0]
}

// ===== Mock Implementations =====

type mockCommandExecutor struct {
	executedCommands []command.Command
	shouldFail       bool
	errorMsg         string
}

func (m *mockCommandExecutor) Execute(commands []command.Command) (*command.ExecutionResult, error) {
	m.executedCommands = commands

	if m.shouldFail {
		errorMsg := m.errorMsg
		if errorMsg == "" {
			errorMsg = "mock error"
		}
		return &command.ExecutionResult{
			Results: []command.Result{{
				Command: commands[0],
				Error:   errors.GitCommandFailed("git", errorMsg),
			}},
		}, nil
	}

	results := make([]command.Result, len(commands))
	for i, cmd := range commands {
		results[i] = command.Result{
			Command: cmd,
			Output:  "success",
		}
	}

	return &command.ExecutionResult{Results: results}, nil
}

// ===== Simple Unit Tests (What testing) =====

func TestAddCommand_ExistingBranch(t *testing.T) {
	mockExec := &mockCommandExecutor{shouldFail: false}
	var buf bytes.Buffer
	cmd := createTestCLICommand(map[string]interface{}{}, []string{"feature/auth"})
	cfg := &config.Config{
		Defaults: config.Defaults{BaseDir: "/test/worktrees"},
	}

	err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

	assert.NoError(t, err)
	assert.Len(t, mockExec.executedCommands, 1)
	expectedArgs := []string{"worktree", "add", "/test/worktrees/feature/auth", "feature/auth"}
	assert.Equal(t, expectedArgs, mockExec.executedCommands[0].Args)
	assert.Contains(t, buf.String(), "Created worktree 'feature/auth'")
}

func TestAddCommand_NewBranch(t *testing.T) {
	mockExec := &mockCommandExecutor{shouldFail: false}
	var buf bytes.Buffer
	cmd := createTestCLICommand(map[string]interface{}{"branch": "feature/payment"}, []string{})
	cfg := &config.Config{
		Defaults: config.Defaults{BaseDir: "/test/worktrees"},
	}

	err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

	assert.NoError(t, err)
	assert.Len(t, mockExec.executedCommands, 1)
	expectedArgs := []string{"worktree", "add", "-b", "feature/payment", "/test/worktrees/feature/payment"}
	assert.Equal(t, expectedArgs, mockExec.executedCommands[0].Args)
}

func TestAddCommand_CustomPath(t *testing.T) {
	mockExec := &mockCommandExecutor{shouldFail: false}
	var buf bytes.Buffer
	cmd := createTestCLICommand(map[string]interface{}{"path": "/custom/path"}, []string{"feature/auth"})
	cfg := &config.Config{
		Defaults: config.Defaults{BaseDir: "/test/worktrees"},
	}

	err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

	assert.NoError(t, err)
	expectedArgs := []string{"worktree", "add", "/custom/path", "feature/auth"}
	assert.Equal(t, expectedArgs, mockExec.executedCommands[0].Args)
}

func TestAddCommand_NoBranchError(t *testing.T) {
	cmd := createTestCLICommand(map[string]interface{}{}, []string{})

	err := validateAddInput(cmd)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch name is required")
}

func TestAddCommand_GitError(t *testing.T) {
	mockExec := &mockCommandExecutor{shouldFail: true}
	var buf bytes.Buffer
	cmd := createTestCLICommand(map[string]interface{}{}, []string{"feature/auth"})
	cfg := &config.Config{
		Defaults: config.Defaults{BaseDir: "/test/worktrees"},
	}

	err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

	assert.Error(t, err)
	assert.Len(t, mockExec.executedCommands, 1)
}

// ===== Real-World Edge Cases =====

func TestAddCommand_InternationalCharacters(t *testing.T) {
	tests := []struct {
		name         string
		branchName   string
		expectedPath string
		shouldWork   bool
	}{
		{
			name:         "Japanese characters",
			branchName:   "機能/ログイン",
			expectedPath: "/test/worktrees/機能/ログイン",
			shouldWork:   true,
		},
		{
			name:         "Spanish accents",
			branchName:   "función/añadir",
			expectedPath: "/test/worktrees/función/añadir",
			shouldWork:   true,
		},
		{
			name:         "Emoji characters",
			branchName:   "feature/🚀-rocket",
			expectedPath: "/test/worktrees/feature/🚀-rocket",
			shouldWork:   true,
		},
		{
			name:         "Chinese characters",
			branchName:   "特性/用户认证",
			expectedPath: "/test/worktrees/特性/用户认证",
			shouldWork:   true,
		},
		{
			name:         "Arabic characters",
			branchName:   "ميزة/تسجيل-الدخول",
			expectedPath: "/test/worktrees/ميزة/تسجيل-الدخول",
			shouldWork:   true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockCommandExecutor{shouldFail: !tt.shouldWork}
			var buf bytes.Buffer
			cmd := createTestCLICommand(map[string]interface{}{}, []string{tt.branchName})
			cfg := &config.Config{
				Defaults: config.Defaults{BaseDir: "/test/worktrees"},
			}

			err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

			if tt.shouldWork {
				assert.NoError(t, err)
				assert.Len(t, mockExec.executedCommands, 1)
				assert.Equal(t, []string{"worktree", "add", tt.expectedPath, tt.branchName},
					mockExec.executedCommands[0].Args)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestAddCommand_SpecialCharacters(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		shouldWork bool
		reason     string
	}{
		{
			name:       "spaces in branch name",
			branchName: "feature/with spaces",
			shouldWork: true,
			reason:     "Git supports spaces in branch names",
		},
		{
			name:       "hyphens and underscores",
			branchName: "feature/test-branch_name",
			shouldWork: true,
			reason:     "Common valid characters",
		},
		{
			name:       "dots in branch name",
			branchName: "release/v1.0.0",
			shouldWork: true,
			reason:     "Version branches commonly use dots",
		},
		{
			name:       "numbers and mixed case",
			branchName: "Feature123/TestBranch",
			shouldWork: true,
			reason:     "Mixed case and numbers are valid",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockCommandExecutor{shouldFail: !tt.shouldWork}
			var buf bytes.Buffer
			cmd := createTestCLICommand(map[string]interface{}{}, []string{tt.branchName})
			cfg := &config.Config{
				Defaults: config.Defaults{BaseDir: "/test/worktrees"},
			}

			err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

			if tt.shouldWork {
				assert.NoError(t, err, tt.reason)
			} else {
				assert.Error(t, err, tt.reason)
			}
		})
	}
}

func TestAddCommand_PathLengthLimits(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		shouldWork bool
		reason     string
	}{
		{
			name: "very long branch name",
			branchName: "feature/this-is-a-very-long-branch-name-that-might-cause-issues-" +
				"with-filesystem-path-limits-in-some-operating-systems-especially-windows",
			shouldWork: true,
			reason:     "Modern filesystems should handle long paths",
		},
		{
			name:       "nested branch structure",
			branchName: "team/backend/feature/authentication/oauth2/implementation",
			shouldWork: true,
			reason:     "Deep nesting should be supported",
		},
		{
			name:       "extremely deep nesting",
			branchName: "a/b/c/d/e/f/g/h/i/j/k/l/m/n/o/p/q/r/s/t/u/v/w/x/y/z",
			shouldWork: true,
			reason:     "Test filesystem path depth limits",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockCommandExecutor{shouldFail: !tt.shouldWork}
			var buf bytes.Buffer
			cmd := createTestCLICommand(map[string]interface{}{}, []string{tt.branchName})
			cfg := &config.Config{
				Defaults: config.Defaults{BaseDir: "/test/worktrees"},
			}

			err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

			if tt.shouldWork {
				assert.NoError(t, err, tt.reason)
				assert.Len(t, mockExec.executedCommands, 1)
				// Verify the path is constructed correctly
				expectedPath := "/test/worktrees/" + tt.branchName
				assert.Equal(t, []string{"worktree", "add", expectedPath, tt.branchName},
					mockExec.executedCommands[0].Args)
			} else {
				assert.Error(t, err, tt.reason)
			}
		})
	}
}

func TestAddCommand_FlagCombinations(t *testing.T) {
	tests := []struct {
		name        string
		flags       map[string]interface{}
		args        []string
		expectError bool
		description string
	}{
		{
			name:        "branch and path combination",
			flags:       map[string]interface{}{"branch": "new-feature", "path": "/custom/path"},
			args:        []string{},
			expectError: false,
			description: "Valid: creating new branch at custom path",
		},
		{
			name:        "detach with commit hash",
			flags:       map[string]interface{}{"detach": true},
			args:        []string{"abc123"},
			expectError: false,
			description: "Valid: detached HEAD at specific commit",
		},
		{
			name:        "force flag with existing path",
			flags:       map[string]interface{}{"force": true},
			args:        []string{"feature/existing"},
			expectError: false,
			description: "Valid: force creation even if path exists",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockCommandExecutor{shouldFail: false}
			var buf bytes.Buffer
			cmd := createTestCLICommand(tt.flags, tt.args)
			cfg := &config.Config{
				Defaults: config.Defaults{BaseDir: "/test/worktrees"},
			}

			err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

			if tt.expectError {
				assert.Error(t, err, tt.description)
			} else {
				assert.NoError(t, err, tt.description)
			}
		})
	}
}

// ===== Cross-Platform and Locale Tests =====

func TestAddCommand_CrossPlatformPaths(t *testing.T) {
	tests := []struct {
		name           string
		branchName     string
		baseDirUnix    string
		baseDirWindows string
		expectedUnix   string
		expectedWin    string
		description    string
	}{
		{
			name:           "standard branch with unix separators",
			branchName:     "feature/auth",
			baseDirUnix:    "/home/user/worktrees",
			baseDirWindows: "C:\\Users\\user\\worktrees",
			expectedUnix:   "/home/user/worktrees/feature/auth",
			expectedWin:    "C:\\Users\\user\\worktrees\\feature\\auth",
			description:    "Should handle path separators correctly on different platforms",
		},
		{
			name:           "deeply nested branch structure",
			branchName:     "team/backend/feature/auth",
			baseDirUnix:    "/tmp/worktrees",
			baseDirWindows: "C:\\temp\\worktrees",
			expectedUnix:   "/tmp/worktrees/team/backend/feature/auth",
			expectedWin:    "C:\\temp\\worktrees\\team\\backend\\feature\\auth",
			description:    "Should handle deep nesting across platforms",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test Unix-style paths
			t.Run("unix_paths", func(t *testing.T) {
				mockExec := &mockCommandExecutor{shouldFail: false}
				var buf bytes.Buffer
				cmd := createTestCLICommand(map[string]interface{}{}, []string{tt.branchName})
				cfg := &config.Config{
					Defaults: config.Defaults{BaseDir: tt.baseDirUnix},
				}

				err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

				assert.NoError(t, err)
				assert.Contains(t, buf.String(), fmt.Sprintf("Created worktree '%s'", tt.branchName))
				assert.Equal(t, []string{"worktree", "add", tt.expectedUnix, tt.branchName},
					mockExec.executedCommands[0].Args)
			})

			// Test Windows-style paths (simulated)
			t.Run("windows_paths", func(t *testing.T) {
				mockExec := &mockCommandExecutor{shouldFail: false}
				var buf bytes.Buffer
				cmd := createTestCLICommand(map[string]interface{}{}, []string{tt.branchName})
				cfg := &config.Config{
					Defaults: config.Defaults{BaseDir: tt.baseDirWindows},
				}

				err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

				assert.NoError(t, err)
				assert.Contains(t, buf.String(), fmt.Sprintf("Created worktree '%s'", tt.branchName))
				// Note: On Unix systems, Windows-style paths are treated as relative paths
				// and will be resolved relative to the repo path
				expectedPath := fmt.Sprintf("/test/repo/%s/%s", tt.baseDirWindows, tt.branchName)
				assert.Equal(t, []string{"worktree", "add", expectedPath, tt.branchName},
					mockExec.executedCommands[0].Args)
			})
		})
	}
}

func TestAddCommand_ErrorMessageQuality(t *testing.T) {
	tests := []struct {
		name              string
		branchName        string
		flags             map[string]interface{}
		mockError         string
		expectedInMessage []string
		description       string
	}{
		{
			name:       "branch already exists error",
			branchName: "feature/existing",
			flags:      map[string]interface{}{},
			mockError:  "fatal: a branch named 'feature/existing' already exists",
			expectedInMessage: []string{
				"branch", "existing", "feature/existing",
			},
			description: "Error should include the problematic branch name",
		},
		{
			name:       "invalid path characters",
			branchName: "feature/invalid:path",
			flags:      map[string]interface{}{},
			mockError:  "fatal: invalid path 'feature/invalid:path'",
			expectedInMessage: []string{
				"invalid", "path", "feature/invalid:path",
			},
			description: "Error should highlight the invalid characters",
		},
		{
			name:       "permission denied",
			branchName: "feature/auth",
			flags:      map[string]interface{}{},
			mockError:  "fatal: could not create work tree dir '/restricted/path'",
			expectedInMessage: []string{
				"could not create", "work tree", "restricted",
			},
			description: "Error should indicate permission issues clearly",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockCommandExecutor{
				shouldFail: true,
				errorMsg:   tt.mockError,
			}
			var buf bytes.Buffer
			cmd := createTestCLICommand(tt.flags, []string{tt.branchName})
			cfg := &config.Config{
				Defaults: config.Defaults{BaseDir: "/test/worktrees"},
			}

			err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

			assert.Error(t, err)
			errorOutput := err.Error() + buf.String()
			
			// Check that error message contains relevant information
			for _, expectedText := range tt.expectedInMessage {
				assert.Contains(t, strings.ToLower(errorOutput), 
					strings.ToLower(expectedText),
					"Error message should contain '%s' for better user guidance", expectedText)
			}
		})
	}
}
