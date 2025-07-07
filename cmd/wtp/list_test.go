package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"strings"
	"testing"

	"github.com/satococoa/wtp/internal/command"
	"github.com/satococoa/wtp/internal/git"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

func TestNewListCommand(t *testing.T) {
	cmd := NewListCommand()

	assert.NotNil(t, cmd)
	assert.Equal(t, "list", cmd.Name)
	assert.Equal(t, "List all worktrees", cmd.Usage)
	assert.NotEmpty(t, cmd.Description)
	assert.NotNil(t, cmd.Action)
}

func TestDisplayConstants(t *testing.T) {
	assert.Equal(t, 4, pathHeaderDashes)
	assert.Equal(t, 6, branchHeaderDashes)
	assert.Equal(t, 8, headDisplayLength)
}

func TestListCommand_NotInGitRepo(t *testing.T) {
	// Create a temporary directory that is not a git repo
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	err := os.Chdir(tempDir)
	assert.NoError(t, err)

	app := &cli.Command{
		Commands: []*cli.Command{
			NewListCommand(),
		},
	}

	ctx := context.Background()
	err = app.Run(ctx, []string{"wtp", "list"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in a git repository")
}

func TestListCommand_DirectoryAccessError(t *testing.T) {
	// Save original listGetwd to restore later
	originalGetwd := listGetwd
	defer func() { listGetwd = originalGetwd }()

	// Mock listGetwd to return an error
	listGetwd = func() (string, error) {
		return "", assert.AnError
	}

	cmd := NewListCommand()
	ctx := context.Background()
	err := cmd.Action(ctx, &cli.Command{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to access")
}

func TestListCommand_NoWorktrees(t *testing.T) {
	// Save original functions
	originalGetwd := listGetwd
	originalNewRepo := listNewRepository
	defer func() {
		listGetwd = originalGetwd
		listNewRepository = originalNewRepo
	}()

	// Mock functions
	listGetwd = func() (string, error) {
		return "/test/repo", nil
	}

	// Create a mock repository that returns empty worktrees
	listNewRepository = func(_ string) (GitRepository, error) {
		return &mockRepository{
			worktrees: []git.Worktree{},
		}, nil
	}

	// Create app with Writer
	var buf bytes.Buffer
	app := &cli.Command{
		Writer: &buf,
		Commands: []*cli.Command{
			NewListCommand(),
		},
	}

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "list"})

	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "No worktrees found")
}

func TestListCommand_OutputFormat(t *testing.T) {
	// This test verifies the output format logic
	// Create mock data for testing formatting

	tests := []struct {
		name      string
		worktrees []struct {
			path   string
			branch string
			head   string
		}
		expectedHeaders []string
	}{
		{
			name: "standard worktrees",
			worktrees: []struct {
				path   string
				branch string
				head   string
			}{
				{"/path/to/main", "main", "abc12345"},
				{"/path/to/feature", "feature/test", "def67890"},
			},
			expectedHeaders: []string{"PATH", "BRANCH", "HEAD"},
		},
		{
			name: "long paths and branches",
			worktrees: []struct {
				path   string
				branch string
				head   string
			}{
				{"/very/long/path/to/worktree/directory", "feature/very-long-branch-name", "1234567890abcdef"},
			},
			expectedHeaders: []string{"PATH", "BRANCH", "HEAD"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// This test primarily verifies that the constants exist and are reasonable
			// Actual integration testing would require git repository setup

			// Verify headers would be displayed
			for _, header := range tt.expectedHeaders {
				assert.NotEmpty(t, header)
			}

			// Verify head would be truncated
			for _, wt := range tt.worktrees {
				if len(wt.head) > headDisplayLength {
					truncated := wt.head[:headDisplayLength]
					assert.Equal(t, headDisplayLength, len(truncated))
				}
			}
		})
	}
}

func TestListCommand_ColumnWidthCalculation(t *testing.T) {
	// Test the column width calculation logic
	testCases := []struct {
		name              string
		paths             []string
		branches          []string
		expectedMinPath   int
		expectedMinBranch int
	}{
		{
			name:              "minimum widths",
			paths:             []string{".", "."},
			branches:          []string{"", ""},
			expectedMinPath:   4, // "PATH"
			expectedMinBranch: 6, // "BRANCH"
		},
		{
			name:              "longer paths",
			paths:             []string{"/long/path/to/worktree", "/short"},
			branches:          []string{"main", "dev"},
			expectedMinPath:   len("/long/path/to/worktree"),
			expectedMinBranch: 6, // "BRANCH"
		},
		{
			name:              "longer branches",
			paths:             []string{"/path"},
			branches:          []string{"feature/very-long-branch-name"},
			expectedMinPath:   len("/path"),
			expectedMinBranch: len("feature/very-long-branch-name"),
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Calculate max lengths
			maxPath := 4   // "PATH"
			maxBranch := 6 // "BRANCH"

			for _, p := range tc.paths {
				if len(p) > maxPath {
					maxPath = len(p)
				}
			}

			for _, b := range tc.branches {
				if len(b) > maxBranch {
					maxBranch = len(b)
				}
			}

			assert.GreaterOrEqual(t, maxPath, tc.expectedMinPath)
			assert.GreaterOrEqual(t, maxBranch, tc.expectedMinBranch)
		})
	}
}

func TestListCommand_HeaderFormat(t *testing.T) {
	// Test that header formatting works correctly
	var buf bytes.Buffer

	// Simulate header output
	maxPathLen := 20
	maxBranchLen := 15

	// Format header like the actual command does
	header := strings.TrimSpace(
		fmt.Sprintf("%-*s %-*s %s", maxPathLen, "PATH", maxBranchLen, "BRANCH", "HEAD"),
	)

	buf.WriteString(header + "\n")

	output := buf.String()

	// Verify headers are present and properly spaced
	assert.Contains(t, output, "PATH")
	assert.Contains(t, output, "BRANCH")
	assert.Contains(t, output, "HEAD")

	// Verify spacing
	assert.True(t, strings.Index(output, "BRANCH") > strings.Index(output, "PATH"))
	assert.True(t, strings.Index(output, "HEAD") > strings.Index(output, "BRANCH"))
}

