package framework

import (
	"strings"
	"testing"
)

func AssertWorktreeCreated(t *testing.T, output string, branch string) {
	t.Helper()
	if !strings.Contains(output, "Created worktree") && !strings.Contains(output, "Preparing worktree") {
		t.Errorf("Expected worktree creation message, got: %s", output)
	}
	if !strings.Contains(output, branch) {
		t.Errorf("Expected branch name '%s' in output, got: %s", branch, output)
	}
}

func AssertErrorContains(t *testing.T, err error, expected string) {
	t.Helper()
	if err == nil {
		t.Errorf("Expected error containing '%s', but got no error", expected)
		return
	}
	if !strings.Contains(err.Error(), expected) {
		t.Errorf("Expected error containing '%s', got: %v", expected, err)
	}
}

func AssertOutputContains(t *testing.T, output string, expected string) {
	t.Helper()
	if !strings.Contains(output, expected) {
		t.Errorf("Expected output containing '%s', got: %s", expected, output)
	}
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
		if !strings.Contains(output, exp) {
			t.Errorf("Expected output to contain '%s', got: %s", exp, output)
		}
	}
}

func AssertNoError(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Errorf("Expected no error, got: %v", err)
	}
}

func AssertError(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		t.Error("Expected error, but got none")
	}
}

func AssertFileExists(t *testing.T, repo *TestRepo, path string) {
	t.Helper()
	if !repo.HasFile(path) {
		t.Errorf("Expected file '%s' to exist", path)
	}
}

func AssertFileNotExists(t *testing.T, repo *TestRepo, path string) {
	t.Helper()
	if repo.HasFile(path) {
		t.Errorf("Expected file '%s' not to exist", path)
	}
}

func AssertFileContains(t *testing.T, repo *TestRepo, path string, content string) {
	t.Helper()
	if !repo.HasFile(path) {
		t.Errorf("File '%s' does not exist", path)
		return
	}
	fileContent := repo.ReadFile(path)
	if !strings.Contains(fileContent, content) {
		t.Errorf("Expected file '%s' to contain '%s', got: %s", path, content, fileContent)
	}
}

func AssertCurrentBranch(t *testing.T, repo *TestRepo, expected string) {
	t.Helper()
	current := repo.CurrentBranch()
	if current != expected {
		t.Errorf("Expected current branch to be '%s', got: '%s'", expected, current)
	}
}

func AssertWorktreeCount(t *testing.T, repo *TestRepo, expected int) {
	t.Helper()
	worktrees := repo.ListWorktrees()
	if len(worktrees) != expected {
		t.Errorf("Expected %d worktrees, got %d: %v", expected, len(worktrees), worktrees)
	}
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

func AssertEqual(t *testing.T, expected, actual interface{}) {
	t.Helper()
	if expected != actual {
		t.Errorf("Expected %v, got %v", expected, actual)
	}
}

func AssertNotEqual(t *testing.T, notExpected, actual interface{}) {
	t.Helper()
	if notExpected == actual {
		t.Errorf("Expected value to not be %v", actual)
	}
}

func AssertTrue(t *testing.T, condition bool, message string) {
	t.Helper()
	if !condition {
		t.Error(message)
	}
}

func AssertFalse(t *testing.T, condition bool, message string) {
	t.Helper()
	if condition {
		t.Error(message)
	}
}
