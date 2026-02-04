package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/urfave/cli/v3"

	"github.com/satococoa/wtp/v2/internal/config"
	"github.com/satococoa/wtp/v2/internal/errors"
	"github.com/satococoa/wtp/v2/internal/git"
)

const configFileMode = 0o600

// Variable to allow mocking in tests
var osGetwd = os.Getwd
var writeFile = os.WriteFile

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
	if _, statErr := os.Stat(configPath); statErr == nil {
		return errors.ConfigAlreadyExists(configPath)
	}

	repoInfo, repoStatErr := os.Stat(repo.Path())
	if repoStatErr != nil {
		return errors.DirectoryAccessFailed("access repository", repo.Path(), repoStatErr)
	}

	if repoInfo.Mode().Perm()&0o222 == 0 {
		return errors.DirectoryAccessFailed(
			"create configuration file",
			configPath,
			fmt.Errorf("repository directory is read-only"),
		)
	}

	// Create configuration with comments
	configContent := `# Worktree Plus Configuration
version: "1.0"

# Default settings for worktrees
defaults:
  # Base directory for worktrees (relative to repository root)
  base_dir: .git/wtp/worktrees

# Hooks that run after creating a worktree
hooks:
  post_create:
    # Example: Copy gitignored files from MAIN worktree to new worktree
    # Note: 'from' is relative to main worktree, 'to' is relative to new worktree
    # - type: copy
    #   from: .env        # Copy actual .env file (gitignored)
    #   to: .env

    # Example: Run a command to show all worktrees
    - type: command
      command: wtp list

    # More examples (commented out):
    
    # Copy AI context files (typically gitignored):
    # - type: copy
    #   from: .claude     # Claude AI context
    #   to: .claude
    # - type: copy
    #   from: .cursor/    # Cursor IDE settings
    #   to: .cursor/

    # Share directories with symlinks:
    # - type: symlink
    #   from: .bin        # Shared tool cache
    #   to: .bin
    
    # Run setup commands:
    # - type: command
    #   command: npm install
    # - type: command
    #   command: echo "Created new worktree!"
`

	if err := ensureWritableDirectory(repo.Path()); err != nil {
		return errors.DirectoryAccessFailed("create configuration file", repo.Path(), err)
	}

	// Write configuration file with comments
	if err := writeFile(configPath, []byte(configContent), configFileMode); err != nil {
		return errors.DirectoryAccessFailed("create configuration file", configPath, err)
	}

	// Get the writer from cli.Command
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	if _, printErr := fmt.Fprintf(w, "Configuration file created: %s\n", configPath); printErr != nil {
		return printErr
	}
	_, printLnErr := fmt.Fprintln(w, "Edit this file to customize your worktree setup.")
	return printLnErr
}

func ensureWritableDirectory(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return fmt.Errorf("%s is not a directory", filepath.Base(path))
	}

	if info.Mode().Perm()&0o200 == 0 {
		return fmt.Errorf("write permission denied for directory: %s", path)
	}

	return nil
}
