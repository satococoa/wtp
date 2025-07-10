package git

import "testing"

func TestWorktreeName(t *testing.T) {
	tests := []struct {
		name     string
		worktree Worktree
		expected string
	}{
		{
			name: "simple path",
			worktree: Worktree{
				Path: "/home/user/worktrees/main",
			},
			expected: "main",
		},
		{
			name: "nested path",
			worktree: Worktree{
				Path: "/home/user/worktrees/feature/auth",
			},
			expected: "auth",
		},
		{
			name: "root path",
			worktree: Worktree{
				Path: "/",
			},
			expected: "/",
		},
		{
			name: "trailing slash",
			worktree: Worktree{
				Path: "/home/user/worktrees/main/",
			},
			expected: "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.worktree.Name()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestWorktreeString(t *testing.T) {
	tests := []struct {
		name     string
		worktree Worktree
		expected string
	}{
		{
			name: "worktree with branch",
			worktree: Worktree{
				Path:   "/home/user/worktrees/main",
				Branch: "main",
				HEAD:   "abcd1234567890",
			},
			expected: "/home/user/worktrees/main [main]",
		},
		{
			name: "worktree without branch",
			worktree: Worktree{
				Path: "/home/user/worktrees/detached",
				HEAD: "abcd1234567890",
			},
			expected: "/home/user/worktrees/detached [abcd1234567890]",
		},
		{
			name: "worktree without HEAD",
			worktree: Worktree{
				Path:   "/home/user/worktrees/main",
				Branch: "main",
			},
			expected: "/home/user/worktrees/main [main]",
		},
		{
			name: "minimal worktree",
			worktree: Worktree{
				Path: "/home/user/worktrees/empty",
			},
			expected: "/home/user/worktrees/empty []",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.worktree.String()
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestWorktree_CompletionName(t *testing.T) {
	tests := []struct {
		name             string
		worktree         Worktree
		repoName         string
		mainWorktreePath string
		expected         string
	}{
		{
			name: "root worktree should show repo name with root indicator",
			worktree: Worktree{
				Path:   "/Users/user/repos/wtp",
				Branch: "main",
			},
			repoName:         "wtp",
			mainWorktreePath: "/Users/user/repos/wtp",
			expected:         "wtp(root worktree)",
		},
		{
			name: "feature branch should preserve prefix",
			worktree: Worktree{
				Path:   "/Users/user/repos/wtp/worktrees/feature-awesome",
				Branch: "feature/awesome",
			},
			repoName:         "wtp",
			mainWorktreePath: "/Users/user/repos/wtp",
			expected:         "feature/awesome",
		},
		{
			name: "fix branch with multiple slashes should preserve full path",
			worktree: Worktree{
				Path:   "/Users/user/repos/wtp/worktrees/fix-123-fix-login",
				Branch: "fix/123/fix-login",
			},
			repoName:         "wtp",
			mainWorktreePath: "/Users/user/repos/wtp",
			expected:         "fix/123/fix-login",
		},
		{
			name: "simple branch name should be preserved",
			worktree: Worktree{
				Path:   "/Users/user/repos/wtp/worktrees/develop",
				Branch: "develop",
			},
			repoName:         "wtp",
			mainWorktreePath: "/Users/user/repos/wtp",
			expected:         "develop",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.worktree.CompletionName(tt.repoName)
			if result != tt.expected {
				t.Errorf("Expected %q, got %q", tt.expected, result)
			}
		})
	}
}

func TestWorktree_IsMainWorktree(t *testing.T) {
	tests := []struct {
		name             string
		worktree         Worktree
		mainWorktreePath string
		expected         bool
	}{
		{
			name: "main worktree should return true",
			worktree: Worktree{
				Path: "/Users/user/repos/wtp",
			},
			mainWorktreePath: "/Users/user/repos/wtp",
			expected:         true,
		},
		{
			name: "different worktree should return false",
			worktree: Worktree{
				Path: "/Users/user/repos/wtp/worktrees/feature-branch",
			},
			mainWorktreePath: "/Users/user/repos/wtp",
			expected:         false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.worktree.IsMainWorktree(tt.mainWorktreePath)
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
