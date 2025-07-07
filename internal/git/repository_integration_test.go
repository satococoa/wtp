package git

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetMainWorktreePath(t *testing.T) {
	// Create test repository
	tempDir := setupTestRepo(t)

	// Change to the repo directory
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	err := os.Chdir(tempDir)
	assert.NoError(t, err)

	// Create repository instance
	repo, err := NewRepository(tempDir)
	assert.NoError(t, err)

	// Test GetMainWorktreePath
	mainPath, err := repo.GetMainWorktreePath()
	assert.NoError(t, err)
	// Use filepath.EvalSymlinks to handle /private prefix on macOS
	expectedPath, _ := filepath.EvalSymlinks(tempDir)
	actualPath, _ := filepath.EvalSymlinks(mainPath)
	assert.Equal(t, expectedPath, actualPath)
}

func TestGetWorktrees_NoWorktrees(t *testing.T) {
	// Create test repository
	tempDir := setupTestRepo(t)

	// Change to the repo directory
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	err := os.Chdir(tempDir)
	assert.NoError(t, err)

	// Create repository instance
	repo, err := NewRepository(tempDir)
	assert.NoError(t, err)

	// Test GetWorktrees - should return main worktree only
	worktrees, err := repo.GetWorktrees()
	assert.NoError(t, err)
	assert.Len(t, worktrees, 1)
	// Handle symlink differences on macOS
	expectedPath, _ := filepath.EvalSymlinks(tempDir)
	actualPath, _ := filepath.EvalSymlinks(worktrees[0].Path)
	assert.Equal(t, expectedPath, actualPath)
	assert.Equal(t, "main", worktrees[0].Branch)
}

func TestGetWorktrees_WithWorktrees(t *testing.T) {
	// Create test repository
	tempDir := setupTestRepo(t)

	// Change to the repo directory
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	err := os.Chdir(tempDir)
	assert.NoError(t, err)

	// Create some test branches
	runCmd(t, tempDir, "git", "checkout", "-b", "feature/test")
	runCmd(t, tempDir, "git", "checkout", "main")

	// Create worktrees
	worktreesDir := filepath.Join(filepath.Dir(tempDir), "worktrees")
	err = os.MkdirAll(worktreesDir, 0755)
	assert.NoError(t, err)

	worktree1Path := filepath.Join(worktreesDir, "feature-test")
	runCmd(t, tempDir, "git", "worktree", "add", worktree1Path, "feature/test")

	// Create repository instance
	repo, err := NewRepository(tempDir)
	assert.NoError(t, err)

	// Test GetWorktrees - should return main + worktree
	worktrees, err := repo.GetWorktrees()
	assert.NoError(t, err)
	assert.Len(t, worktrees, 2)

	// Check main worktree
	var mainWorktree, featureWorktree *Worktree
	for i := range worktrees {
		if worktrees[i].Branch == "main" {
			mainWorktree = &worktrees[i]
		} else if worktrees[i].Branch == "feature/test" {
			featureWorktree = &worktrees[i]
		}
	}

	assert.NotNil(t, mainWorktree)
	expectedMainPath, _ := filepath.EvalSymlinks(tempDir)
	actualMainPath, _ := filepath.EvalSymlinks(mainWorktree.Path)
	assert.Equal(t, expectedMainPath, actualMainPath)
	assert.Equal(t, "main", mainWorktree.Branch)

	assert.NotNil(t, featureWorktree)
	expectedFeaturePath, _ := filepath.EvalSymlinks(worktree1Path)
	actualFeaturePath, _ := filepath.EvalSymlinks(featureWorktree.Path)
	assert.Equal(t, expectedFeaturePath, actualFeaturePath)
	assert.Equal(t, "feature/test", featureWorktree.Branch)
}

func TestGetWorktrees_GitCommandFailure(t *testing.T) {
	// Create a directory that's not a git repository
	tempDir := t.TempDir()

	// Change to the non-git directory
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	err := os.Chdir(tempDir)
	assert.NoError(t, err)

	// Create repository instance (this will fail at git command execution)
	repo := &Repository{path: tempDir}

	// Test GetWorktrees - should fail
	_, err = repo.GetWorktrees()
	assert.Error(t, err)
}

func TestGetMainWorktreePath_GitCommandFailure(t *testing.T) {
	// Create a directory that's not a git repository
	tempDir := t.TempDir()

	// Change to the non-git directory
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	err := os.Chdir(tempDir)
	assert.NoError(t, err)

	// Create repository instance (this will fail at git command execution)
	repo := &Repository{path: tempDir}

	// Test GetMainWorktreePath - should fail
	_, err = repo.GetMainWorktreePath()
	assert.Error(t, err)
}
