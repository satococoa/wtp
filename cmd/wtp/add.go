package main

import (
	"context"
	"fmt"
	"os"

	"github.com/satococoa/wtp/internal/config"
	"github.com/satococoa/wtp/internal/errors"
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
	// Validate inputs
	if err := validateAddInput(cmd); err != nil {
		return err
	}

	// Setup repository and configuration
	repo, cfg, mainRepoPath, err := setupRepoAndConfig()
	if err != nil {
		return err
	}

	// Resolve worktree path and branch name
	var firstArg string
	if cmd.Args().Len() > 0 {
		firstArg = cmd.Args().Get(0)
	}
	workTreePath, branchName := resolveWorktreePath(cfg, mainRepoPath, firstArg, cmd)

	// Handle branch resolution if needed
	if err := handleBranchResolution(cmd, repo, branchName); err != nil {
		return err
	}

	// Build and execute git worktree command
	args := buildGitWorktreeArgs(cmd, workTreePath, branchName)
	if err := repo.ExecuteGitCommand(args...); err != nil {
		return errors.WorktreeCreationFailed(workTreePath, branchName, err)
	}

	// Display success message
	displaySuccessMessage(branchName, workTreePath)

	// Execute post-create hooks
	if err := executePostCreateHooks(cfg, mainRepoPath, workTreePath); err != nil {
		return err
	}

	return nil
}

func validateAddInput(cmd *cli.Command) error {
	if cmd.Args().Len() == 0 && cmd.String("branch") == "" {
		return errors.BranchNameRequired("wtp add <branch-name>")
	}
	return nil
}

func setupRepoAndConfig() (*git.Repository, *config.Config, string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return nil, nil, "", errors.DirectoryAccessFailed("access current", ".", err)
	}

	repo, err := git.NewRepository(cwd)
	if err != nil {
		return nil, nil, "", errors.NotInGitRepository()
	}

	mainRepoPath, err := repo.GetMainWorktreePath()
	if err != nil {
		mainRepoPath = repo.Path()
	}

	cfg, err := config.LoadConfig(mainRepoPath)
	if err != nil {
		configPath := mainRepoPath + "/.wtp.yml"
		return nil, nil, "", errors.ConfigLoadFailed(configPath, err)
	}

	return repo, cfg, mainRepoPath, nil
}

func handleBranchResolution(cmd *cli.Command, repo *git.Repository, branchName string) error {
	if cmd.String("branch") == "" && cmd.String("track") == "" && branchName != "" && !cmd.Bool("detach") {
		resolvedBranch, isRemoteBranch, err := repo.ResolveBranch(branchName)
		if err != nil {
			return err
		}
		if isRemoteBranch {
			if err := cmd.Set("track", resolvedBranch); err != nil {
				return fmt.Errorf("failed to set track flag: %w", err)
			}
		}
	}
	return nil
}

func displaySuccessMessage(branchName, workTreePath string) {
	if branchName != "" {
		fmt.Printf("Created worktree '%s' at %s\n", branchName, workTreePath)
	} else {
		fmt.Printf("Created worktree at %s\n", workTreePath)
	}
}

func executePostCreateHooks(cfg *config.Config, repoPath, workTreePath string) error {
	if cfg.HasHooks() {
		fmt.Println("\nExecuting post-create hooks...")
		executor := hooks.NewExecutor(cfg, repoPath)
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

	// Handle branch and track flags
	track := cmd.String("track")
	branch := cmd.String("branch")

	// Handle branch and track flags together
	if branch != "" {
		args = append(args, "-b", branch)
	}

	// If tracking a remote branch and no explicit -b flag, add -b automatically
	if track != "" && branch == "" && !cmd.Bool("detach") {
		// When tracking remote branch, we need to create local branch with -b
		args = append(args, "--track", "-b", branchName)
	} else if track != "" {
		// Add track flag when specified
		args = append(args, "--track")
	}

	// Add worktree path
	args = append(args, workTreePath)

	// Handle arguments based on whether explicit path was specified
	if cmd.String("path") != "" {
		// Explicit path case: first arg is branch name, add remaining args
		// Only add branch name if not using -b flag (to avoid duplication)
		if branch == "" && track == "" {
			args = append(args, branchName)
		} else if track != "" && branch == "" {
			// When tracking with -b, need to specify the remote branch
			args = append(args, track)
		}
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
		} else if track != "" && branch == "" && !cmd.Bool("detach") {
			// When auto-tracking remote branch, need to specify remote branch as commit-ish
			args = append(args, track)
		} else if track == "" {
			// No -b flag and no --track: first arg is branch name
			args = append(args, branchName)
		}
		// Add any additional arguments (for certain cases)
		if cmd.Args().Len() > 1 && branch == "" && track == "" {
			args = append(args, cmd.Args().Slice()[1:]...)
		}
	}

	return args
}
