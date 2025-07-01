package main

import (
	"context"
	"fmt"
	"os"
	"strings"

	"github.com/satococoa/wtp/internal/config"
	"github.com/satococoa/wtp/internal/git"
	"github.com/urfave/cli/v3"
)

// NewRemoveCommand creates the remove command definition
func NewRemoveCommand() *cli.Command {
	return &cli.Command{
		Name:      "remove",
		Usage:     "Remove a worktree",
		UsageText: "wtp remove <branch-name>",
		Description: "Removes the worktree associated with the specified branch.\n\n" +
			"Examples:\n" +
			"  wtp remove feature/old                  # Remove worktree\n" +
			"  wtp remove -f feature/dirty             # Force remove dirty worktree\n" +
			"  wtp remove --with-branch feature/done   # Also delete the branch",
		ShellComplete: completeWorktrees,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Usage:   "Force removal even if worktree is dirty",
				Aliases: []string{"f"},
			},
			&cli.BoolFlag{
				Name:  "with-branch",
				Usage: "Also remove the branch after removing worktree",
			},
			&cli.BoolFlag{
				Name:  "force-branch",
				Usage: "Force branch deletion even if not merged (requires --with-branch)",
			},
		},
		Action: removeCommand,
	}
}

func removeCommand(_ context.Context, cmd *cli.Command) error {
	branchName := cmd.Args().Get(0)
	force := cmd.Bool("force")
	withBranch := cmd.Bool("with-branch")
	forceBranch := cmd.Bool("force-branch")

	if branchName == "" {
		return fmt.Errorf("branch name is required")
	}

	// Validate flags
	if forceBranch && !withBranch {
		return fmt.Errorf("--force-branch requires --with-branch")
	}

	// Get current working directory (should be a git repository)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Initialize repository
	repo, err := git.NewRepository(cwd)
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Get main repository path for config loading
	mainRepoPath, err := repo.GetMainWorktreePath()
	if err != nil {
		// Fallback to current repository path if we can't determine main repo
		mainRepoPath = repo.Path()
	}

	// Load configuration from main repository
	cfg, err := config.LoadConfig(mainRepoPath)
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Resolve worktree path using configuration
	// Use branch name as worktree name
	workTreePath := cfg.ResolveWorktreePath(repo.Path(), branchName)

	// Remove worktree
	if err := repo.RemoveWorktree(workTreePath, force); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	fmt.Printf("Removed worktree '%s' at %s\n", branchName, workTreePath)

	// Remove branch if requested
	if withBranch {
		// Build git branch delete command
		args := []string{"branch"}
		if forceBranch {
			args = append(args, "-D")
		} else {
			args = append(args, "-d")
		}
		args = append(args, branchName)

		// Execute git branch delete
		if err := repo.ExecuteGitCommand(args...); err != nil {
			// Check if it's an unmerged branch error
			if !forceBranch && strings.Contains(err.Error(), "not fully merged") {
				return fmt.Errorf("branch '%s' is not fully merged. Use --force-branch to delete anyway", branchName)
			}
			return fmt.Errorf("failed to remove branch: %w", err)
		}

		fmt.Printf("Removed branch '%s'\n", branchName)
	}

	return nil
}
