package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"
)

// Test helpers for integration tests
func setupTestGitRepoForAdd(t *testing.T) string {
	// Create temporary directory
	tempDir := t.TempDir()

	// Initialize git repository
	err := os.Chdir(tempDir)
	assert.NoError(t, err)

	runGitCommand(t, tempDir, "init")
	runGitCommand(t, tempDir, "config", "user.email", "test@example.com")
	runGitCommand(t, tempDir, "config", "user.name", "Test User")

	// Create initial commit
	err = os.WriteFile(filepath.Join(tempDir, "README.md"), []byte("# Test Repo"), 0644)
	assert.NoError(t, err)
	runGitCommand(t, tempDir, "add", "README.md")
	runGitCommand(t, tempDir, "commit", "-m", "Initial commit")

	return tempDir
}

func createConfigFile(t *testing.T, repoPath, content string) {
	configPath := filepath.Join(repoPath, ".wtp.yml")
	err := os.WriteFile(configPath, []byte(content), 0644)
	assert.NoError(t, err)
}

// Tests for addCommand function
func TestAddCommand_ValidationError(t *testing.T) {
	// Save current directory
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()

	// Setup test repo
	_ = setupTestGitRepoForAdd(t)

	app := &cli.Command{
		Commands: []*cli.Command{
			NewAddCommand(),
		},
	}

	var buf bytes.Buffer
	app.Writer = &buf

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "add"})

	// Should fail with validation error
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "branch name is required")
}

func TestAddCommand_Success(t *testing.T) {
	// Save current directory
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()

	// Setup test repo
	testRepo := setupTestGitRepoForAdd(t)

	// Create config file
	configContent := `version: 1
defaults:
  base_dir: "../worktrees"
`
	createConfigFile(t, testRepo, configContent)

	// Create a test branch
	runGitCommand(t, testRepo, "checkout", "-b", "feature/test")
	runGitCommand(t, testRepo, "checkout", "main")

	app := &cli.Command{
		Commands: []*cli.Command{
			NewAddCommand(),
		},
	}

	var buf bytes.Buffer
	app.Writer = &buf

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "add", "feature/test"})

	// Should succeed
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Created worktree")
}

func TestAddCommand_WithFlags(t *testing.T) {
	// Save current directory
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()

	// Setup test repo
	testRepo := setupTestGitRepoForAdd(t)

	// Create config file
	configContent := `version: 1
defaults:
  base_dir: "../worktrees"
`
	createConfigFile(t, testRepo, configContent)

	app := &cli.Command{
		Commands: []*cli.Command{
			NewAddCommand(),
		},
	}

	var buf bytes.Buffer
	app.Writer = &buf

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "add", "-b", "new-feature", "main"})

	// Should succeed
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Created worktree")
}

func TestSetupRepoAndConfig_NotInGitRepo(t *testing.T) {
	// Save current directory
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()

	// Change to temp directory that's not a git repo
	tempDir := t.TempDir()
	err := os.Chdir(tempDir)
	assert.NoError(t, err)

	//nolint:dogsled
	_, _, _, err = setupRepoAndConfig()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in a git repository")
}

func TestSetupRepoAndConfig_Success(t *testing.T) {
	// Save current directory
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()

	// Setup test repo
	testRepo := setupTestGitRepoForAdd(t)

	// Create config file
	configContent := `version: 1
defaults:
  base_dir: "../worktrees"
`
	createConfigFile(t, testRepo, configContent)

	repo, cfg, mainRepoPath, err := setupRepoAndConfig()
	assert.NoError(t, err)
	assert.NotNil(t, repo)
	assert.NotNil(t, cfg)
	assert.NotEmpty(t, mainRepoPath)
	assert.Equal(t, "../worktrees", cfg.Defaults.BaseDir)
}

func TestHandleBranchResolution_NewBranch(t *testing.T) {
	// Save current directory
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()

	// Setup test repo
	_ = setupTestGitRepoForAdd(t)

	// Create a mock CLI command with branch flag
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "branch"},
		},
	}
	err := cmd.Set("branch", "new-feature")
	assert.NoError(t, err)

	// Setup repository
	repo, _, _, err := setupRepoAndConfig()
	assert.NoError(t, err)

	// Test branch resolution - should succeed for new branch
	gitExec := newRepositoryExecutor(repo)
	err = handleBranchResolutionWithExecutor(cmd, gitExec, "")
	assert.NoError(t, err)
}

