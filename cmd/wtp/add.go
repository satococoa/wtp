package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/satococoa/wtp/internal/command"
	"github.com/satococoa/wtp/internal/config"
	"github.com/satococoa/wtp/internal/errors"
	"github.com/satococoa/wtp/internal/git"
	"github.com/satococoa/wtp/internal/hooks"
	wtpio "github.com/satococoa/wtp/internal/io"
)

// NewAddCommand creates the add command definition
func NewAddCommand() *cli.Command {
	return &cli.Command{
		Name:      "add",
		Usage:     "Create a new worktree",
		UsageText: "wtp add <existing-branch>\n       wtp add -b <new-branch> [<commit>]",
		Description: "Creates a new worktree for the specified branch. If the branch doesn't exist locally " +
			"but exists on a remote, it will be automatically tracked.\n\n" +
			"Examples:\n" +
			"  wtp add feature/auth                    # Create worktree from existing branch\n" +
			"  wtp add -b new-feature                  # Create new branch and worktree\n" +
			"  wtp add -b hotfix/urgent main           # Create new branch from main commit",
		ShellComplete: completeBranches,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "branch",
				Usage:   "Create new branch",
				Aliases: []string{"b"},
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
	resolvedTrack, err := resolveBranchTracking(cmd, branchName, mainRepoPath)
	if err != nil {
		return err
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
	displaySuccessMessage(w, branchName, workTreePath, cfg, mainRepoPath)

	// Execute post-create hooks
	if err := executePostCreateHooks(w, cfg, mainRepoPath, workTreePath); err != nil {
		// Log warning but don't fail the entire operation
		fmt.Fprintf(w, "Warning: Hook execution failed: %v\n", err)
	}

	return nil
}

// buildWorktreeCommand builds a git worktree command using the new command package
func buildWorktreeCommand(
	cmd *cli.Command, workTreePath, _, resolvedTrack string,
) command.Command {
	opts := command.GitWorktreeAddOptions{
		Branch: cmd.String("branch"),
	}

	// Use resolved track if provided
	if resolvedTrack != "" {
		opts.Track = resolvedTrack
	}

	var commitish string

	// Handle different argument patterns based on flags
	if resolvedTrack != "" {
		// When using resolved tracking, the commitish is the remote branch
		commitish = resolvedTrack
		// If there's an argument, it's the local branch name (not used as commitish)
		if cmd.Args().Len() > 0 && opts.Branch == "" {
			// The first argument is the branch name when using resolved tracking without -b
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
		return errors.BranchNameRequired("wtp add <existing-branch> | -b <new-branch> [<commit>]")
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

// getBranches gets available branch names and writes them to the writer (testable)
func getBranches(w io.Writer) error {
	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Get all branches using git for-each-ref for better control
	gitCmd := exec.Command("git", "for-each-ref", "--format=%(refname:short)", "refs/heads", "refs/remotes")
	gitCmd.Dir = cwd
	output, err := gitCmd.Output()
	if err != nil {
		return err
	}

	branches := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Use a map to avoid duplicates
	seen := make(map[string]bool)

	for _, branch := range branches {
		if branch == "" {
			continue
		}

		// Skip HEAD references and bare origin
		if branch == "origin/HEAD" || branch == "origin" {
			continue
		}

		// Remove remote prefix for display, but keep track of what we've seen
		displayName := branch
		if strings.HasPrefix(branch, "origin/") {
			// For remote branches, show without the origin/ prefix
			displayName = strings.TrimPrefix(branch, "origin/")
		}

		// Skip if already seen (handles case where local and remote have same name)
		if seen[displayName] {
			continue
		}

		seen[displayName] = true
		fmt.Fprintln(w, displayName)
	}

	return nil
}

// completeBranches provides branch name completion for urfave/cli (wrapper for getBranches)
func completeBranches(_ context.Context, cmd *cli.Command) {
	current, previous := completionArgsFromCommand(cmd)
	if maybeCompleteFlagSuggestions(cmd, current, previous) {
		return
	}

	var buf bytes.Buffer
	if err := getBranches(&buf); err != nil {
		return
	}

	// Output each line using fmt.Println for urfave/cli compatibility
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		fmt.Println(scanner.Text())
	}
}

// displaySuccessMessage is a convenience wrapper for displaySuccessMessageWithCommitish
func displaySuccessMessage(w io.Writer, branchName, workTreePath string, cfg *config.Config, mainRepoPath string) {
	displaySuccessMessageWithCommitish(w, branchName, workTreePath, "", cfg, mainRepoPath)
}

func displaySuccessMessageWithCommitish(
	w io.Writer, branchName, workTreePath, commitish string, cfg *config.Config, mainRepoPath string,
) {
	fmt.Fprintln(w, "✅ Worktree created successfully!")
	fmt.Fprintln(w)
	fmt.Fprintf(w, "📁 Location: %s\n", workTreePath)

	if branchName != "" {
		fmt.Fprintf(w, "🌿 Branch: %s\n", branchName)
	} else if commitish != "" {
		fmt.Fprintf(w, "🏷️  Commit: %s\n", commitish)
	}

	fmt.Fprintln(w)
	fmt.Fprintln(w, "💡 To switch to the new worktree, run:")

	// Use the consistent worktree naming logic
	isMain := isMainWorktree(workTreePath, mainRepoPath)
	worktreeName := getWorktreeNameFromPath(workTreePath, cfg, mainRepoPath, isMain)
	fmt.Fprintf(w, "   wtp cd %s\n", worktreeName)
}

// isMainWorktree checks if the given path is the main worktree
func isMainWorktree(workTreePath, mainRepoPath string) bool {
	absWorkTreePath, err := filepath.Abs(workTreePath)
	if err != nil {
		return false
	}

	absMainRepoPath, err := filepath.Abs(mainRepoPath)
	if err != nil {
		return false
	}

	return absWorkTreePath == absMainRepoPath
}

// resolveWorktreePath determines the worktree path and branch name based on arguments
func resolveWorktreePath(
	cfg *config.Config, repoPath, firstArg string, cmd *cli.Command,
) (workTreePath, branchName string) {
	// Generate path automatically from branch name
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

// resolveBranchTracking handles branch resolution and tracking setup
func resolveBranchTracking(
	cmd *cli.Command, branchName string, mainRepoPath string,
) (string, error) {
	// Only auto-resolve branch when not creating a new branch and branch name exists
	if cmd.String("branch") != "" || branchName == "" {
		return "", nil
	}

	repo, err := git.NewRepository(mainRepoPath)
	if err != nil {
		return "", err
	}

	// Check if branch exists locally or in remotes
	resolvedBranch, isRemote, err := repo.ResolveBranch(branchName)
	if err != nil {
		// Check if it's a multiple branches error
		if strings.Contains(err.Error(), "exists in multiple remotes") {
			return "", &MultipleBranchesError{
				BranchName: branchName,
				GitError:   err,
			}
		}
		return "", err
	}

	// If it's a remote branch, we need to set up tracking
	if isRemote {
		return resolvedBranch, nil
	}

	return "", nil
}
