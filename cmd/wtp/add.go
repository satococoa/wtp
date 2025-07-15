package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/satococoa/wtp/internal/command"
	"github.com/satococoa/wtp/internal/config"
	"github.com/satococoa/wtp/internal/errors"
	"github.com/satococoa/wtp/internal/git"
	"github.com/satococoa/wtp/internal/hooks"
	wtpio "github.com/satococoa/wtp/internal/io"
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
	// Wrap in FlushingWriter to ensure real-time output for all operations
	fw := wtpio.NewFlushingWriter(w)
	// Validate inputs
	if err := validateAddInput(cmd); err != nil {
		return err
	}

	// Setup repository and configuration
	_, cfg, mainRepoPath, err := setupRepoAndConfig()
	if err != nil {
		return err
	}

	// Create command executor
	executor := command.NewRealExecutor()

	return addCommandWithCommandExecutor(cmd, fw, executor, cfg, mainRepoPath)
}

// addCommandWithCommandExecutor is the new implementation using CommandExecutor
func addCommandWithCommandExecutor(
	cmd *cli.Command, w io.Writer, cmdExec command.Executor, cfg *config.Config, mainRepoPath string,
) error {
	// Resolve worktree path and branch name
	var firstArg string
	if cmd.Args().Len() > 0 {
		firstArg = cmd.Args().Get(0)
	}
	workTreePath, branchName := resolveWorktreePath(cfg, mainRepoPath, firstArg, cmd)

	// Resolve branch if needed
	var resolvedTrack string
	// Only auto-resolve branch when:
	// 1. Not creating a new branch (-b flag)
	// 2. Not explicitly tracking a remote (--track flag)
	// 3. Not in detached mode (--detach flag)
	// 4. Branch name is provided
	if cmd.String("branch") == "" && cmd.String("track") == "" && !cmd.Bool("detach") && branchName != "" {
		repo, err := git.NewRepository(mainRepoPath)
		if err != nil {
			return err
		}

		// Check if branch exists locally or in remotes
		resolvedBranch, isRemote, err := repo.ResolveBranch(branchName)
		if err != nil {
			// Check if it's a multiple branches error
			if strings.Contains(err.Error(), "exists in multiple remotes") {
				return &MultipleBranchesError{
					BranchName: branchName,
					GitError:   err,
				}
			}
			return err
		}

		// If it's a remote branch, we need to set up tracking
		if isRemote {
			resolvedTrack = resolvedBranch
		}
	}

	// Build git worktree command using the new command builder
	worktreeCmd := buildWorktreeCommand(cmd, workTreePath, branchName, resolvedTrack)

	// Execute the command
	result, err := cmdExec.Execute([]command.Command{worktreeCmd})
	if err != nil {
		return err
	}

	// Check if command succeeded
	if len(result.Results) > 0 && result.Results[0].Error != nil {
		gitError := result.Results[0].Error
		gitOutput := result.Results[0].Output

		// Analyze git error output for better error messages
		return analyzeGitWorktreeError(workTreePath, branchName, gitError, gitOutput)
	}

	// Display success message
	displaySuccessMessage(w, branchName, workTreePath)

	// Execute post-create hooks
	if err := executePostCreateHooks(w, cfg, mainRepoPath, workTreePath); err != nil {
		// Log warning but don't fail the entire operation
		fmt.Fprintf(w, "Warning: Hook execution failed: %v\n", err)
	}

	// Change directory if requested
	if shouldChangeDirectory(cmd, cfg) {
		fmt.Fprintln(w)
		changeToWorktree(w, workTreePath)
	}

	return nil
}

// buildWorktreeCommand builds a git worktree command using the new command package
func buildWorktreeCommand(cmd *cli.Command, workTreePath, _, resolvedTrack string) command.Command {
	opts := command.GitWorktreeAddOptions{
		Force:  cmd.Bool("force"),
		Detach: cmd.Bool("detach"),
		Branch: cmd.String("branch"),
		Track:  cmd.String("track"),
	}

	// Use resolved track if provided and no explicit track flag
	if resolvedTrack != "" && opts.Track == "" {
		opts.Track = resolvedTrack
	}

	var commitish string

	// Handle different argument patterns based on flags
	if opts.Track != "" {
		// When using --track, the commitish is the remote branch specified in --track
		commitish = opts.Track
		// If there's an argument, it's the local branch name (not used as commitish)
		if cmd.Args().Len() > 0 && opts.Branch == "" {
			// The first argument is the branch name when using --track without -b
			opts.Branch = cmd.Args().Get(0)
		}
	} else if cmd.Args().Len() > 0 {
		// Normal case: first argument is the branch/commitish
		commitish = cmd.Args().Get(0)
		// If branch creation with -b, second arg (if any) is the commitish
		if opts.Branch != "" && cmd.Args().Len() > 1 {
			commitish = cmd.Args().Get(1)
		}
	}

	return command.GitWorktreeAdd(workTreePath, commitish, opts)
}

