package main

import (
	"bytes"
	"context"
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
