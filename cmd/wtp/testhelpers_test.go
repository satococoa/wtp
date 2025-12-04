package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/satococoa/wtp/v2/internal/config"
)

// TestMain sets up the test environment for all tests in the package.
// It isolates tests from the user's real global config by setting XDG_CONFIG_HOME
// to a temporary directory.
func TestMain(m *testing.M) {
	// Create a temp directory for XDG_CONFIG_HOME to isolate tests
	// from the user's real global config
	tmpDir, err := os.MkdirTemp("", "wtp-test-config-*")
	if err != nil {
		os.Exit(1)
	}

	if err := os.Setenv("XDG_CONFIG_HOME", tmpDir); err != nil {
		_ = os.RemoveAll(tmpDir)
		os.Exit(1)
	}

	code := m.Run()

	// Cleanup
	_ = os.Unsetenv("XDG_CONFIG_HOME")
	_ = os.RemoveAll(tmpDir)

	os.Exit(code)
}

// RunWriterCommonTests runs a common pair of tests for functions that write
// to an io.Writer and may interact with a Git repo. It validates that the
// function does not panic in non-repo contexts and when a bare .git dir exists.
func RunWriterCommonTests(t *testing.T, name string, fn func(io.Writer) error) {
	t.Helper()

	t.Run(name+": should write to writer without panic", func(t *testing.T) {
		var buf bytes.Buffer
		assert.NotPanics(t, func() { _ = fn(&buf) })
	})

	t.Run(name+": should handle git directory gracefully", func(t *testing.T) {
		tempDir := t.TempDir()
		gitDir := filepath.Join(tempDir, ".git")
		assert.NoError(t, os.MkdirAll(gitDir, 0o755))

		oldDir, _ := os.Getwd()
		t.Cleanup(func() { _ = os.Chdir(oldDir) })
		assert.NoError(t, os.Chdir(tempDir))

		var buf bytes.Buffer
		assert.NotPanics(t, func() { _ = fn(&buf) })
	})
}

// RunNameFromPathTests executes a shared set of assertions for worktree
// naming helpers that map absolute paths to display names.
func RunNameFromPathTests(
	t *testing.T,
	label string,
	fn func(worktreePath string, cfg *config.Config, mainRepoPath string, isMain bool) string,
) {
	t.Helper()

	t.Run(label+": main worktree returns @", func(t *testing.T) {
		cfg := &config.Config{Defaults: config.Defaults{BaseDir: ".worktrees"}}
		name := fn("/path/to/repo", cfg, "/path/to/repo", true)
		assert.Equal(t, "@", name)
	})

	t.Run(label+": non-main returns relative path", func(t *testing.T) {
		cfg := &config.Config{Defaults: config.Defaults{BaseDir: ".worktrees"}}
		name := fn("/path/to/repo/.worktrees/feature/test", cfg, "/path/to/repo", false)
		assert.Equal(t, "feature/test", name)
	})

	t.Run(label+": outside base_dir returns relative-to-base", func(t *testing.T) {
		cfg := &config.Config{Defaults: config.Defaults{BaseDir: ".worktrees"}}
		// When worktree is outside base_dir, filepath.Rel returns a relative path
		// with .. segments; this should be surfaced as-is.
		name := fn("/completely/different/path", cfg, "/path/to/repo", false)
		assert.Equal(t, "../../../../completely/different/path", name)
	})
}
