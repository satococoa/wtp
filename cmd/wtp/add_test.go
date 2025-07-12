package main

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"

	"github.com/satococoa/wtp/internal/command"
	"github.com/satococoa/wtp/internal/config"
	"github.com/satococoa/wtp/internal/errors"
)

// ===== Command Structure Tests =====

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

// ===== Error Type Tests =====

func TestWorkTreeAlreadyExistsError(t *testing.T) {
	t.Run("should format error message with branch name and solutions", func(t *testing.T) {
		// Given: a WorktreeAlreadyExistsError with branch and git error
		originalErr := &MockGitError{msg: "branch already checked out"}
		err := &WorktreeAlreadyExistsError{
			BranchName: "feature/awesome",
			Path:       "/path/to/worktree",
			GitError:   originalErr,
		}

		// When: getting error message
		message := err.Error()

		// Then: should contain branch name, solutions, and original error
		assert.Contains(t, message, "feature/awesome")
		assert.Contains(t, message, "already checked out in another worktree")
		assert.Contains(t, message, "--force")
		assert.Contains(t, message, "Choose a different branch")
		assert.Contains(t, message, "Remove the existing worktree")
		assert.Contains(t, message, "branch already checked out")
	})

	t.Run("should handle empty branch name", func(t *testing.T) {
		// Given: error with empty branch name
		err := &WorktreeAlreadyExistsError{
			BranchName: "",
			Path:       "/path/to/worktree",
			GitError:   &MockGitError{msg: "test error"},
		}

		// When: getting error message
		message := err.Error()

		// Then: should still provide valid message
		assert.Contains(t, message, "worktree for branch ''")
		assert.Contains(t, message, "test error")
	})
}

func TestPathAlreadyExistsError(t *testing.T) {
	t.Run("should format error message with path and solutions", func(t *testing.T) {
		// Given: a PathAlreadyExistsError with path and git error
		originalErr := &MockGitError{msg: "directory not empty"}
		err := &PathAlreadyExistsError{
			Path:     "/existing/path",
			GitError: originalErr,
		}

		// When: getting error message
		message := err.Error()

		// Then: should contain path, solutions, and original error
		assert.Contains(t, message, "/existing/path")
		assert.Contains(t, message, "already exists and is not empty")
		assert.Contains(t, message, "--force flag")
		assert.Contains(t, message, "--path flag")
		assert.Contains(t, message, "Remove the existing directory")
		assert.Contains(t, message, "directory not empty")
	})

	t.Run("should handle empty path", func(t *testing.T) {
		// Given: error with empty path
		err := &PathAlreadyExistsError{
			Path:     "",
			GitError: &MockGitError{msg: "test error"},
		}

		// When: getting error message
		message := err.Error()

		// Then: should still provide valid message
		assert.Contains(t, message, "destination path already exists:")
		assert.Contains(t, message, "test error")
	})
}

func TestMultipleBranchesError(t *testing.T) {
	t.Run("should format error message with branch name and track suggestions", func(t *testing.T) {
		// Given: a MultipleBranchesError with branch name
		originalErr := &MockGitError{msg: "multiple remotes found"}
		err := &MultipleBranchesError{
			BranchName: "feature/shared",
			GitError:   originalErr,
		}

		// When: getting error message
		message := err.Error()

		// Then: should contain branch name, track suggestions, and original error
		assert.Contains(t, message, "feature/shared")
		assert.Contains(t, message, "exists in multiple remotes")
		assert.Contains(t, message, "--track origin/feature/shared")
		assert.Contains(t, message, "--track upstream/feature/shared")
		assert.Contains(t, message, "multiple remotes found")
	})

	t.Run("should handle special characters in branch name", func(t *testing.T) {
		// Given: error with special characters in branch name
		err := &MultipleBranchesError{
			BranchName: "feature/fix-bugs-#123",
			GitError:   &MockGitError{msg: "test error"},
		}

		// When: getting error message
		message := err.Error()

		// Then: should properly format all instances of branch name
		assert.Contains(t, message, "feature/fix-bugs-#123")
		assert.Contains(t, message, "--track origin/feature/fix-bugs-#123")
		assert.Contains(t, message, "--track upstream/feature/fix-bugs-#123")
	})
}

// Mock error for testing
type MockGitError struct {
	msg string
}

