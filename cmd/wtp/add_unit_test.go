package main

import (
	"bytes"
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"

	"github.com/satococoa/wtp/internal/config"
	"github.com/satococoa/wtp/internal/errors"
)

// Note: TestBuildGitWorktreeArgs already exists in add_test.go
// This file focuses on testing the git command execution integration

func TestAddCommandWithExecutor_GitCommandExecution(t *testing.T) {
	tests := []struct {
		name         string
		flags        map[string]interface{}
		args         []string
		expectedArgs []string
		expectError  bool
		mockError    error
	}{
		{
			name:         "successful worktree creation",
			flags:        map[string]interface{}{},
			args:         []string{"feature/test"},
			expectedArgs: []string{"worktree", "add", "/test/worktrees/feature/test", "feature/test"},
			expectError:  false,
		},
		{
			name: "worktree creation with force",
			flags: map[string]interface{}{
				"force": true,
			},
			args:         []string{"feature/test"},
			expectedArgs: []string{"worktree", "add", "--force", "/test/worktrees/feature/test", "feature/test"},
			expectError:  false,
		},
		{
			name:         "git command failure",
			flags:        map[string]interface{}{},
			args:         []string{"feature/test"},
			expectedArgs: []string{"worktree", "add", "/test/worktrees/feature/test", "feature/test"},
			expectError:  true,
			mockError:    errors.GitCommandFailed("git worktree add", "fatal: already exists"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test setup
			cmd := createTestCommand(tt.flags, tt.args)
			var buf bytes.Buffer
			mockExec := newMockGitExecutor()

			if tt.mockError != nil {
				mockExec.SetExecuteError(tt.mockError)
			}

			cfg := &config.Config{
				Defaults: config.Defaults{
					BaseDir: "/test/worktrees",
				},
			}

			// Execute the function
			err := addCommandWithExecutor(cmd, &buf, mockExec, cfg, "/test/repo")

			// Check error expectation
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}

			// Verify git command was called with correct arguments
			commands := mockExec.GetExecutedCommands()
			assert.Len(t, commands, 1)
			assert.Equal(t, tt.expectedArgs, commands[0])
		})
	}
}

func TestHandleBranchResolutionWithExecutor(t *testing.T) {
	tests := []struct {
		name             string
		branchName       string
		flags            map[string]interface{}
		resolvedBranch   string
		isRemoteBranch   bool
		resolveBranchErr error
		expectError      bool
		expectedTrack    string
	}{
		{
			name:           "local branch - no resolution needed",
			branchName:     "feature/test",
			flags:          map[string]interface{}{},
			resolvedBranch: "feature/test",
			isRemoteBranch: false,
			expectError:    false,
		},
		{
			name:           "remote branch - sets track flag",
			branchName:     "feature/test",
			flags:          map[string]interface{}{},
			resolvedBranch: "origin/feature/test",
			isRemoteBranch: true,
			expectError:    false,
			expectedTrack:  "origin/feature/test",
		},
		{
			name:       "branch flag already set - skip resolution",
			branchName: "feature/test",
			flags: map[string]interface{}{
				"branch": "new-feature",
			},
			expectError: false,
		},
		{
			name:       "track flag already set - skip resolution",
			branchName: "feature/test",
			flags: map[string]interface{}{
				"track": "origin/develop",
			},
			expectError: false,
		},
		{
			name:       "detach flag set - skip resolution",
			branchName: "abc1234",
			flags: map[string]interface{}{
				"detach": true,
			},
			expectError: false,
		},
		{
			name:             "branch resolution error",
			branchName:       "nonexistent",
			flags:            map[string]interface{}{},
			resolveBranchErr: errors.BranchNotFound("nonexistent"),
			expectError:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create test command
			cmd := createTestCommand(tt.flags, []string{tt.branchName})

			// Create mock executor
			mockExec := newMockGitExecutor()
			mockExec.SetResolveBranch(tt.resolvedBranch, tt.isRemoteBranch, tt.resolveBranchErr)

			// Execute function
			err := handleBranchResolutionWithExecutor(cmd, mockExec, tt.branchName)

			// Check error expectation
			if tt.expectError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)

				// Check if track flag was set correctly
				if tt.expectedTrack != "" {
					assert.Equal(t, tt.expectedTrack, cmd.String("track"))
				}
			}
		})
	}
}

// Helper function to create test CLI command with flags and args
func createTestCommand(flags map[string]interface{}, args []string) *cli.Command {
	app := &cli.Command{
		Name: "test",
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "path"},
			&cli.BoolFlag{Name: "force"},
			&cli.BoolFlag{Name: "detach"},
			&cli.StringFlag{Name: "branch"},
			&cli.StringFlag{Name: "track"},
			&cli.BoolFlag{Name: "cd"},
			&cli.BoolFlag{Name: "no-cd"},
		},
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
