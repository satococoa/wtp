package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/satococoa/wtp/v2/internal/config"
)

const (
	legacyWarningMessage = "Warning: detected legacy worktrees directory at %s\n"
	legacyConfigMessage  = "Set base_dir: \"../worktrees\" in %s to keep using it, or move worktrees to %s\n"
)

func warnLegacyBaseDir(errWriter io.Writer, repoRoot string) {
	if errWriter == nil {
		errWriter = os.Stderr
	}

	if config.FileExists(repoRoot) {
		return
	}

	legacyBaseDir := filepath.Clean(filepath.Join(repoRoot, "..", "worktrees"))
	info, err := os.Stat(legacyBaseDir)
	if err != nil || !info.IsDir() {
		return
	}

	entries, err := os.ReadDir(legacyBaseDir)
	if err != nil || len(entries) == 0 {
		return
	}

	newDefault := filepath.Clean(filepath.Join(repoRoot, config.DefaultBaseDir))
	configPath := filepath.Join(repoRoot, config.ConfigFileName)

	if _, err := fmt.Fprintf(errWriter, legacyWarningMessage, legacyBaseDir); err != nil {
		return
	}
	if _, err := fmt.Fprintf(errWriter, legacyConfigMessage, configPath, newDefault); err != nil {
		return
	}
}
