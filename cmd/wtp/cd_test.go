package main

import (
	"context"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

// ===== Critical User Scenarios =====

// This is the most important test - the core value proposition
func TestCdCommand_AlwaysOutputsAbsolutePath(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("TODO: Fix for Windows - test uses Unix-specific paths")
	}

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

// Test the architectural guarantee: no environment variable dependency
func TestCdCommand_NoEnvironmentVariableDependency(t *testing.T) {
	// This test ensures we maintain the "pure function" architecture

	// Make sure no environment variables affect the core function
	originalEnv := os.Getenv("WTP_SHELL_INTEGRATION")
	t.Cleanup(func() {
		if originalEnv != "" {
			require.NoError(t, os.Setenv("WTP_SHELL_INTEGRATION", originalEnv))
		} else {
			require.NoError(t, os.Unsetenv("WTP_SHELL_INTEGRATION"))
		}
	})

	// Test with various environment states
	envStates := []struct {
		name  string
		value string
	}{
		{"no env var", ""},
		{"env var set to 1", "1"},
		{"env var set to 0", "0"},
		{"env var set to random", "random"},
	}

	for _, env := range envStates {
		t.Run(env.name, func(t *testing.T) {
			if env.value == "" {
				require.NoError(t, os.Unsetenv("WTP_SHELL_INTEGRATION"))
			} else {
				require.NoError(t, os.Setenv("WTP_SHELL_INTEGRATION", env.value))
			}

			// The core resolution function should work regardless of environment
			worktreeList := "worktree /test/main\nHEAD abc\nbranch refs/heads/main\n\n"
			worktrees := parseWorktreesFromOutput(worktreeList)
			mainPath := findMainWorktreePath(worktrees)

			resolvedPath := resolveCdWorktreePath("@", worktrees, mainPath)
			assert.Equal(t, "/test/main", resolvedPath,
				"Path resolution must not depend on environment variables")
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
			restore := silenceStdout(t)
			defer restore()

			completeWorktreesForCd(context.Background(), cmd)
		})
	})
}
