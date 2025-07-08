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
}

func (m *mockCommandExecutor) Execute(commands []command.Command) (*command.ExecutionResult, error) {
	m.executedCommands = commands

	if m.shouldFail {
		return &command.ExecutionResult{
			Results: []command.Result{{
				Command: commands[0],
				Error:   errors.GitCommandFailed("git", "mock error"),
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

// ===== Living Specifications: Core User Workflows =====

// TestUserCreatesWorktree_WithExistingLocalBranch_ShouldCreateWorktreeAtDefaultPath tests
// the most common user workflow: creating a worktree for an existing local branch.
//
// User Story: As a developer working on a feature branch, I want to create a worktree
// for an existing branch so I can quickly switch to working on that feature in isolation.
//
// Business Value: This eliminates the need to stash changes or commit incomplete work
// when switching between features, improving developer productivity.
func TestUserCreatesWorktree_WithExistingLocalBranch_ShouldCreateWorktreeAtDefaultPath(t *testing.T) {
	// Given: User has an existing local branch named "feature/auth"
	// And: User is in a git repository
	// And: No worktree conflicts exist
	mockExec := &mockCommandExecutor{
		shouldFail: false,
	}
	
	var buf bytes.Buffer
	cmd := createTestCLICommand(map[string]interface{}{}, []string{"feature/auth"})
	cfg := &config.Config{
		Defaults: config.Defaults{
			BaseDir: "/test/worktrees",
		},
	}
	
	// When: User runs "wtp add feature/auth"
	err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")
	
	// Then: Worktree should be created successfully
	assert.NoError(t, err)
	
	// And: Git worktree add command should be executed
	assert.Len(t, mockExec.executedCommands, 1)
	expectedArgs := []string{"worktree", "add", "/test/worktrees/feature/auth", "feature/auth"}
	assert.Equal(t, expectedArgs, mockExec.executedCommands[0].Args)
	
	// And: User should see success confirmation
	output := buf.String()
	assert.Contains(t, output, "Created worktree 'feature/auth'")
}

// TestUserCreatesWorktree_WithNewBranchFlag_ShouldCreateBranchAndWorktree tests
// creating a new branch and worktree simultaneously.
//
// User Story: As a developer starting a new feature, I want to create both a new branch
// and its worktree in one command so I can immediately start working on the feature.
//
// Business Value: Streamlines the workflow of starting new features by combining
// branch creation and worktree setup into a single operation.
func TestUserCreatesWorktree_WithNewBranchFlag_ShouldCreateBranchAndWorktree(t *testing.T) {
	// Given: User wants to create a new branch "feature/payment"
	// And: User is in a git repository
	// And: Branch does not exist yet
	mockExec := &mockCommandExecutor{
		shouldFail: false,
	}
	
	var buf bytes.Buffer
	cmd := createTestCLICommand(map[string]interface{}{"branch": "feature/payment"}, []string{})
	cfg := &config.Config{
		Defaults: config.Defaults{
			BaseDir: "/test/worktrees",
		},
	}
	
	// When: User runs "wtp add --branch feature/payment"
	err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")
	
	// Then: New branch and worktree should be created
	assert.NoError(t, err)
	
	// And: Git should create a new branch with the worktree
	assert.Len(t, mockExec.executedCommands, 1)
	expectedArgs := []string{"worktree", "add", "-b", "feature/payment", "/test/worktrees/feature/payment"}
	assert.Equal(t, expectedArgs, mockExec.executedCommands[0].Args)
	
	// And: User should see success confirmation
	output := buf.String()
	assert.Contains(t, output, "Created worktree 'feature/payment'")
}

// TestUserCreatesWorktree_WithCustomPath_ShouldCreateAtSpecifiedLocation tests
// the flexibility to specify custom worktree locations.
//
// User Story: As a developer with specific project organization needs, I want to
// specify exactly where my worktree should be created so it fits my workflow.
//
// Business Value: Provides flexibility for different team workflows and project
// structures, accommodating various developer preferences and constraints.
func TestUserCreatesWorktree_WithCustomPath_ShouldCreateAtSpecifiedLocation(t *testing.T) {
	// Given: User wants to create worktree at a specific path
	// And: User specifies both path and branch
	mockExec := &mockCommandExecutor{
		shouldFail: false,
	}
	
	var buf bytes.Buffer
	cmd := createTestCLICommand(map[string]interface{}{"path": "/custom/path"}, []string{"feature/auth"})
	cfg := &config.Config{
		Defaults: config.Defaults{
			BaseDir: "/test/worktrees",
		},
	}
	
	// When: User runs "wtp add --path /custom/path feature/auth"
	err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")
	
	// Then: Worktree should be created at the specified path
	assert.NoError(t, err)
	
	// And: Git should use the custom path
	assert.Len(t, mockExec.executedCommands, 1)
	expectedArgs := []string{"worktree", "add", "/custom/path", "feature/auth"}
	assert.Equal(t, expectedArgs, mockExec.executedCommands[0].Args)
	
	// And: User should see success confirmation
	output := buf.String()
	assert.Contains(t, output, "Created worktree 'feature/auth'")
}

// TestUserCreatesWorktree_WithoutBranchName_ShouldShowBranchRequiredError tests
// input validation from the user's perspective.
//
// User Story: As a developer, when I forget to specify a branch name, I want to
// receive a clear error message so I understand what's required.
//
// Business Value: Clear error messages reduce frustration and improve the user
// experience by guiding users toward correct usage.
func TestUserCreatesWorktree_WithoutBranchName_ShouldShowBranchRequiredError(t *testing.T) {
	// Given: User is in a git repository
	// And: User doesn't specify a branch name or --branch flag
	cmd := createTestCLICommand(map[string]interface{}{}, []string{})
	
	// When: User runs "wtp add" with no arguments
	// First validate input as the main command would
	err := validateAddInput(cmd)
	
	// Then: User should receive a clear error message
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch name is required")
}

// TestUserCreatesWorktree_WhenPathAlreadyExists_ShouldRequireForceFlag tests
// conflict resolution from the user's perspective.
//
// User Story: As a developer, when I try to create a worktree where a directory
// already exists, I want to be warned and given the option to force overwrite.
//
// Business Value: Prevents accidental data loss while providing flexibility for
// experienced users who want to overwrite existing directories.
func TestUserCreatesWorktree_WhenPathAlreadyExists_ShouldRequireForceFlag(t *testing.T) {
	// Given: Directory already exists at the target path
	// And: User tries to create worktree without force flag
	mockExec := &mockCommandExecutor{
		shouldFail: true,
	}
	
	var buf bytes.Buffer
	cmd := createTestCLICommand(map[string]interface{}{}, []string{"feature/auth"})
	cfg := &config.Config{
		Defaults: config.Defaults{
			BaseDir: "/test/worktrees",
		},
	}
	
	// When: User runs "wtp add feature/auth" and path exists
	err := addCommandWithCommandExecutor(cmd, &buf, mockExec, cfg, "/test/repo")
	
	// Then: User should receive an error about the conflict
	assert.Error(t, err)
	
	// And: Command was attempted but failed
	assert.Len(t, mockExec.executedCommands, 1)
}