func (e *MockGitError) Error() string {
	return e.msg
}

// ===== Helper Function Tests =====

func TestSetupRepoAndConfig(t *testing.T) {
	t.Run("should setup repository and config from current directory", func(t *testing.T) {
		// Given: we are in a git repository with config
		// When: setting up repo and config
		repo, cfg, mainRepoPath, err := setupRepoAndConfig()

		// Then: should return valid repo, config, and paths
		// Note: This test requires being in a git repository
		if err != nil {
			// Skip test if not in git repo - this is expected in some environments
			t.Skip("Not in a git repository - skipping test")
		}

		assert.NotNil(t, repo)
		assert.NotNil(t, cfg)
		assert.NotEmpty(t, mainRepoPath)
	})
}

func TestChangeToWorktree(t *testing.T) {
	t.Run("should output cd command when shell integration disabled", func(t *testing.T) {
		// Given: shell integration is disabled
		oldEnv := os.Getenv("WTP_SHELL_INTEGRATION")
		defer func() {
			if oldEnv != "" {
				os.Setenv("WTP_SHELL_INTEGRATION", oldEnv)
			} else {
				os.Unsetenv("WTP_SHELL_INTEGRATION")
			}
		}()
		os.Unsetenv("WTP_SHELL_INTEGRATION")

		var buf bytes.Buffer
		workTreePath := "/path/to/worktree"

		// When: changing to worktree
		changeToWorktree(&buf, workTreePath)

		// Then: should output cd command and integration hint
		output := buf.String()
		assert.Contains(t, output, "cd /path/to/worktree")
		assert.Contains(t, output, "Enable shell integration")
	})

	t.Run("should output path directly when shell integration enabled", func(t *testing.T) {
		// Given: shell integration is enabled
		oldEnv := os.Getenv("WTP_SHELL_INTEGRATION")
		defer func() {
			if oldEnv != "" {
				os.Setenv("WTP_SHELL_INTEGRATION", oldEnv)
			} else {
				os.Unsetenv("WTP_SHELL_INTEGRATION")
			}
		}()
		os.Setenv("WTP_SHELL_INTEGRATION", "1")

		var buf bytes.Buffer
		workTreePath := "/path/to/worktree"

		// When: changing to worktree
		changeToWorktree(&buf, workTreePath)

		// Then: should output only the path
		output := buf.String()
		assert.Equal(t, "/path/to/worktree", output)
		assert.NotContains(t, output, "cd")
		assert.NotContains(t, output, "Enable shell integration")
	})

	t.Run("should handle paths with spaces", func(t *testing.T) {
		// Given: shell integration is disabled and path has spaces
		oldEnv := os.Getenv("WTP_SHELL_INTEGRATION")
		defer func() {
			if oldEnv != "" {
				os.Setenv("WTP_SHELL_INTEGRATION", oldEnv)
			} else {
				os.Unsetenv("WTP_SHELL_INTEGRATION")
			}
		}()
		os.Unsetenv("WTP_SHELL_INTEGRATION")

		var buf bytes.Buffer
		workTreePath := "/path/to/my worktree"

		// When: changing to worktree
		changeToWorktree(&buf, workTreePath)

		// Then: should output path correctly
		output := buf.String()
		assert.Contains(t, output, "cd /path/to/my worktree")
	})
}

