package main

import (
	"context"
	"fmt"
	"io"
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
			"but exists on a remote, it will be automatically tracked.\n\n" +
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
			&cli.BoolFlag{
				Name:  "cd",
				Usage: "Change directory to the new worktree after creation",
			},
			&cli.BoolFlag{
				Name:  "no-cd",
				Usage: "Do not change directory to the new worktree after creation",
			},
		},
		Action: addCommand,
	}
}

func addCommand(_ context.Context, cmd *cli.Command) error {
	// Get the writer from cli.Command
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}
	// Validate inputs
	if err := validateAddInput(cmd); err != nil {
		return err
	}

	// Setup repository and configuration
	repo, cfg, mainRepoPath, err := setupRepoAndConfig()
	if err != nil {
		return err
	}

	// Create git executor
	gitExec := newRepositoryExecutor(repo)

	return addCommandWithExecutor(cmd, w, gitExec, cfg, mainRepoPath)
}

func addCommandWithExecutor(cmd *cli.Command, w io.Writer, gitExec GitExecutor, cfg *config.Config, mainRepoPath string) error {
	// Resolve worktree path and branch name
	var firstArg string
	if cmd.Args().Len() > 0 {
		firstArg = cmd.Args().Get(0)
	}
	workTreePath, branchName := resolveWorktreePath(cfg, mainRepoPath, firstArg, cmd)

	// Handle branch resolution if needed
	if err := handleBranchResolutionWithExecutor(cmd, gitExec, branchName); err != nil {
		return err
	}

	// Build and execute git worktree command
	args := buildGitWorktreeArgs(cmd, workTreePath, branchName)
	if err := gitExec.ExecuteGitCommand(args...); err != nil {
		return errors.WorktreeCreationFailed(workTreePath, branchName, err)
	}

	// Display success message
	displaySuccessMessage(w, branchName, workTreePath)

	// Execute post-create hooks
	if err := executePostCreateHooks(w, cfg, mainRepoPath, workTreePath); err != nil {
		return err
	}

	// Change directory if requested
	if shouldChangeDirectory(cmd, cfg) {
		fmt.Fprintln(w)
		changeToWorktree(w, workTreePath)
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

func handleBranchResolutionWithExecutor(cmd *cli.Command, gitExec GitExecutor, branchName string) error {
	if cmd.String("branch") == "" && cmd.String("track") == "" && branchName != "" && !cmd.Bool("detach") {
		resolvedBranch, isRemoteBranch, err := gitExec.ResolveBranch(branchName)
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

func displaySuccessMessage(w io.Writer, branchName, workTreePath string) {
	if branchName != "" {
		fmt.Fprintf(w, "Created worktree '%s' at %s\n", branchName, workTreePath)
	} else {
		fmt.Fprintf(w, "Created worktree at %s\n", workTreePath)
	}
}

func executePostCreateHooks(w io.Writer, cfg *config.Config, repoPath, workTreePath string) error {
	if cfg.HasHooks() {
		fmt.Fprintln(w, "\nExecuting post-create hooks...")
		executor := hooks.NewExecutor(cfg, repoPath)
		if err := executor.ExecutePostCreateHooks(workTreePath); err != nil {
			fmt.Fprintf(w, "Warning: Hook execution failed: %v\n", err)
		} else {
			fmt.Fprintln(w, "✓ All hooks executed successfully")
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

	// Add basic flags
	args = appendBasicFlags(args, cmd)

	// Handle branch and track flags
	track := cmd.String("track")
	branch := cmd.String("branch")
	args = appendBranchAndTrackFlags(args, branch, track, branchName, cmd.Bool("detach"))

	// Add worktree path
	args = append(args, workTreePath)

	// Handle positional arguments
	args = appendPositionalArgs(args, cmd, branch, track, branchName)

	return args
}

func appendBasicFlags(args []string, cmd *cli.Command) []string {
	if cmd.Bool("force") {
		args = append(args, "--force")
	}
	if cmd.Bool("detach") {
		args = append(args, "--detach")
	}
	return args
}

func appendBranchAndTrackFlags(args []string, branch, track, branchName string, isDetached bool) []string {
	if branch != "" {
		args = append(args, "-b", branch)
	}

	// If tracking a remote branch and no explicit -b flag, add -b automatically
	if track != "" && branch == "" && !isDetached {
		// When tracking remote branch, we need to create local branch with -b
		args = append(args, "--track", "-b", branchName)
	} else if track != "" {
		// Add track flag when specified
		args = append(args, "--track")
	}

	return args
}

func appendPositionalArgs(args []string, cmd *cli.Command, branch, track, branchName string) []string {
	if cmd.String("path") != "" {
		// Explicit path case
		return appendExplicitPathArgs(args, cmd, branch, track, branchName)
	}
	// Auto-generated path case
	return appendAutoPathArgs(args, cmd, branch, track, branchName)
}

func appendExplicitPathArgs(args []string, cmd *cli.Command, branch, track, branchName string) []string {
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
	return args
}

func appendAutoPathArgs(args []string, cmd *cli.Command, branch, track, branchName string) []string {
	if branch != "" {
		// Using -b flag: first arg (if any) is the commit-ish to branch from
		if cmd.Args().Len() > 0 {
			args = append(args, cmd.Args().Get(0))
		}
	} else if track != "" && branch == "" {
		// When tracking a remote branch (with or without --detach), need to specify remote branch as commit-ish
		args = append(args, track)
	} else if track == "" && !cmd.Bool("detach") {
		// No -b flag and no --track: first arg is branch name (unless detached)
		args = append(args, branchName)
	} else if cmd.Bool("detach") && track == "" {
		// Detached mode without tracking: first arg is the commit-ish
		if cmd.Args().Len() > 0 {
			args = append(args, cmd.Args().Get(0))
		}
	}
	// Add any additional arguments (for certain cases)
	if cmd.Args().Len() > 1 && branch == "" && track == "" {
		args = append(args, cmd.Args().Slice()[1:]...)
	}
	return args
}

func shouldChangeDirectory(cmd *cli.Command, cfg *config.Config) bool {
	// Check command-line flags first
	if cmd.Bool("cd") {
		return true
	}
	if cmd.Bool("no-cd") {
		return false
	}
	// Fall back to config setting
	return cfg.Defaults.CDAfterCreate
}

func changeToWorktree(w io.Writer, workTreePath string) {
	// Check if shell integration is enabled
	if os.Getenv("WTP_SHELL_INTEGRATION") != "1" {
		fmt.Fprintf(w, "To change directory, run: cd %s\n", workTreePath)
		fmt.Fprintln(w, "(Enable shell integration with: eval \"$(wtp completion zsh)\")")
		return
	}

	// Output the path for the shell function to use
	fmt.Fprint(w, workTreePath)
}
