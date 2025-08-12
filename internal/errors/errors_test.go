package errors

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNotInGitRepository(t *testing.T) {
	err := NotInGitRepository()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in a git repository")
	assert.Contains(t, err.Error(), "git init")
	assert.Contains(t, err.Error(), "Solutions:")
}

func TestGitCommandFailed(t *testing.T) {
	tests := []struct {
		name     string
		command  string
		output   string
		expected []string
	}{
		{
			name:    "with output",
			command: "git worktree add",
			output:  "fatal: path already exists",
			expected: []string{
				"git command failed: git worktree add",
				"fatal: path already exists",
				"Try running the git command manually",
			},
		},
		{
			name:    "empty output",
			command: "git status",
			output:  "",
			expected: []string{
				"git command failed: git status",
				"no additional details available",
				"Try running the git command manually",
			},
		},
		{
			name:    "whitespace output",
			command: "git branch",
			output:  "   \n  \t  ",
			expected: []string{
				"git command failed: git branch",
				"no additional details available",
				"Try running the git command manually",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := GitCommandFailed(tt.command, tt.output)

			assert.Error(t, err)
			for _, expected := range tt.expected {
				assert.Contains(t, err.Error(), expected)
			}
		})
	}
}

func TestBranchNameRequired(t *testing.T) {
	commandExample := "wtp add <branch-name>"
	err := BranchNameRequired(commandExample)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch name is required")
	assert.Contains(t, err.Error(), commandExample)
	assert.Contains(t, err.Error(), "wtp add feature/auth")
	assert.Contains(t, err.Error(), "Examples:")
}

func TestWorktreeNameRequired(t *testing.T) {
	err := WorktreeNameRequired()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worktree name is required")
	assert.Contains(t, err.Error(), "wtp cd")
	assert.Contains(t, err.Error(), "wtp list")
}

func TestInvalidBranchName(t *testing.T) {
	branchName := "invalid..branch"
	err := InvalidBranchName(branchName)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "invalid branch name")
	assert.Contains(t, err.Error(), branchName)
	assert.Contains(t, err.Error(), "git check-ref-format")
}

func TestWorktreeNotFound(t *testing.T) {
	tests := []struct {
		name               string
		worktreeName       string
		availableWorktrees []string
		expected           []string
	}{
		{
			name:               "with available worktrees",
			worktreeName:       "feature/missing",
			availableWorktrees: []string{"main", "develop", "feature/auth"},
			expected: []string{
				"worktree 'feature/missing' not found",
				"Available worktrees:",
				"main",
				"develop",
				"feature/auth",
				"wtp list",
			},
		},
		{
			name:               "no available worktrees",
			worktreeName:       "missing",
			availableWorktrees: []string{},
			expected: []string{
				"worktree 'missing' not found",
				"No worktrees found",
				"wtp list",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WorktreeNotFound(tt.worktreeName, tt.availableWorktrees)

			assert.Error(t, err)
			for _, expected := range tt.expected {
				assert.Contains(t, err.Error(), expected)
			}
		})
	}
}

func TestWorktreeCreationFailed(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		branch   string
		gitError error
		expected []string
	}{
		{
			name:     "already checked out",
			path:     "/path/to/worktree",
			branch:   "feature/auth",
			gitError: fmt.Errorf("fatal: 'feature/auth' is already checked out"),
			expected: []string{
				"failed to create worktree at '/path/to/worktree' for branch 'feature/auth'",
				"already checked out",
				"--force",
				"Original error:",
			},
		},
		{
			name:     "destination exists",
			path:     "/path/exists",
			branch:   "main",
			gitError: fmt.Errorf("fatal: destination path '/path/exists' already exists"),
			expected: []string{
				"failed to create worktree at '/path/exists' for branch 'main'",
				"destination path",
				"already exists",
				"Remove the existing directory",
				"Original error:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := WorktreeCreationFailed(tt.path, tt.branch, tt.gitError)

			assert.Error(t, err)
			for _, expected := range tt.expected {
				assert.Contains(t, err.Error(), expected)
			}
		})
	}
}

func TestWorktreeRemovalFailed(t *testing.T) {
	path := "/path/to/worktree"
	gitError := fmt.Errorf("fatal: working tree is dirty")
	err := WorktreeRemovalFailed(path, gitError)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove worktree")
	assert.Contains(t, err.Error(), path)
	assert.Contains(t, err.Error(), "working tree is dirty")
	assert.Contains(t, err.Error(), "Original error:")
}

func TestBranchRemovalFailed(t *testing.T) {
	tests := []struct {
		name       string
		branchName string
		gitError   error
		forced     bool
		expected   []string
	}{
		{
			name:       "not merged",
			branchName: "feature/auth",
			gitError:   fmt.Errorf("error: branch 'feature/auth' is not fully merged"),
			forced:     false,
			expected: []string{
				"failed to remove branch 'feature/auth'",
				"not fully merged",
				"--force-branch",
				"Original error:",
			},
		},
		{
			name:       "forced deletion failed",
			branchName: "main",
			gitError:   fmt.Errorf("error: cannot delete currently checked out branch 'main'"),
			forced:     true,
			expected: []string{
				"failed to remove branch 'main'",
				"checked out",
				"Original error:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := BranchRemovalFailed(tt.branchName, tt.gitError, tt.forced)

			assert.Error(t, err)
			for _, expected := range tt.expected {
				assert.Contains(t, err.Error(), expected)
			}
		})
	}
}

