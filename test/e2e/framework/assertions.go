// Package framework provides helpers for end-to-end testing.
package framework

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// AssertWorktreeCreated verifies worktree creation output contains the expected branch name.
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

// AssertErrorContains ensures an error is present and includes the expected text.
func AssertErrorContains(t *testing.T, err error, expected string) {
	t.Helper()
	assert.Error(t, err, "Expected error containing '%s', but got no error", expected)
	assert.Contains(t, err.Error(), expected, "Expected error containing '%s', got: %v", expected, err)
}

// AssertOutputContains checks that output includes the expected substring.
func AssertOutputContains(t *testing.T, output, expected string) {
	t.Helper()
	assert.Contains(t, output, expected, "Expected output containing '%s', got: %s", expected, output)
}

// AssertHelpfulError validates that an error message contains helpful guidance markers.
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

// AssertMultipleStringsInOutput verifies that all expected substrings appear in the output.
func AssertMultipleStringsInOutput(t *testing.T, output string, expected []string) {
	t.Helper()
	for _, exp := range expected {
		assert.Contains(t, output, exp, "Expected output to contain '%s', got: %s", exp, output)
	}
}

// AssertNoError fails the test if an unexpected error is present.
func AssertNoError(t *testing.T, err error) {
	t.Helper()
	assert.NoError(t, err)
}

// AssertError fails the test if an error is not present.
func AssertError(t *testing.T, err error) {
	t.Helper()
	assert.Error(t, err)
}

// AssertFileExists checks that a file exists in the repository.
func AssertFileExists(t *testing.T, repo *TestRepo, path string) {
	t.Helper()
	assert.True(t, repo.HasFile(path), "Expected file '%s' to exist", path)
}

// AssertFileNotExists checks that a file does not exist in the repository.
func AssertFileNotExists(t *testing.T, repo *TestRepo, path string) {
	t.Helper()
	assert.False(t, repo.HasFile(path), "Expected file '%s' not to exist", path)
}

// AssertFileContains checks that a file contains the expected content.
func AssertFileContains(t *testing.T, repo *TestRepo, path, content string) {
	t.Helper()
	assert.True(t, repo.HasFile(path), "File '%s' does not exist", path)
	if repo.HasFile(path) {
		fileContent := repo.ReadFile(path)
		assert.Contains(t, fileContent, content, "Expected file '%s' to contain '%s', got: %s", path, content, fileContent)
	}
}

// AssertCurrentBranch verifies the repository is on the expected branch.
func AssertCurrentBranch(t *testing.T, repo *TestRepo, expected string) {
	t.Helper()
	current := repo.CurrentBranch()
	assert.Equal(t, expected, current, "Expected current branch to be '%s', got: '%s'", expected, current)
}

// AssertWorktreeCount ensures the repository has the expected number of worktrees.
func AssertWorktreeCount(t *testing.T, repo *TestRepo, expected int) {
	t.Helper()
	worktrees := repo.ListWorktrees()
	assert.Len(t, worktrees, expected, "Expected %d worktrees, got %d: %v", expected, len(worktrees), worktrees)
}

// AssertWorktreeExists asserts that a worktree entry contains the provided path substring.
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

// AssertWorktreeNotExists asserts that no worktree entry contains the provided path substring.
func AssertWorktreeNotExists(t *testing.T, repo *TestRepo, path string) {
	t.Helper()
	worktrees := repo.ListWorktrees()
	for _, wt := range worktrees {
		if strings.Contains(wt, path) {
			t.Errorf("Expected worktree at path '%s' not to exist, but it does", path)
		}
	}
}

// AssertEqual compares two values for equality.
func AssertEqual(t *testing.T, expected, actual any) {
	t.Helper()
	assert.Equal(t, expected, actual)
}

// AssertNotEqual compares two values and fails if they are equal.
func AssertNotEqual(t *testing.T, notExpected, actual any) {
	t.Helper()
	assert.NotEqual(t, notExpected, actual)
}

// AssertTrue fails the test if the condition is false.
func AssertTrue(t *testing.T, condition bool, message string) {
	t.Helper()
	assert.True(t, condition, message)
}

// AssertFalse fails the test if the condition is true.
func AssertFalse(t *testing.T, condition bool, message string) {
	t.Helper()
	assert.False(t, condition, message)
}
