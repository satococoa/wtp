package framework

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func AssertWorktreeCreated(t *testing.T, output, branch string) {
	t.Helper()
	// Check for new friendly success message format
	if !strings.Contains(output, "✅ Worktree created successfully!") &&
		!strings.Contains(output, "Created worktree") &&
		!strings.Contains(output, "Preparing worktree") {
		t.Errorf("Expected worktree creation message, got: %s", output)
	}
	if !strings.Contains(output, branch) {
		t.Errorf("Expected branch name '%s' in output, got: %s", branch, output)
	}
}

func AssertErrorContains(t *testing.T, err error, expected string) {
	t.Helper()
	assert.Error(t, err, "Expected error containing '%s', but got no error", expected)
	assert.Contains(t, err.Error(), expected, "Expected error containing '%s', got: %v", expected, err)
}

func AssertOutputContains(t *testing.T, output, expected string) {
	t.Helper()
	assert.Contains(t, output, expected, "Expected output containing '%s', got: %s", expected, output)
}

func AssertHelpfulError(t *testing.T, output string) {
	t.Helper()

	helpfulElements := []string{
		"Suggestions:",
		"Solutions:",
		"Solution:",
		"Cause:",
		"Tip:",
		"•",
		"Examples:",
		"Usage:",
	}

	found := false
	for _, element := range helpfulElements {
		if strings.Contains(output, element) {
			found = true
			break
		}
	}

	if !found {
		t.Errorf("Error message does not appear to be helpful. Got: %s", output)
	}
}

func AssertMultipleStringsInOutput(t *testing.T, output string, expected []string) {
	t.Helper()
	for _, exp := range expected {
		assert.Contains(t, output, exp, "Expected output to contain '%s', got: %s", exp, output)
	}
}

func AssertNoError(t *testing.T, err error) {
	t.Helper()
	assert.NoError(t, err)
}

func AssertError(t *testing.T, err error) {
	t.Helper()
	assert.Error(t, err)
}

func AssertFileExists(t *testing.T, repo *TestRepo, path string) {
	t.Helper()
	assert.True(t, repo.HasFile(path), "Expected file '%s' to exist", path)
}

func AssertFileNotExists(t *testing.T, repo *TestRepo, path string) {
	t.Helper()
	assert.False(t, repo.HasFile(path), "Expected file '%s' not to exist", path)
}

func AssertFileContains(t *testing.T, repo *TestRepo, path, content string) {
	t.Helper()
	assert.True(t, repo.HasFile(path), "File '%s' does not exist", path)
	if repo.HasFile(path) {
		fileContent := repo.ReadFile(path)
		assert.Contains(t, fileContent, content, "Expected file '%s' to contain '%s', got: %s", path, content, fileContent)
	}
}

func AssertCurrentBranch(t *testing.T, repo *TestRepo, expected string) {
	t.Helper()
	current := repo.CurrentBranch()
	assert.Equal(t, expected, current, "Expected current branch to be '%s', got: '%s'", expected, current)
}

func AssertWorktreeCount(t *testing.T, repo *TestRepo, expected int) {
	t.Helper()
	worktrees := repo.ListWorktrees()
	assert.Len(t, worktrees, expected, "Expected %d worktrees, got %d: %v", expected, len(worktrees), worktrees)
}

func AssertWorktreeExists(t *testing.T, repo *TestRepo, path string) {
	t.Helper()
	worktrees := repo.ListWorktrees()
	found := false
	for _, wt := range worktrees {
		if strings.Contains(wt, path) {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("Expected worktree at path '%s' to exist, got worktrees: %v", path, worktrees)
	}
}

func AssertWorktreeNotExists(t *testing.T, repo *TestRepo, path string) {
	t.Helper()
	worktrees := repo.ListWorktrees()
	for _, wt := range worktrees {
		if strings.Contains(wt, path) {
			t.Errorf("Expected worktree at path '%s' not to exist, but it does", path)
		}
	}
}

func AssertEqual(t *testing.T, expected, actual any) {
	t.Helper()
	assert.Equal(t, expected, actual)
}

func AssertNotEqual(t *testing.T, notExpected, actual any) {
	t.Helper()
	assert.NotEqual(t, notExpected, actual)
}

func AssertTrue(t *testing.T, condition bool, message string) {
	t.Helper()
	assert.True(t, condition, message)
}

func AssertFalse(t *testing.T, condition bool, message string) {
	t.Helper()
	assert.False(t, condition, message)
}
