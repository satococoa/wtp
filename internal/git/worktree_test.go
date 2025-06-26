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