func TestHandleBranchResolution_ExistingBranch(t *testing.T) {
	// Save current directory
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()

	// Setup test repo
	testRepo := setupTestGitRepoForAdd(t)

	// Create existing branch
	runGitCommand(t, testRepo, "checkout", "-b", "existing-branch")
	runGitCommand(t, testRepo, "checkout", "main")

	// Create a mock CLI command without branch flag
	cmd := &cli.Command{
		Flags: []cli.Flag{
			&cli.StringFlag{Name: "branch"},
			&cli.StringFlag{Name: "track"},
		},
	}

	// Setup repository
	repo, _, _, err := setupRepoAndConfig()
	assert.NoError(t, err)

	// Test branch resolution - should succeed for existing branch
	gitExec := newRepositoryExecutor(repo)
	err = handleBranchResolutionWithExecutor(cmd, gitExec, "existing-branch")
	assert.NoError(t, err)
}

// Test removeBranch function
func TestRemoveBranch_Success(t *testing.T) {
	// Save current directory
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()

	// Setup test repo
	testRepo := setupTestGitRepoForAdd(t)

	// Create a test branch
	runGitCommand(t, testRepo, "checkout", "-b", "test-branch")
	runGitCommand(t, testRepo, "checkout", "main")

	// Setup repository
	repo, _, _, err := setupRepoAndConfig()
	assert.NoError(t, err)

	var buf bytes.Buffer

	// Test removing the branch
	err = removeBranch(&buf, repo, "test-branch", false)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Removed branch 'test-branch'")
}

func TestRemoveBranch_NotMerged(t *testing.T) {
	// Save current directory
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()

	// Setup test repo
	testRepo := setupTestGitRepoForAdd(t)

	// Create a test branch with changes
	runGitCommand(t, testRepo, "checkout", "-b", "unmerged-branch")
	err := os.WriteFile(filepath.Join(testRepo, "test.txt"), []byte("test"), 0644)
	assert.NoError(t, err)
	runGitCommand(t, testRepo, "add", "test.txt")
	runGitCommand(t, testRepo, "commit", "-m", "Add test file")
	runGitCommand(t, testRepo, "checkout", "main")

	// Setup repository
	repo, _, _, err := setupRepoAndConfig()
	assert.NoError(t, err)

	var buf bytes.Buffer

	// Test removing unmerged branch - should fail
	err = removeBranch(&buf, repo, "unmerged-branch", false)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove branch")
}

func TestRemoveBranch_Forced(t *testing.T) {
	// Save current directory
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()

	// Setup test repo
	testRepo := setupTestGitRepoForAdd(t)

	// Create a test branch with changes
	runGitCommand(t, testRepo, "checkout", "-b", "unmerged-branch")
	err := os.WriteFile(filepath.Join(testRepo, "test.txt"), []byte("test"), 0644)
	assert.NoError(t, err)
	runGitCommand(t, testRepo, "add", "test.txt")
	runGitCommand(t, testRepo, "commit", "-m", "Add test file")
	runGitCommand(t, testRepo, "checkout", "main")

	// Setup repository
	repo, _, _, err := setupRepoAndConfig()
	assert.NoError(t, err)

	var buf bytes.Buffer

	// Test force removing unmerged branch - should succeed
	err = removeBranch(&buf, repo, "unmerged-branch", true)
	assert.NoError(t, err)

	output := buf.String()
	assert.Contains(t, output, "Removed branch 'unmerged-branch'")
}

func TestAddCommand_WithHooks(t *testing.T) {
	// Save current directory
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()

	// Setup test repo
	testRepo := setupTestGitRepoForAdd(t)

	// Create config file with hooks
	configContent := fmt.Sprintf(`version: 1
defaults:
  base_dir: "../worktrees"
hooks:
  post_create:
    - type: copy
      from: "%s/README.md"
      to: "copied_readme.md"
`, testRepo)
	createConfigFile(t, testRepo, configContent)

	// Create a test branch
	runGitCommand(t, testRepo, "checkout", "-b", "hook-test")
	runGitCommand(t, testRepo, "checkout", "main")

	app := &cli.Command{
		Commands: []*cli.Command{
			NewAddCommand(),
		},
	}

	var buf bytes.Buffer
	app.Writer = &buf

	ctx := context.Background()
	err := app.Run(ctx, []string{"wtp", "add", "hook-test"})

	// Should succeed
	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Created worktree")

	// Check if hook was executed
	worktreePath := filepath.Join(filepath.Dir(testRepo), "worktrees", "hook-test")
	copiedFile := filepath.Join(worktreePath, "copied_readme.md")
	assert.FileExists(t, copiedFile)
}