func TestAppendExplicitPathArgs(t *testing.T) {
	t.Run("should append branch name when no flags are provided", func(t *testing.T) {
		// Given: no branch or track flags, and a branch name
		initialArgs := []string{"worktree", "add", "/path/to/worktree"}
		cmdArgs := []string{"some-branch"}

		// When: appending explicit path args
		result := callAppendExplicitPathArgs(initialArgs, cmdArgs, "", "", "some-branch")

		// Then: should append the branch name
		expected := []string{"worktree", "add", "/path/to/worktree", "some-branch"}
		assert.Equal(t, expected, result)
	})

	t.Run("should not append branch name when -b flag is used", func(t *testing.T) {
		// Given: branch flag is provided
		initialArgs := []string{"worktree", "add", "-b", "new-branch", "/path/to/worktree"}
		cmdArgs := []string{"base-branch"}

		// When: appending explicit path args with branch flag
		result := callAppendExplicitPathArgs(initialArgs, cmdArgs, "new-branch", "", "base-branch")

		// Then: should not append branch name (already handled by -b flag)
		expected := []string{"worktree", "add", "-b", "new-branch", "/path/to/worktree"}
		assert.Equal(t, expected, result)
	})

	t.Run("should append remote branch when track flag is used without -b", func(t *testing.T) {
		// Given: track flag is provided without branch flag
		initialArgs := []string{"worktree", "add", "--track", "/path/to/worktree"}
		cmdArgs := []string{"feature-branch"}

		// When: appending explicit path args with track
		result := callAppendExplicitPathArgs(initialArgs, cmdArgs, "", "origin/feature-branch", "feature-branch")

		// Then: should append the remote branch
		expected := []string{"worktree", "add", "--track", "/path/to/worktree", "origin/feature-branch"}
		assert.Equal(t, expected, result)
	})

	t.Run("should append additional command arguments", func(t *testing.T) {
		// Given: command has additional arguments beyond the first (commit-ish)
		initialArgs := []string{"worktree", "add", "/path/to/worktree"}
		cmdArgs := []string{"branch-name", "commit-hash"}

		// When: appending explicit path args
		result := callAppendExplicitPathArgs(initialArgs, cmdArgs, "", "", "branch-name")

		// Then: should append branch name and additional args
		expected := []string{"worktree", "add", "/path/to/worktree", "branch-name", "commit-hash"}
		assert.Equal(t, expected, result)
	})

	t.Run("should handle empty initial args", func(t *testing.T) {
		// Given: empty initial args array
		initialArgs := []string{}
		cmdArgs := []string{"branch-name"}

		// When: appending explicit path args
		result := callAppendExplicitPathArgs(initialArgs, cmdArgs, "", "", "branch-name")

		// Then: should create new array with branch name
		expected := []string{"branch-name"}
		assert.Equal(t, expected, result)
	})

	t.Run("should handle command with only one argument", func(t *testing.T) {
		// Given: command with only branch name argument
		initialArgs := []string{"worktree", "add", "/path/to/worktree"}
		cmdArgs := []string{"branch-name"}

		// When: appending explicit path args
		result := callAppendExplicitPathArgs(initialArgs, cmdArgs, "", "", "branch-name")

		// Then: should only append branch name (no additional args)
		expected := []string{"worktree", "add", "/path/to/worktree", "branch-name"}
		assert.Equal(t, expected, result)
	})
}

// For testing appendExplicitPathArgs, we need to create a CLI command
// since the function needs a cli.Command interface
func callAppendExplicitPathArgs(initialArgs, cmdArgs []string, branch, track, branchName string) []string {
	// Create a simple CLI command with the args we need
	cmd := createTestCLICommandWithArgs(map[string]interface{}{
		"branch": branch,
		"track":  track,
	}, cmdArgs)
	return appendExplicitPathArgs(initialArgs, cmd, branch, track, branchName)
}

// Helper to create CLI command with specific args
func createTestCLICommandWithArgs(flags map[string]interface{}, args []string) *cli.Command {
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

	cmdArgs := []string{"test", "add"}
	for key, value := range flags {
		switch v := value.(type) {
		case bool:
			if v {
				cmdArgs = append(cmdArgs, "--"+key)
			}
		case string:
			if v != "" {
				cmdArgs = append(cmdArgs, "--"+key, v)
			}
		}
	}
	cmdArgs = append(cmdArgs, args...)

	ctx := context.Background()
	_ = app.Run(ctx, cmdArgs)

	return app.Commands[0]
}

// ===== Pure Business Logic Tests =====

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
			app := &cli.Command{
				Name: "test",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "branch"},
				},
				Action: func(_ context.Context, cmd *cli.Command) error {
					return validateAddInput(cmd)
				},
			}

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