// analyzeGitWorktreeError analyzes git worktree errors and provides specific error messages
func analyzeGitWorktreeError(workTreePath, branchName string, gitError error, gitOutput string) error {
	errorOutput := strings.ToLower(gitOutput)

	// Check for specific error types
	if isBranchNotFoundError(errorOutput) {
		return errors.BranchNotFound(branchName)
	}

	if isWorktreeAlreadyExistsError(errorOutput) {
		return &WorktreeAlreadyExistsError{
			BranchName: branchName,
			Path:       workTreePath,
			GitError:   gitError,
		}
	}

	if isPathAlreadyExistsError(errorOutput) {
		return &PathAlreadyExistsError{
			Path:     workTreePath,
			GitError: gitError,
		}
	}

	if isMultipleBranchesError(errorOutput) {
		return &MultipleBranchesError{
			BranchName: branchName,
			GitError:   gitError,
		}
	}

	if isInvalidPathError(errorOutput, workTreePath, gitOutput) {
		return fmt.Errorf(`failed to create worktree at '%s'

The git command failed to create the worktree directory.

Possible causes:
  • Invalid path specified
  • Parent directory doesn't exist
  • Insufficient permissions
  • Path points to a file instead of directory

Details: %s

Tip: Check that the parent directory exists and you have write permissions.

Original error: %v`, workTreePath, gitOutput, gitError)
	}

	// Default error with helpful context
	return fmt.Errorf(`worktree creation failed for path '%s'

The git command encountered an error while creating the worktree.

Details: %s

Tip: Run 'git worktree list' to see existing worktrees, or check git documentation for valid worktree paths.

Original error: %v`, workTreePath, gitOutput, gitError)
}

// Helper functions to reduce cyclomatic complexity
func isBranchNotFoundError(errorOutput string) bool {
	return strings.Contains(errorOutput, "invalid reference") ||
		strings.Contains(errorOutput, "not a valid object name") ||
		(strings.Contains(errorOutput, "pathspec") && strings.Contains(errorOutput, "did not match"))
}

func isWorktreeAlreadyExistsError(errorOutput string) bool {
	return strings.Contains(errorOutput, "already checked out") ||
		strings.Contains(errorOutput, "already used by worktree")
}

func isPathAlreadyExistsError(errorOutput string) bool {
	return strings.Contains(errorOutput, "already exists")
}

func isMultipleBranchesError(errorOutput string) bool {
	return strings.Contains(errorOutput, "more than one remote") || strings.Contains(errorOutput, "ambiguous")
}

func isInvalidPathError(errorOutput, workTreePath, gitOutput string) bool {
	return strings.Contains(errorOutput, "could not create directory") ||
		strings.Contains(errorOutput, "unable to create") ||
		strings.Contains(errorOutput, "is not a directory") ||
		strings.Contains(errorOutput, "fatal:") ||
		strings.Contains(workTreePath, "/dev/") ||
		gitOutput == ""
}

// Custom error types for specific worktree errors
type WorktreeAlreadyExistsError struct {
	BranchName string
	Path       string
	GitError   error
}

func (e *WorktreeAlreadyExistsError) Error() string {
	return fmt.Sprintf(`worktree for branch '%s' already exists

The branch '%s' is already checked out in another worktree.

Solutions:
  • Use '--force' flag to allow multiple checkouts
  • Choose a different branch
  • Remove the existing worktree first

Original error: %v`, e.BranchName, e.BranchName, e.GitError)
}

type PathAlreadyExistsError struct {
	Path     string
	GitError error
}

func (e *PathAlreadyExistsError) Error() string {
	return fmt.Sprintf(`destination path already exists: %s

The target directory already exists and is not empty.

Solutions:
  • Use --force flag to overwrite existing directory
  • Choose a different path with --path flag
  • Remove the existing directory
  • Use a different branch name

Original error: %v`, e.Path, e.GitError)
}

type MultipleBranchesError struct {
	BranchName string
	GitError   error
}

func (e *MultipleBranchesError) Error() string {
	return fmt.Sprintf(`branch '%s' exists in multiple remotes

Use the --track flag to specify which remote to use:
  • wtp add --track origin/%s %s
  • wtp add --track upstream/%s %s

Original error: %v`, e.BranchName, e.BranchName, e.BranchName, e.BranchName, e.BranchName, e.GitError)
}

func executePostCreateHooks(w io.Writer, cfg *config.Config, repoPath, workTreePath string) error {
	if cfg.HasHooks() {
		fmt.Fprintln(w, "\nExecuting post-create hooks...")

		executor := hooks.NewExecutor(cfg, repoPath)
		if err := executor.ExecutePostCreateHooks(w, workTreePath); err != nil {
			return err
		}

		fmt.Fprintln(w, "✓ All hooks executed successfully")
	}
	return nil
}

func validateAddInput(cmd *cli.Command) error {
	if cmd.Args().Len() == 0 && cmd.String("branch") == "" {
		return errors.BranchNameRequired("wtp add <branch-name>")
	}

	// Check for conflicting flags
	if cmd.String("branch") != "" && cmd.Bool("detach") {
		return fmt.Errorf(`conflicting flags: cannot use both -b/--branch and --detach

The -b/--branch flag creates a new branch, while --detach creates a detached HEAD.
These options are incompatible.

Choose one:
  • Use -b to create and checkout a new branch
  • Use --detach to checkout in detached HEAD state`)
	}

	// Check for --track with --detach without -b
	if cmd.String("track") != "" && cmd.Bool("detach") && cmd.String("branch") == "" {
		return fmt.Errorf(`--track can only be used if a new branch is created

The --track flag sets up tracking for a new branch, but --detach creates a detached HEAD.
To use --track, you must also use -b to create a new branch.

Examples:
  • wtp add --track origin/main -b my-branch
  • wtp add --detach origin/main (without tracking)`)
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

func displaySuccessMessage(w io.Writer, branchName, workTreePath string) {
	if branchName != "" {
		fmt.Fprintf(w, "Created worktree '%s' at %s\n", branchName, workTreePath)
	} else {
		fmt.Fprintf(w, "Created worktree at %s\n", workTreePath)
	}
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