func TestConfigLoadFailed(t *testing.T) {
	tests := []struct {
		name     string
		path     string
		reason   error
		expected []string
	}{
		{
			name:   "file not found",
			path:   ".wtp.yml",
			reason: fmt.Errorf("no such file or directory"),
			expected: []string{
				"failed to load configuration from '.wtp.yml'",
				"no such file or directory",
				"wtp init",
			},
		},
		{
			name:   "invalid yaml",
			path:   "/custom/config.yml",
			reason: fmt.Errorf("yaml: unmarshal error"),
			expected: []string{
				"failed to load configuration from '/custom/config.yml'",
				"yaml",
				"yamllint.com",
				"Original error:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := ConfigLoadFailed(tt.path, tt.reason)

			assert.Error(t, err)
			for _, expected := range tt.expected {
				assert.Contains(t, err.Error(), expected)
			}
		})
	}
}

func TestConfigAlreadyExists(t *testing.T) {
	path := ".wtp.yml"
	err := ConfigAlreadyExists(path)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "configuration file already exists")
	assert.Contains(t, err.Error(), path)
	assert.Contains(t, err.Error(), "Options:")
}

func TestDirectoryAccessFailed(t *testing.T) {
	tests := []struct {
		name      string
		operation string
		path      string
		reason    error
		expected  []string
	}{
		{
			name:      "permission denied",
			operation: "access",
			path:      ".",
			reason:    fmt.Errorf("permission denied"),
			expected: []string{
				"failed to access directory: .",
				"permission denied",
				"Check directory permissions",
				"Original error:",
			},
		},
		{
			name:      "directory does not exist",
			operation: "create",
			path:      "/path/to/dir",
			reason:    fmt.Errorf("no such file or directory"),
			expected: []string{
				"failed to create directory: /path/to/dir",
				"no such file or directory",
				"Create the parent directory",
				"Original error:",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := DirectoryAccessFailed(tt.operation, tt.path, tt.reason)

			assert.Error(t, err)
			for _, expected := range tt.expected {
				assert.Contains(t, err.Error(), expected)
			}
		})
	}
}

func TestShellIntegrationRequired(t *testing.T) {
	err := ShellIntegrationRequired()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "shell integration")
	assert.Contains(t, err.Error(), "eval")
	assert.Contains(t, err.Error(), "wtp shell")
}

func TestUnsupportedShell(t *testing.T) {
	shell := "tcsh"
	supportedShells := []string{"bash", "zsh", "fish"}
	err := UnsupportedShell(shell, supportedShells)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unsupported shell")
	assert.Contains(t, err.Error(), shell)
	assert.Contains(t, err.Error(), "bash")
	assert.Contains(t, err.Error(), "zsh")
	assert.Contains(t, err.Error(), "fish")
}

func TestBranchNotFound(t *testing.T) {
	branchName := "feature/missing"
	err := BranchNotFound(branchName)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch 'feature/missing' not found")
	assert.Contains(t, err.Error(), "local or remote branches")
	assert.Contains(t, err.Error(), "git branch -a")
}

func TestMultipleBranchesFound(t *testing.T) {
	branchName := "feature"
	remotes := []string{"origin", "upstream"}
	err := MultipleBranchesFound(branchName, remotes)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch 'feature' exists in multiple remotes")
	assert.Contains(t, err.Error(), "origin")
	assert.Contains(t, err.Error(), "upstream")
	assert.Contains(t, err.Error(), "Specify the remote explicitly")
}

func TestHookExecutionFailed(t *testing.T) {
	hookIndex := 0
	hookType := "copy"
	originalError := fmt.Errorf("permission denied")
	err := HookExecutionFailed(hookIndex, hookType, originalError)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to execute copy hook")
	assert.Contains(t, err.Error(), hookType)
	assert.Contains(t, err.Error(), "permission denied")
	assert.Contains(t, err.Error(), "Original error:")
}

func TestErrorMessages_HelpfulContent(t *testing.T) {
	// Test that all error messages contain helpful suggestions
	tests := []struct {
		name     string
		errorFn  func() error
		keywords []string
	}{
		{
			name:     "NotInGitRepository contains solutions",
			errorFn:  NotInGitRepository,
			keywords: []string{"Solutions:", "git init", "Navigate"},
		},
		{
			name:     "WorktreeNameRequired contains examples",
			errorFn:  WorktreeNameRequired,
			keywords: []string{"wtp cd", "wtp list"},
		},
		{
			name:     "ShellIntegrationRequired contains setup",
			errorFn:  ShellIntegrationRequired,
			keywords: []string{"eval", "wtp shell"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.errorFn()
			assert.Error(t, err)

			// Check that error messages are helpful and contain guidance
			errMsg := err.Error()
			assert.True(t, len(errMsg) > 50, "Error message should be detailed")

			for _, keyword := range tt.keywords {
				assert.Contains(t, errMsg, keyword)
			}
		})
	}
}

func TestErrorMessages_Format(t *testing.T) {
	// Test that error messages are well-formatted
	tests := []func() error{
		NotInGitRepository,
		func() error { return BranchNameRequired("example") },
		WorktreeNameRequired,
		ShellIntegrationRequired,
	}

	for i, errorFn := range tests {
		t.Run(fmt.Sprintf("error_%d_format", i), func(t *testing.T) {
			err := errorFn()
			assert.Error(t, err)

			errMsg := err.Error()

			// Error messages should not be empty
			assert.NotEmpty(t, errMsg)

			// Error messages should not start or end with whitespace
			assert.Equal(t, strings.TrimSpace(errMsg), errMsg)

			// Error messages should contain newlines for readability
			assert.Contains(t, errMsg, "\n")
		})
	}
}
