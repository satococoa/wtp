package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/satococoa/wtp/v2/internal/config"
)

func TestWarnLegacyBaseDir_WarnsWhenLegacyDirExistsAndNoConfig(t *testing.T) {
	baseDir := t.TempDir()
	repoRoot := filepath.Join(baseDir, "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("failed to create repo root: %v", err)
	}

	legacyDir := filepath.Join(repoRoot, "..", "worktrees")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatalf("failed to create legacy dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "placeholder"), []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to write placeholder: %v", err)
	}

	var buf bytes.Buffer
	warnLegacyBaseDir(&buf, repoRoot)

	output := buf.String()
	if output == "" {
		t.Fatalf("expected warning output, got empty string")
	}
	if !bytes.Contains(buf.Bytes(), []byte("legacy worktrees directory")) {
		t.Errorf("expected warning message to mention legacy directory, got: %s", output)
	}
	if !bytes.Contains(buf.Bytes(), []byte(config.ConfigFileName)) {
		t.Errorf("expected warning to mention config file, got: %s", output)
	}
}

func TestWarnLegacyBaseDir_NoWarningWhenConfigExists(t *testing.T) {
	baseDir := t.TempDir()
	repoRoot := filepath.Join(baseDir, "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("failed to create repo root: %v", err)
	}

	configPath := filepath.Join(repoRoot, config.ConfigFileName)
	if err := os.WriteFile(configPath, []byte("version: 1.0\n"), 0o644); err != nil {
		t.Fatalf("failed to write config: %v", err)
	}

	legacyDir := filepath.Join(repoRoot, "..", "worktrees")
	if err := os.MkdirAll(legacyDir, 0o755); err != nil {
		t.Fatalf("failed to create legacy dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(legacyDir, "placeholder"), []byte("x"), 0o644); err != nil {
		t.Fatalf("failed to write placeholder: %v", err)
	}

	var buf bytes.Buffer
	warnLegacyBaseDir(&buf, repoRoot)

	if buf.Len() != 0 {
		t.Fatalf("expected no warning output, got: %s", buf.String())
	}
}

func TestWarnLegacyBaseDir_NoWarningWhenLegacyMissing(t *testing.T) {
	baseDir := t.TempDir()
	repoRoot := filepath.Join(baseDir, "repo")
	if err := os.MkdirAll(repoRoot, 0o755); err != nil {
		t.Fatalf("failed to create repo root: %v", err)
	}

	var buf bytes.Buffer
	warnLegacyBaseDir(&buf, repoRoot)

	if buf.Len() != 0 {
		t.Fatalf("expected no warning output, got: %s", buf.String())
	}
}