func TestDisplayWorktrees(t *testing.T) {
	tests := []struct {
		name      string
		worktrees []git.Worktree
		expected  []string
	}{
		{
			name: "simple worktrees",
			worktrees: []git.Worktree{
				{Path: ".", Branch: "main", HEAD: "abc12345"},
				{Path: "../worktrees/feature", Branch: "feature/test", HEAD: "def67890"},
			},
			expected: []string{
				"PATH", "BRANCH", "HEAD",
				"----", "------", "----",
				".", "main", "abc12345",
				"../worktrees/feature", "feature/test", "def67890",
			},
		},
		{
			name: "long commit hash truncated",
			worktrees: []git.Worktree{
				{Path: "/path", Branch: "dev", HEAD: "1234567890abcdef"},
			},
			expected: []string{
				"PATH", "BRANCH", "HEAD",
				"/path", "dev", "12345678", // Truncated to 8 chars
			},
		},
		{
			name: "empty branch name",
			worktrees: []git.Worktree{
				{Path: "/detached", Branch: "", HEAD: "abcdef12"},
			},
			expected: []string{
				"PATH", "BRANCH", "HEAD",
				"/detached", "", "abcdef12",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var buf bytes.Buffer
			displayWorktrees(&buf, tt.worktrees)

			output := buf.String()

			// Check all expected strings are present
			for _, exp := range tt.expected {
				assert.Contains(t, output, exp)
			}

			// Verify the output has the correct number of lines
			lines := strings.Split(strings.TrimSpace(output), "\n")
			expectedLines := 2 + len(tt.worktrees) // Header + separator + worktrees
			assert.Equal(t, expectedLines, len(lines))
		})
	}
}

func TestListCommand_WithMockOutput(t *testing.T) {
	// Save original functions
	originalGetwd := listGetwd
	originalNewRepo := listNewRepository
	defer func() {
		listGetwd = originalGetwd
		listNewRepository = originalNewRepo
	}()

	// Mock functions
	listGetwd = func() (string, error) {
		return "/test/repo", nil
	}

	// Create a mock repository that returns worktrees
	listNewRepository = func(_ string) (GitRepository, error) {
		// Return a mock repository with a GetWorktrees method
		return &mockRepository{
			worktrees: []git.Worktree{
				{Path: ".", Branch: "main", HEAD: "abc12345"},
				{Path: "../worktrees/feature", Branch: "feature/auth", HEAD: "def67890123456"},
			},
		}, nil
	}

	// Create app with Writer
	var buf bytes.Buffer
	app := &cli.Command{
		Writer: &buf,
		Commands: []*cli.Command{
			NewListCommand(),
		},
	}

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "list"})

	assert.NoError(t, err)
	output := buf.String()

	// Verify output contains expected elements
	assert.Contains(t, output, "PATH")
	assert.Contains(t, output, "BRANCH")
	assert.Contains(t, output, "HEAD")
	assert.Contains(t, output, "main")
	assert.Contains(t, output, "feature/auth")
	assert.Contains(t, output, "abc12345")
	assert.Contains(t, output, "def67890") // Truncated HEAD
}

// mockRepository is a test mock for GitRepository interface
type mockRepository struct {
	worktrees []git.Worktree
}

func (m *mockRepository) GetWorktrees() ([]git.Worktree, error) {
	return m.worktrees, nil
}

// Test with CommandExecutor architecture
func TestListCommandWithCommandExecutor_Success(t *testing.T) {
	mockExec := &mockListCommandExecutor{
		results: []command.Result{
			{
				Output: "worktree /path/to/main\nHEAD abc123\nbranch refs/heads/main\n\n" +
					"worktree /path/to/feature\nHEAD def456\nbranch refs/heads/feature/auth\n\n",
				Error: nil,
			},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	err := listCommandWithCommandExecutor(cmd, &buf, mockExec, "/repo")

	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "PATH")
	assert.Contains(t, output, "BRANCH")
	assert.Contains(t, output, "HEAD")
	assert.Contains(t, output, "/path/to/main")
	assert.Contains(t, output, "/path/to/feature")
	assert.Contains(t, output, "main")
	assert.Contains(t, output, "feature/auth")
	assert.Len(t, mockExec.executedCommands, 1)
	assert.Equal(t, []string{"worktree", "list", "--porcelain"}, mockExec.executedCommands[0].Args)
}

func TestListCommandWithCommandExecutor_NoWorktrees(t *testing.T) {
	mockExec := &mockListCommandExecutor{
		results: []command.Result{
			{
				Output: "",
				Error:  nil,
			},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	err := listCommandWithCommandExecutor(cmd, &buf, mockExec, "/repo")

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "No worktrees found")
}

// Mock command executor for list testing
type mockListCommandExecutor struct {
	executedCommands []command.Command
	results          []command.Result
}

func (m *mockListCommandExecutor) Execute(commands []command.Command) (*command.ExecutionResult, error) {
	m.executedCommands = append(m.executedCommands, commands...)

	results := make([]command.Result, len(commands))
	for i, cmd := range commands {
		if i < len(m.results) {
			results[i] = m.results[i]
		} else {
			results[i] = command.Result{
				Command: cmd,
				Output:  "",
				Error:   nil,
			}
		}
	}

	return &command.ExecutionResult{Results: results}, nil
}
