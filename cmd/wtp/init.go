package main

import (
	"context"
	"fmt"
	"os"

	"github.com/satococoa/wtp/internal/config"
	"github.com/satococoa/wtp/internal/errors"
	"github.com/satococoa/wtp/internal/git"
	"github.com/urfave/cli/v3"
)

const configFileMode = 0o600

// Variable to allow mocking in tests
var osGetwd = os.Getwd

// NewInitCommand creates the init command definition
func NewInitCommand() *cli.Command {
	return &cli.Command{
		Name:  "init",
		Usage: "Initialize configuration file",
		Description: "Creates a .wtp.yml configuration file in the repository root " +
			"with example hooks and settings.",
		Action: initCommand,
	}
}

func initCommand(_ context.Context, cmd *cli.Command) error {
	// Get current working directory (should be a git repository)
	cwd, err := osGetwd()
	if err != nil {
		return errors.DirectoryAccessFailed("access current", ".", err)
	}

	// Initialize repository
	repo, err := git.NewRepository(cwd)
	if err != nil {
		return errors.NotInGitRepository()
	}

	// Check if config file already exists
	configPath := fmt.Sprintf("%s/%s", repo.Path(), config.ConfigFileName)
	if _, err := os.Stat(configPath); err == nil {
		return errors.ConfigAlreadyExists(configPath)
	}

	// Create configuration with comments
	configContent := `# Worktree Plus Configuration
version: "1.0"

# Default settings for worktrees
defaults:
  # Base directory for worktrees (relative to repository root)
  base_dir: ../worktrees

  # Automatically change to the new worktree directory after creation
  cd_after_create: true

# Hooks that run after creating a worktree
hooks:
  post_create:
    # Example: Copy environment file
    - type: copy
      from: .env.example
      to: .env

    # Example: Run a command to show all worktrees
    - type: command
      command: wtp list

    # More examples (commented out):
    # - type: command
    #   command: echo "Created new worktree!"
    # - type: command
    #   command: ls -la
    # - type: command
    #   command: npm install
`

	// Write configuration file with comments
	if err := os.WriteFile(configPath, []byte(configContent), configFileMode); err != nil {
		return errors.DirectoryAccessFailed("create configuration file", configPath, err)
	}

	// Get the writer from cli.Command
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	fmt.Fprintf(w, "Configuration file created: %s\n", configPath)
	fmt.Fprintln(w, "Edit this file to customize your worktree setup.")
	return nil
}