func TestResolveWorktreePath(t *testing.T) {
	tests := []struct {
		name         string
		pathFlag     string
		branchName   string
		baseDir      string
		expectedPath string
	}{
		{
			name:         "default path from branch name",
			pathFlag:     "",
			branchName:   "feature/auth",
			baseDir:      "/test/worktrees",
			expectedPath: "/test/worktrees/feature/auth",
		},
		{
			name:         "explicit path overrides default",
			pathFlag:     "/custom/path",
			branchName:   "feature/auth",
			baseDir:      "/test/worktrees",
			expectedPath: "/custom/path",
		},
		{
			name:         "branch with nested structure",
			pathFlag:     "",
			branchName:   "team/backend/feature",
			baseDir:      "/worktrees",
			expectedPath: "/worktrees/team/backend/feature",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Defaults: config.Defaults{BaseDir: tt.baseDir},
			}
			cmd := createTestCLICommand(map[string]interface{}{
				"path": tt.pathFlag,
			}, []string{tt.branchName})

			path, _ := resolveWorktreePath(cfg, "/test/repo", tt.branchName, cmd)
			assert.Equal(t, tt.expectedPath, path)
		})
	}
}

// ===== Command Building Tests =====

func TestBuildGitWorktreeArgs(t *testing.T) {
	tests := []struct {
		name         string
		workTreePath string
		branchName   string
		flags        map[string]interface{}
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
			want:         []string{"worktree", "add", "--detach", "/path/to/worktree"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

			ctx := context.Background()
			err := app.Run(ctx, args)
			assert.NoError(t, err)
		})
	}
}

// ===== Command Execution Tests =====

func TestAddCommand_CommandConstruction(t *testing.T) {
	tests := []struct {
		name             string
		flags            map[string]interface{}
		args             []string
		expectedCommands []command.Command
		expectError      bool
	}{
		{
			name: "basic worktree creation",
			flags: map[string]interface{}{
				"branch": "feature/test",
			},
			args: []string{"feature/test"},
			expectedCommands: []command.Command{{
				Name: "git",
				Args: []string{"worktree", "add", "-b", "feature/test", "/test/worktrees/feature/test"},
			}},
			expectError: false,
		},
		{
			name: "worktree with force flag",
			flags: map[string]interface{}{
				"force":  true,
				"branch": "feature/test",
			},
			args: []string{"feature/test"},
			expectedCommands: []command.Command{{
				Name: "git",
				Args: []string{"worktree", "add", "--force", "-b", "feature/test", "/test/worktrees/feature/test"},
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
			expectError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := createTestCLICommand(tt.flags, tt.args)
			var buf bytes.Buffer
			mockExec := &mockCommandExecutor{}

			cfg := &config.Config{
				Defaults: config.Defaults{
					BaseDir: "/test/worktrees",
				},
			}

			err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				// Verify that the correct git command was constructed
				assert.Equal(t, tt.expectedCommands, mockExec.executedCommands)
			}
		})
	}
}

func TestAddCommand_SuccessMessage(t *testing.T) {
	tests := []struct {
		name           string
		branchName     string
		expectedOutput string
	}{
		{
			name:           "with branch name",
			branchName:     "feature/auth",
			expectedOutput: "Created worktree 'feature/auth'",
		},
		{
			name:           "new branch",
			branchName:     "new-feature",
			expectedOutput: "Created worktree 'new-feature'",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := createTestCLICommand(map[string]interface{}{"branch": tt.branchName}, []string{tt.branchName})
			var buf bytes.Buffer
			mockExec := &mockCommandExecutor{}

			cfg := &config.Config{
				Defaults: config.Defaults{BaseDir: "/test/worktrees"},
			}

			err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

			assert.NoError(t, err)
			assert.Contains(t, buf.String(), tt.expectedOutput)
		})
	}
}

// ===== Error Handling Tests =====

