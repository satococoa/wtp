package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/satococoa/wtp/internal/git"
	"github.com/urfave/cli/v3"
)

// NewCdCommand creates the cd command definition
func NewCdCommand() *cli.Command {
	return &cli.Command{
		Name:  "cd",
		Usage: "Change directory to worktree (requires shell integration)",
		Description: "Change the current working directory to the specified worktree. " +
			"This command requires shell integration to be set up first.\n\n" +
			"To enable shell integration, add the following to your shell config:\n" +
			"  Bash: eval \"$(wtp shell-init --cd)\"\n" +
			"  Zsh:  eval \"$(wtp shell-init --cd)\"\n" +
			"  Fish: wtp shell-init --cd | source",
		ArgsUsage: "<worktree-name>",
		Action:    cdToWorktree,
	}
}

func cdToWorktree(_ context.Context, cmd *cli.Command) error {
	// Check if we're running inside the shell function
	if os.Getenv("WTP_SHELL_INTEGRATION") != "1" {
		return fmt.Errorf("cd command requires shell integration. Run 'wtp shell-init --cd' for setup instructions")
	}

	args := cmd.Args()
	if args.Len() == 0 {
		return fmt.Errorf("worktree name required")
	}

	worktreeName := args.Get(0)

	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Initialize repository
	repo, err := git.NewRepository(cwd)
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Get all worktrees
	worktrees, err := repo.GetWorktrees()
	if err != nil {
		return fmt.Errorf("failed to get worktrees: %w", err)
	}

	// Find the worktree by name
	var targetPath string
	for _, wt := range worktrees {
		// Match by branch name or directory name
		if wt.Branch == worktreeName || filepath.Base(wt.Path) == worktreeName {
			targetPath = wt.Path
			break
		}
	}

	if targetPath == "" {
		return fmt.Errorf("worktree '%s' not found", worktreeName)
	}

	// Output the path for the shell function to cd to
	fmt.Println(targetPath)
	return nil
}
