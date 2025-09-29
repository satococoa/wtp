package main

import (
	"bytes"
	"io"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/satococoa/wtp/internal/config"
)

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

// RunWithSilencedStdout executes the provided function while temporarily
// redirecting os.Stdout to an OS-specific null device. This helps tests avoid
// polluting output when exercising code paths that print directly to stdout.
func RunWithSilencedStdout(t *testing.T, fn func()) {
	t.Helper()

	devNull, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("open devnull: %v", err)
	}

	oldStdout := os.Stdout
	os.Stdout = devNull
	t.Cleanup(func() {
		os.Stdout = oldStdout
		_ = devNull.Close()
	})

	fn()
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
