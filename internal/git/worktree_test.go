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
			name: "root worktree should show repo name with branch and root indicator",
			worktree: Worktree{
				Path:   "/Users/user/repos/giselle",
				Branch: "fix-nodes",
				IsMain: true,
			},
			repoName:         "giselle",
			mainWorktreePath: "/Users/user/repos/giselle",
			expected:         "giselle@fix-nodes(root worktree)",
		},
		{
			name: "root worktree with main branch should show repo name with branch and root indicator",
			worktree: Worktree{
				Path:   "/Users/user/repos/wtp",
				Branch: "main",
				IsMain: true,
			},
			repoName:         "wtp",
			mainWorktreePath: "/Users/user/repos/wtp",
			expected:         "wtp@main(root worktree)",
		},
		{
			name: "feature branch worktree should show branch when worktree name differs",
			worktree: Worktree{
				Path:   "/Users/user/repos/wtp/worktrees/feature-awesome",
				Branch: "feature/awesome",
			},
			repoName:         "wtp",
			mainWorktreePath: "/Users/user/repos/wtp",
			expected:         "feature-awesome@feature/awesome",
		},
		{
			name: "fix branch with multiple slashes should show worktree name and branch",
			worktree: Worktree{
				Path:   "/Users/user/repos/wtp/worktrees/fix-123-fix-login",
				Branch: "fix/123/fix-login",
			},
			repoName:         "wtp",
			mainWorktreePath: "/Users/user/repos/wtp",
			expected:         "fix-123-fix-login@fix/123/fix-login",
		},
		{
			name: "simple branch where worktree name matches branch should show branch only",
			worktree: Worktree{
				Path:   "/Users/user/repos/wtp/worktrees/develop",
				Branch: "develop",
			},
			repoName:         "wtp",
			mainWorktreePath: "/Users/user/repos/wtp",
			expected:         "develop",
		},
		{
			name: "feature branch where worktree name matches branch should show branch only",
			worktree: Worktree{
				Path:   "/Users/user/repos/wtp/worktrees/feature/new-top-page",
				Branch: "feature/new-top-page",
			},
			repoName:         "wtp",
			mainWorktreePath: "/Users/user/repos/wtp",
			expected:         "feature/new-top-page",
		},
		{
			name: "worktree in worktrees directory should not be detected as root",
			worktree: Worktree{
				Path:   "/Users/user/repos/giselle/.worktrees/stripe-basil-update",
				Branch: "stripe-basil-migration",
			},
			repoName:         "giselle",
			mainWorktreePath: "/Users/user/repos/giselle",
			expected:         "stripe-basil-update@stripe-basil-migration",
		},
		{
			name: "worktree in .worktrees directory should not be detected as root",
			worktree: Worktree{
				Path:   "/Users/satoshi/dev/src/github.com/giselles-ai/giselle/.worktrees/stripe-basil-update",
				Branch: "stripe-basil-migration",
			},
			repoName:         "giselle",
			mainWorktreePath: "/Users/satoshi/dev/src/github.com/giselles-ai/giselle",
			expected:         "stripe-basil-update@stripe-basil-migration",
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
