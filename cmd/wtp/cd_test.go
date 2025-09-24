package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

// ===== Critical User Scenarios =====

// This is the most important test - the core value proposition
func TestCdCommand_AlwaysOutputsAbsolutePath(t *testing.T) {
	// Setup a realistic worktree scenario
	worktreeList := `worktree /Users/dev/project/main
HEAD abc123
branch refs/heads/main

worktree /Users/dev/project/worktrees/feature/auth
HEAD def456
branch refs/heads/feature/auth

`

	tests := []struct {
		name          string
		worktreeName  string
		expectedPath  string
		shouldSucceed bool
	}{
		{
			name:          "main worktree by @ symbol",
			worktreeName:  "@",
			expectedPath:  "/Users/dev/project/main",
			shouldSucceed: true,
		},
		{
			name:          "feature worktree by branch name",
			worktreeName:  "feature/auth",
			expectedPath:  "/Users/dev/project/worktrees/feature/auth",
			shouldSucceed: true,
		},
		{
			name:          "feature worktree by directory name",
			worktreeName:  "auth",
			expectedPath:  "/Users/dev/project/worktrees/feature/auth",
			shouldSucceed: true, // Directory-based resolution works as expected
		},
		{
			name:          "nonexistent worktree",
			worktreeName:  "nonexistent",
			shouldSucceed: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worktrees := parseWorktreesFromOutput(worktreeList)
			mainPath := findMainWorktreePath(worktrees)

			resolvedPath := resolveCdWorktreePath(tt.worktreeName, worktrees, mainPath)

			if tt.shouldSucceed {
				assert.Equal(t, tt.expectedPath, resolvedPath,
					"cd command must output correct absolute path")
				assert.True(t, filepath.IsAbs(resolvedPath),
					"cd command must always output absolute paths")
			} else {
				assert.Empty(t, resolvedPath,
					"cd command should return empty string for nonexistent worktrees")
			}
		})
	}
}

// Test critical error scenarios that users will encounter
func TestCdCommand_UserFacingErrors(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "no arguments",
			args:          []string{},
			expectedError: "worktree name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
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

// Test edge cases that could break in production
func TestCdCommand_EdgeCases(t *testing.T) {
	tests := []struct {
		name         string
		worktreeName string
		worktreeList string
		expected     string
		shouldFind   bool
	}{
		{
			name:         "worktree name with special characters",
			worktreeName: "feature/fix-auth-123",
			worktreeList: "worktree /path/feature-fix-auth-123\nHEAD abc\nbranch refs/heads/feature/fix-auth-123\n\n",
			expected:     "/path/feature-fix-auth-123",
			shouldFind:   true,
		},
		{
			name:         "completion marker removal (asterisk)",
			worktreeName: "feature*",
			worktreeList: "worktree /path/feature\nHEAD abc\nbranch refs/heads/feature\n\n",
			expected:     "/path/feature",
			shouldFind:   true,
		},
		{
			name:         "empty worktree list",
			worktreeName: "any",
			worktreeList: "",
			expected:     "",
			shouldFind:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worktrees := parseWorktreesFromOutput(tt.worktreeList)
			mainPath := findMainWorktreePath(worktrees)

			result := resolveCdWorktreePath(tt.worktreeName, worktrees, mainPath)

			if tt.shouldFind {
				assert.Equal(t, tt.expected, result)
			} else {
				assert.Empty(t, result)
			}
		})
	}
}

// Only test command structure that affects user behavior
func TestCdCommand_CoreBehavior(t *testing.T) {
	cmd := NewCdCommand()
	assert.Equal(t, "cd", cmd.Name)
	assert.Equal(t, "Output absolute path to worktree", cmd.Usage)
	assert.NotNil(t, cmd.ShellComplete)
	// The rest is implementation detail - what matters is that it works
}

// ===== Worktree Completion Tests =====

func TestGetWorktreeNameFromPathCd(t *testing.T) {
	RunNameFromPathTests(t, "cd", getWorktreeNameFromPathCd)
}

func TestGetWorktreesForCd(t *testing.T) {
	RunWriterCommonTests(t, "getWorktreesForCd", getWorktreesForCd)
}

func TestCompleteWorktreesForCd(t *testing.T) {
	t.Run("should not panic when called", func(t *testing.T) {
		cmd := &cli.Command{}

		// Should not panic even without proper git setup
		assert.NotPanics(t, func() {
			// Capture stdout to avoid noise in tests
			oldStdout := os.Stdout
			os.Stdout = os.NewFile(0, os.DevNull)
			defer func() { os.Stdout = oldStdout }()

			completeWorktreesForCd(context.Background(), cmd)
		})
	})
}