func TestAddCommand_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		flags         map[string]interface{}
		args          []string
		expectedError string
	}{
		{
			name:          "no branch name",
			flags:         map[string]interface{}{},
			args:          []string{},
			expectedError: "branch name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := createTestCLICommand(tt.flags, tt.args)
			err := validateAddInput(cmd)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestAddCommand_ExecutionError(t *testing.T) {
	mockExec := &mockCommandExecutor{shouldFail: true}
	var buf bytes.Buffer
	cmd := createTestCLICommand(map[string]interface{}{"branch": "feature/auth"}, []string{"feature/auth"})
	cfg := &config.Config{
		Defaults: config.Defaults{BaseDir: "/test/worktrees"},
	}

	err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

	assert.Error(t, err)
	assert.Len(t, mockExec.executedCommands, 1)
}

// ===== Edge Cases Tests =====

func TestAddCommand_InternationalCharacters(t *testing.T) {
	tests := []struct {
		name         string
		branchName   string
		expectedPath string
	}{
		{
			name:         "Japanese characters",
			branchName:   "機能/ログイン",
			expectedPath: "/test/worktrees/機能/ログイン",
		},
		{
			name:         "Spanish accents",
			branchName:   "función/añadir",
			expectedPath: "/test/worktrees/función/añadir",
		},
		{
			name:         "Emoji characters",
			branchName:   "feature/🚀-rocket",
			expectedPath: "/test/worktrees/feature/🚀-rocket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockCommandExecutor{}
			var buf bytes.Buffer
			cmd := createTestCLICommand(map[string]interface{}{"branch": tt.branchName}, []string{tt.branchName})
			cfg := &config.Config{
				Defaults: config.Defaults{BaseDir: "/test/worktrees"},
			}

			err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

			assert.NoError(t, err)
			assert.Len(t, mockExec.executedCommands, 1)
			assert.Equal(t, []string{"worktree", "add", "-b", tt.branchName, tt.expectedPath},
				mockExec.executedCommands[0].Args)
		})
	}
}

func TestAddCommand_PathHandling(t *testing.T) {
	tests := []struct {
		name         string
		branchName   string
		customPath   string
		expectedPath string
	}{
		{
			name:         "spaces in branch name",
			branchName:   "feature/with spaces",
			expectedPath: "/test/worktrees/feature/with spaces",
		},
		{
			name:         "custom path overrides default",
			branchName:   "feature/auth",
			customPath:   "/custom/path",
			expectedPath: "/custom/path",
		},
		{
			name:         "deeply nested branch",
			branchName:   "team/backend/feature/auth",
			expectedPath: "/test/worktrees/team/backend/feature/auth",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			flags := map[string]interface{}{
				"branch": tt.branchName,
			}
			if tt.customPath != "" {
				flags["path"] = tt.customPath
			}

			mockExec := &mockCommandExecutor{}
			var buf bytes.Buffer
			cmd := createTestCLICommand(flags, []string{tt.branchName})
			cfg := &config.Config{
				Defaults: config.Defaults{BaseDir: "/test/worktrees"},
			}

			err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

			assert.NoError(t, err)
			expectedArgs := []string{"worktree", "add", "-b", tt.branchName, tt.expectedPath}
			assert.Equal(t, expectedArgs, mockExec.executedCommands[0].Args)
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

	ctx := context.Background()
	_ = app.Run(ctx, cmdArgs)

	return app.Commands[0]
}

// ===== Integration Tests =====

func TestAddCommand_Integration(t *testing.T) {
	t.Run("should coordinate all components successfully", func(t *testing.T) {
		// Given: a CLI command with proper setup
		app := &cli.Command{
			Name: "test",
			Commands: []*cli.Command{
				{
					Name: "add",
					Flags: []cli.Flag{
						&cli.StringFlag{Name: "path"},
						&cli.BoolFlag{Name: "force"},
						&cli.BoolFlag{Name: "detach"},
						&cli.StringFlag{Name: "branch", Aliases: []string{"b"}},
						&cli.StringFlag{Name: "track", Aliases: []string{"t"}},
						&cli.BoolFlag{Name: "cd"},
						&cli.BoolFlag{Name: "no-cd"},
					},
					Action: addCommand,
				},
			},
		}

		// When: running add command with nonexistent branch
		ctx := context.Background()
		err := app.Run(ctx, []string{"test", "add", "nonexistent-test-branch"})

		// Then: should return appropriate error for branch not found
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("should handle validation errors gracefully", func(t *testing.T) {
		// Given: a CLI command with no arguments
		app := &cli.Command{
			Name: "test",
			Commands: []*cli.Command{
				{
					Name: "add",
					Flags: []cli.Flag{
						&cli.StringFlag{Name: "path"},
						&cli.BoolFlag{Name: "force"},
						&cli.BoolFlag{Name: "detach"},
						&cli.StringFlag{Name: "branch", Aliases: []string{"b"}},
						&cli.StringFlag{Name: "track", Aliases: []string{"t"}},
						&cli.BoolFlag{Name: "cd"},
						&cli.BoolFlag{Name: "no-cd"},
					},
					Action: addCommand,
				},
			},
		}

		// When: running add command without branch name
		ctx := context.Background()
		err := app.Run(ctx, []string{"test", "add"})

		// Then: should return validation error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "branch name is required")
	})

	t.Run("should handle conflicting flags", func(t *testing.T) {
		// Given: a CLI command with conflicting flags
		app := &cli.Command{
			Name: "test",
			Commands: []*cli.Command{
				{
					Name: "add",
					Flags: []cli.Flag{
						&cli.StringFlag{Name: "path"},
						&cli.BoolFlag{Name: "force"},
						&cli.BoolFlag{Name: "detach"},
						&cli.StringFlag{Name: "branch", Aliases: []string{"b"}},
						&cli.StringFlag{Name: "track", Aliases: []string{"t"}},
						&cli.BoolFlag{Name: "cd"},
						&cli.BoolFlag{Name: "no-cd"},
					},
					Action: addCommand,
				},
			},
		}

		// When: running add command with both -b and --detach
		ctx := context.Background()
		err := app.Run(ctx, []string{"test", "add", "--branch", "new-branch", "--detach", "some-commit"})

		// Then: should return conflict error
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "conflicting flags")
	})
}

func TestExecutePostCreateHooks_Integration(t *testing.T) {
	t.Run("should handle hooks when config has no hooks", func(t *testing.T) {
		// Given: a config with no hooks
		cfg := &config.Config{}
		var buf bytes.Buffer

		// When: executing post create hooks
		err := executePostCreateHooks(&buf, cfg, "/test/repo", "/test/worktree")

		// Then: should complete without error and no output
		assert.NoError(t, err)
		assert.Empty(t, buf.String())
	})

	t.Run("should handle hook execution errors", func(t *testing.T) {
		// Given: a config with hooks that might fail
		cfg := &config.Config{
			Hooks: config.Hooks{
				PostCreate: []config.Hook{
					{Type: "command", Command: "nonexistent-command-xyz test"},
				},
			},
		}
		var buf bytes.Buffer

		// When: executing post create hooks
		err := executePostCreateHooks(&buf, cfg, "/test/repo", "/test/worktree")

		// Then: should return error for failed hook execution
		// This tests the error handling path in executePostCreateHooks
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to execute hook")
		assert.Contains(t, buf.String(), "Executing post-create hooks")
	})
}

func TestShouldChangeDirectory_Integration(t *testing.T) {
	t.Run("should respect command flags over config", func(t *testing.T) {
		// Given: config with CDAfterCreate=true but command has --no-cd
		cfg := &config.Config{
			Defaults: config.Defaults{CDAfterCreate: true},
		}
		cmd := createTestCLICommand(map[string]interface{}{"no-cd": true}, []string{"test-branch"})

		// When: checking if should change directory
		result := shouldChangeDirectory(cmd, cfg)

		// Then: command flag should override config
		assert.False(t, result)
	})

	t.Run("should use config default when no flags", func(t *testing.T) {
		// Given: config with CDAfterCreate=true and no command flags
		cfg := &config.Config{
			Defaults: config.Defaults{CDAfterCreate: true},
		}
		cmd := createTestCLICommand(map[string]interface{}{}, []string{"test-branch"})

		// When: checking if should change directory
		result := shouldChangeDirectory(cmd, cfg)

		// Then: should use config default
		assert.True(t, result)
	})
}

func TestDisplaySuccessMessage_Integration(t *testing.T) {
	t.Run("should format message with branch name", func(t *testing.T) {
		// Given: a buffer and branch name
		var buf bytes.Buffer
		branchName := "feature/awesome"
		workTreePath := "/path/to/worktree"

		// When: displaying success message
		displaySuccessMessage(&buf, branchName, workTreePath)

		// Then: should format correctly
		output := buf.String()
		assert.Contains(t, output, "Created worktree 'feature/awesome'")
		assert.Contains(t, output, "/path/to/worktree")
	})

	t.Run("should handle empty branch name", func(t *testing.T) {
		// Given: a buffer and no branch name
		var buf bytes.Buffer
		branchName := ""
		workTreePath := "/path/to/worktree"

		// When: displaying success message
		displaySuccessMessage(&buf, branchName, workTreePath)

		// Then: should format correctly without branch name
		output := buf.String()
		assert.Contains(t, output, "Created worktree at")
		assert.Contains(t, output, "/path/to/worktree")
		assert.NotContains(t, output, "''")
	})
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
