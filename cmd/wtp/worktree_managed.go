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

	absWorktreePath, err := filepath.Abs(worktreePath)
	if err != nil {
		return false
	}

	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return false
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
