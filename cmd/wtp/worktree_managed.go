package main

import (
	"path/filepath"
	"strings"

	"github.com/satococoa/wtp/v2/internal/config"
)

// isWorktreeManagedCommon determines whether a worktree path is considered managed by wtp.
// The logic is shared across multiple commands so that we consistently classify worktrees.
func isWorktreeManagedCommon(worktreePath string, cfg *config.Config, mainRepoPath string, isMain bool) bool {
	if isMain {
		return true
	}

	// Fallback to default configuration if none is provided
	if cfg == nil {
		cfg = &config.Config{
			Defaults: config.Defaults{
				BaseDir: config.DefaultBaseDir,
			},
		}
	}

	baseDir := cfg.ResolveWorktreePath(mainRepoPath, "")
	baseDir = strings.TrimSuffix(baseDir, string(filepath.Separator))

	// Normalize path separators for cross-platform compatibility.
	// On Windows, git commands return paths with forward slashes (e.g., "D:/foo/bar")
	// while Go's filepath functions use backslashes (e.g., "D:\foo\bar").
	// filepath.Clean normalizes these to the OS-native separator.
	absWorktreePath, err := filepath.Abs(filepath.Clean(worktreePath))
	if err != nil {
		return false
	}

	absBaseDir, err := filepath.Abs(filepath.Clean(baseDir))
	if err != nil {
		return false
	}

	// Resolve symlinks to handle cases where paths may point to the same location
	// via different routes (e.g., C:\Users\...\Documents -> D:\Documents on Windows).
	// If EvalSymlinks fails, fall back to the original absolute path.
	if resolved, err := filepath.EvalSymlinks(absWorktreePath); err == nil {
		absWorktreePath = resolved
	}
	if resolved, err := filepath.EvalSymlinks(absBaseDir); err == nil {
		absBaseDir = resolved
	}

	relPath, err := filepath.Rel(absBaseDir, absWorktreePath)
	if err != nil {
		return false
	}

	if relPath == "." || relPath == "" {
		return true
	}

	if relPath == ".." || strings.HasPrefix(relPath, ".."+string(filepath.Separator)) {
		return false
	}

	return true
}
