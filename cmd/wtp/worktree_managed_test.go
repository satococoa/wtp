package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/satococoa/wtp/internal/config"
)

func TestIsWorktreeManagedCommon_ReturnsTrueForManagedWorktree(t *testing.T) {
	tempDir := t.TempDir()
	mainRepoPath := filepath.Join(tempDir, "repo")
	worktreePath := filepath.Join(tempDir, "worktrees", "repo", "feature", "foo")

	require.NoError(t, os.MkdirAll(mainRepoPath, 0o755))

	cfg := &config.Config{
		Defaults: config.Defaults{
			BaseDir: config.DefaultBaseDir,
		},
	}

	assert.True(t, isWorktreeManagedCommon(worktreePath, cfg, mainRepoPath, false))
}

func TestIsWorktreeManagedCommon_ReturnsFalseOutsideBaseDir(t *testing.T) {
	tempDir := t.TempDir()
	mainRepoPath := filepath.Join(tempDir, "repo")
	worktreePath := filepath.Join(tempDir, "external", "feature", "foo")

	require.NoError(t, os.MkdirAll(mainRepoPath, 0o755))

	cfg := &config.Config{
		Defaults: config.Defaults{
			BaseDir: config.DefaultBaseDir,
		},
	}

	assert.False(t, isWorktreeManagedCommon(worktreePath, cfg, mainRepoPath, false))
}

func TestIsWorktreeManagedCommon_HandlesRelativeWorktreePaths(t *testing.T) {
	tempDir := t.TempDir()
	mainRepoPath := filepath.Join(tempDir, "repo")
	cfg := &config.Config{
		Defaults: config.Defaults{
			BaseDir: config.DefaultBaseDir,
		},
	}

	cwd, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() {
		require.NoError(t, os.Chdir(cwd))
	})

	require.NoError(t, os.MkdirAll(mainRepoPath, 0o755))

	canonicalMainRepoPath, err := filepath.EvalSymlinks(mainRepoPath)
	require.NoError(t, err)

	require.NoError(t, os.Chdir(canonicalMainRepoPath))

	relativePath := filepath.Join("..", "worktrees", "repo", "feature", "bar")

	baseDir := cfg.ResolveWorktreePath(canonicalMainRepoPath, "")
	baseDir = strings.TrimSuffix(baseDir, string(filepath.Separator))
	baseDir = filepath.Clean(baseDir)

	absWorktreePath, err := filepath.Abs(relativePath)
	require.NoError(t, err)

	relPath, err := filepath.Rel(baseDir, absWorktreePath)
	require.NoError(t, err)

	require.NotEqual(t, "..", relPath)
	require.False(t, strings.HasPrefix(relPath, ".."+string(filepath.Separator)))

	assert.True(t, isWorktreeManagedCommon(relativePath, cfg, canonicalMainRepoPath, false))
}
