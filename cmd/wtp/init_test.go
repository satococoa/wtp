package main

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"

	"github.com/satococoa/wtp/internal/config"
)

func TestNewInitCommand(t *testing.T) {
	cmd := NewInitCommand()

	assert.NotNil(t, cmd)
	assert.Equal(t, "init", cmd.Name)
	assert.Equal(t, "Initialize configuration file", cmd.Usage)
	assert.NotEmpty(t, cmd.Description)
	assert.NotNil(t, cmd.Action)
}

func TestConfigFileMode(t *testing.T) {
	assert.Equal(t, os.FileMode(0o600), os.FileMode(configFileMode))
}

func TestInitCommand_NotInGitRepo(t *testing.T) {
	// Create a temporary directory that is not a git repo
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	err := os.Chdir(tempDir)
	assert.NoError(t, err)

	app := &cli.Command{
		Commands: []*cli.Command{
			NewInitCommand(),
		},
	}

	ctx := context.Background()
	err = app.Run(ctx, []string{"wtp", "init"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in a git repository")
}

func TestInitCommand_ConfigAlreadyExists(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	err := os.Chdir(tempDir)
	assert.NoError(t, err)

	// Initialize as a git repository
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = tempDir
	err = gitCmd.Run()
	if err != nil {
		t.Skip("git not available")
	}

	// Create existing config file
	configPath := filepath.Join(tempDir, config.ConfigFileName)
	err = os.WriteFile(configPath, []byte("existing config"), 0644)
	assert.NoError(t, err)

	app := &cli.Command{
		Commands: []*cli.Command{
			NewInitCommand(),
		},
	}

	ctx := context.Background()
	err = app.Run(ctx, []string{"wtp", "init"})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestInitCommand_Success(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	err := os.Chdir(tempDir)
	assert.NoError(t, err)

	// Initialize as a git repository
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = tempDir
	err = gitCmd.Run()
	if err != nil {
		t.Skip("git not available")
	}

	app := &cli.Command{
		Commands: []*cli.Command{
			NewInitCommand(),
		},
	}

	var buf bytes.Buffer
	app.Writer = &buf

	ctx := context.Background()
	err = app.Run(ctx, []string{"wtp", "init"})
	assert.NoError(t, err)

	// Check output
	output := buf.String()
	assert.Contains(t, output, "Configuration file created:")
	assert.Contains(t, output, config.ConfigFileName)
	assert.Contains(t, output, "Edit this file to customize your worktree setup.")

	// Verify config file was created
	configPath := filepath.Join(tempDir, config.ConfigFileName)
	info, err := os.Stat(configPath)
	assert.NoError(t, err)
	assert.False(t, info.IsDir())

	// Check file permissions
	assert.Equal(t, os.FileMode(0o600), info.Mode().Perm())

	// Verify content
	content, err := os.ReadFile(configPath)
	assert.NoError(t, err)
	contentStr := string(content)

	// Check for required sections
	assert.Contains(t, contentStr, "version: \"1.0\"")
	assert.Contains(t, contentStr, "defaults:")
	assert.Contains(t, contentStr, "base_dir: ../worktrees")
	assert.Contains(t, contentStr, "cd_after_create: true")
	assert.Contains(t, contentStr, "hooks:")
	assert.Contains(t, contentStr, "post_create:")

	// Check for example hooks
	assert.Contains(t, contentStr, "type: copy")
	assert.Contains(t, contentStr, "from: .env.example")
	assert.Contains(t, contentStr, "to: .env")
	assert.Contains(t, contentStr, "type: command")
	assert.Contains(t, contentStr, "command: wtp list")

	// Check for comments
	assert.Contains(t, contentStr, "# Worktree Plus Configuration")
	assert.Contains(t, contentStr, "# Default settings for worktrees")
	assert.Contains(t, contentStr, "# Hooks that run after creating a worktree")
}

func TestInitCommand_DirectoryAccessError(t *testing.T) {
	// Save original os.Getwd to restore later
	originalGetwd := osGetwd
	defer func() { osGetwd = originalGetwd }()

	// Mock os.Getwd to return an error
	osGetwd = func() (string, error) {
		return "", assert.AnError
	}

	cmd := NewInitCommand()
	ctx := context.Background()
	err := cmd.Action(ctx, &cli.Command{})

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to access")
}

func TestInitCommand_WriteFileError(t *testing.T) {
	// Create a temporary directory
	tempDir := t.TempDir()

	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	err := os.Chdir(tempDir)
	assert.NoError(t, err)

	// Initialize as a git repository
	gitCmd := exec.Command("git", "init")
	gitCmd.Dir = tempDir
	err = gitCmd.Run()
	if err != nil {
		t.Skip("git not available")
	}

	// Make directory read-only to cause write error
	err = os.Chmod(tempDir, 0555)
	assert.NoError(t, err)
	defer func() { _ = os.Chmod(tempDir, 0755) }() // Restore permissions for cleanup

	cmd := NewInitCommand()
	ctx := context.Background()
	err = cmd.Action(ctx, &cli.Command{})

	assert.Error(t, err)
	// The error could be either "failed to access" or "failed to create"
	errorMsg := err.Error()
	assert.True(t, strings.Contains(errorMsg, "failed to") &&
		(strings.Contains(errorMsg, "access") || strings.Contains(errorMsg, "create")))
}
