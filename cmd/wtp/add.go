package main

import (
	"context"
	"fmt"
	"os"

	"github.com/satococoa/wtp/internal/config"
	"github.com/satococoa/wtp/internal/git"
	"github.com/satococoa/wtp/internal/hooks"
	"github.com/urfave/cli/v3"
)

// NewAddCommand creates the add command definition
func NewAddCommand() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "Create a new worktree",
		UsageText: "wtp add [--path <path>] [git-worktree-options...] <branch-name> [<commit-ish>]",
		Description: "Creates a new worktree for the specified branch. If the branch doesn't exist locally " +
			"but exists on a remote, it will be automatically tracked. Supports all git worktree flags.\n\n" +
			"Examples:\n" +
			"  wtp add feature/auth                    # Auto-generate path: ../worktrees/feature/auth\n" +
			"  wtp add --path /tmp/test feature/auth   # Use explicit path\n" +
			"  wtp add -b new-feature main             # Create new branch from main\n" +
			"  wtp add --detach abc1234                # Detached HEAD at commit",
		ShellComplete: completeBranches,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:  "path",
				Usage: "Specify explicit path for worktree (instead of auto-generation)",
			},
			&cli.BoolFlag{
				Name:    "force",
				Usage:   "Checkout <commit-ish> even if already checked out in other worktree",
				Aliases: []string{"f"},
			},
			&cli.BoolFlag{
				Name:  "detach",
				Usage: "Make the new worktree's HEAD detached",
			},
			&cli.BoolFlag{
				Name:  "checkout",
				Usage: "Populate the new worktree (default)",
			},
			&cli.BoolFlag{
				Name:  "lock",
				Usage: "Keep the new worktree locked",
			},
			&cli.StringFlag{
				Name:  "reason",
				Usage: "Reason for locking",
			},
			&cli.BoolFlag{
				Name:  "orphan",
				Usage: "Create orphan branch in new worktree",
			},
			&cli.StringFlag{
				Name:    "branch",
				Usage:   "Create new branch",
				Aliases: []string{"b"},
			},
			&cli.StringFlag{
				Name:    "track",
				Usage:   "Set upstream branch",
				Aliases: []string{"t"},
			},
		},
		Action: addCommand,
	}
}

func addCommand(_ context.Context, cmd *cli.Command) error {
	// Check if we have a branch name from either args or -b flag
	if cmd.Args().Len() == 0 && cmd.String("branch") == "" {
		return fmt.Errorf("branch name is required")
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

	// Resolve worktree path and branch name
	var firstArg string
	if cmd.Args().Len() > 0 {
		firstArg = cmd.Args().Get(0)
	}
	workTreePath, branchName := resolveWorktreePath(cfg, repo.Path(), firstArg, cmd)

	// Build git worktree add command
	args := buildGitWorktreeArgs(cmd, workTreePath, branchName)

	// Execute git worktree add
	if err := repo.ExecuteGitCommand(args...); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Display appropriate success message
	if branchName != "" {
		fmt.Printf("Created worktree '%s' at %s\n", branchName, workTreePath)
	} else {
		fmt.Printf("Created worktree at %s\n", workTreePath)
	}

	// Execute post-create hooks
	if cfg.HasHooks() {
		fmt.Println("\nExecuting post-create hooks...")
		executor := hooks.NewExecutor(cfg, repo.Path())
		if err := executor.ExecutePostCreateHooks(workTreePath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Hook execution failed: %v\n", err)
		} else {
			fmt.Println("✓ All hooks executed successfully")
		}
	}

	return nil
}

// resolveWorktreePath determines the worktree path and branch name based on flags and arguments
func resolveWorktreePath(
	cfg *config.Config, repoPath, firstArg string, cmd *cli.Command,
) (workTreePath, branchName string) {
	// Check if explicit path is specified via --path flag
	if explicitPath := cmd.String("path"); explicitPath != "" {
		// Explicit path specified - use it as-is, branch name from first argument
		return explicitPath, firstArg
	}

	// No explicit path - generate path automatically from branch name
	branchName = firstArg

	// If -b flag is provided, use that as the branch name for path generation
	if newBranch := cmd.String("branch"); newBranch != "" {
		branchName = newBranch
	}

	// If still no branch name, try to use the first argument
	if branchName == "" && firstArg != "" {
		branchName = firstArg
	}

	workTreePath = cfg.ResolveWorktreePath(repoPath, branchName)
	return workTreePath, branchName
}

func buildGitWorktreeArgs(cmd *cli.Command, workTreePath, branchName string) []string {
	args := []string{"worktree", "add"}

	// Add flags (excluding our custom --path flag)
	if cmd.Bool("force") {
		args = append(args, "--force")
	}
	if cmd.Bool("detach") {
		args = append(args, "--detach")
	}
	if cmd.Bool("checkout") {
		args = append(args, "--checkout")
	}
	if cmd.Bool("lock") {
		args = append(args, "--lock")
	}
	if reason := cmd.String("reason"); reason != "" {
		args = append(args, "--reason", reason)
	}
	if cmd.Bool("orphan") {
		args = append(args, "--orphan")
	}
	if branch := cmd.String("branch"); branch != "" {
		args = append(args, "-b", branch)
	}
	if track := cmd.String("track"); track != "" {
		args = append(args, "--track", track)
	}

	// Add worktree path
	args = append(args, workTreePath)

	// Handle arguments based on whether explicit path was specified
	if cmd.String("path") != "" {
		// Explicit path case: first arg is branch name, add remaining args
		args = append(args, branchName)
		if cmd.Args().Len() > 1 {
			args = append(args, cmd.Args().Slice()[1:]...)
		}
	} else {
		// Auto-generated path case
		if cmd.String("branch") != "" {
			// Using -b flag: first arg (if any) is the commit-ish to branch from
			if cmd.Args().Len() > 0 {
				args = append(args, cmd.Args().Get(0))
			}
		} else {
			// No -b flag: first arg is branch name
			args = append(args, branchName)
		}
		// Add any additional arguments (for both cases)
		if cmd.Args().Len() > 1 {
			args = append(args, cmd.Args().Slice()[1:]...)
		}
	}

	return args
}
