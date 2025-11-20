package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/satococoa/wtp/internal/config"
	"github.com/satococoa/wtp/internal/git"
)

func TestDetectLegacyWorktreeMigrations(t *testing.T) {
	tempDir := t.TempDir()
	mainRepoPath := filepath.Join(tempDir, "repo")
	requireNoErr(t, os.MkdirAll(mainRepoPath, 0o755))

	cfg := &config.Config{
		Defaults: config.Defaults{BaseDir: config.DefaultBaseDir},
	}

	legacyWorktreePath := filepath.Join(filepath.Dir(mainRepoPath), "worktrees", "feature", "foo")
	modernWorktreePath := filepath.Join(
		filepath.Dir(mainRepoPath),
		"worktrees",
		filepath.Base(mainRepoPath),
		"feature",
		"bar",
	)

	worktrees := []git.Worktree{
		{Path: mainRepoPath, IsMain: true},
		{Path: legacyWorktreePath},
		{Path: modernWorktreePath},
	}

	migrations := detectLegacyWorktreeMigrations(mainRepoPath, cfg, worktrees)

	assert.Len(t, migrations, 1)
	expectedCurrent := filepath.Join("..", "worktrees", "feature", "foo")
	expectedSuggested := filepath.Join("..", "worktrees", filepath.Base(mainRepoPath), "feature", "foo")

	assert.Equal(t, filepath.Clean(expectedCurrent), migrations[0].currentRel)
	assert.Equal(t, filepath.Clean(expectedSuggested), migrations[0].suggestedRel)
}

func TestMaybeWarnLegacyWorktreeLayout_EmitsWarningWithoutConfig(t *testing.T) {
	tempDir := t.TempDir()
	mainRepoPath := filepath.Join(tempDir, "repo")
	requireNoErr(t, os.MkdirAll(mainRepoPath, 0o755))

	cfg := &config.Config{
		Defaults: config.Defaults{BaseDir: config.DefaultBaseDir},
	}

	legacyWorktree := filepath.Join(filepath.Dir(mainRepoPath), "worktrees", "feature", "foo")
	worktrees := []git.Worktree{
		{Path: mainRepoPath, IsMain: true},
		{Path: legacyWorktree},
	}

	var buf bytes.Buffer
	maybeWarnLegacyWorktreeLayout(&buf, mainRepoPath, cfg, worktrees)

	output := buf.String()
	assert.NotEmpty(t, output)
	assert.Contains(t, output, "Legacy worktree layout detected")

	moveCommand := strings.Join([]string{
		"git worktree move",
		filepath.Join("..", "worktrees", "feature", "foo"),
		filepath.Join("..", "worktrees", filepath.Base(mainRepoPath), "feature", "foo"),
	}, " ")
	assert.Contains(t, output, moveCommand)
}

func TestMaybeWarnLegacyWorktreeLayout_SuppressedWhenConfigPresent(t *testing.T) {
	tempDir := t.TempDir()
	mainRepoPath := filepath.Join(tempDir, "repo")
	requireNoErr(t, os.MkdirAll(mainRepoPath, 0o755))

	configPath := filepath.Join(mainRepoPath, config.ConfigFileName)
	requireNoErr(t, os.WriteFile(configPath, []byte("version: 1.0\n"), 0o600))

	cfg := &config.Config{
		Defaults: config.Defaults{BaseDir: config.DefaultBaseDir},
	}

	worktrees := []git.Worktree{
		{Path: mainRepoPath, IsMain: true},
		{Path: filepath.Join(filepath.Dir(mainRepoPath), "worktrees", "feature", "foo")},
	}

	var buf bytes.Buffer
	maybeWarnLegacyWorktreeLayout(&buf, mainRepoPath, cfg, worktrees)

	assert.Empty(t, buf.String())
}

func requireNoErr(t *testing.T, err error) {
	t.Helper()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
