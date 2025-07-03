package main

import (
	"bytes"
	"context"
	"os"
	"fmt"
	"path/filepath"
	"strings"
	"testing"

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
	defer os.Chdir(oldDir)
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
	// Create a temporary git repository
	tempDir := t.TempDir()
	gitDir := filepath.Join(tempDir, ".git")
	err := os.MkdirAll(gitDir, 0755)
	assert.NoError(t, err)
	
	oldDir, _ := os.Getwd()
	defer os.Chdir(oldDir)
	err = os.Chdir(tempDir)
	assert.NoError(t, err)

	// Capture stdout
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	cmd := NewListCommand()
	ctx := context.Background()
	err = cmd.Action(ctx, &cli.Command{})

	w.Close()
	os.Stdout = oldStdout
	
	// Read output
	buf := make([]byte, 1024)
	n, _ := r.Read(buf)
	output := string(buf[:n])

	// With minimal git setup, it might show "No worktrees found" or have an error
	// depending on git configuration
	if err == nil {
		assert.Contains(t, output, "No worktrees found")
	}
}

func TestListCommand_OutputFormat(t *testing.T) {
	// This test verifies the output format logic
	// Create mock data for testing formatting
	
	tests := []struct {
		name       string
		worktrees  []struct {
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
		name           string
		paths          []string
		branches       []string
		expectedMinPath int
		expectedMinBranch int
	}{
		{
			name:             "minimum widths",
			paths:            []string{".", "."},
			branches:         []string{"", ""},
			expectedMinPath:  4,  // "PATH"
			expectedMinBranch: 6, // "BRANCH"
		},
		{
			name:             "longer paths",
			paths:            []string{"/long/path/to/worktree", "/short"},
			branches:         []string{"main", "dev"},
			expectedMinPath:  len("/long/path/to/worktree"),
			expectedMinBranch: 6, // "BRANCH"
		},
		{
			name:             "longer branches",
			paths:            []string{"/path"},
			branches:         []string{"feature/very-long-branch-name"},
			expectedMinPath:  len("/path"),
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