// Package main provides the entrypoint for the wtp CLI commands.
package main

import (
	"bufio"
	"bytes"
	"context"
	stderrors "errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/satococoa/wtp/v2/internal/command"
	"github.com/satococoa/wtp/v2/internal/config"
	"github.com/satococoa/wtp/v2/internal/errors"
	"github.com/satococoa/wtp/v2/internal/git"
	"github.com/satococoa/wtp/v2/internal/hooks"
	wtpio "github.com/satococoa/wtp/v2/internal/io"
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
			"  wtp add -b hotfix/urgent main           # Create new branch from main commit\n" +
			"  wtp add -b feature/x --exec \"npm test\" # Execute command in the new worktree",
		ShellComplete: completeBranches,
		Flags: []cli.Flag{
			&cli.StringFlag{
				Name:    "branch",
				Usage:   "Create new branch",
				Aliases: []string{"b"},
			},
			&cli.StringFlag{
				Name:  "exec",
				Usage: "Execute command in newly created worktree after hooks",
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
		cmdLine := commandLineString(worktreeCmd)
		gitError = formatWorktreeAddError(gitError, cmdLine)

		// Analyze git error output for better error messages
		creatingNewBranch := cmd.String("branch") != ""
		return analyzeGitWorktreeError(workTreePath, branchName, gitError, gitOutput, mainRepoPath, creatingNewBranch)
	}

	if err := executePostCreateHooks(w, cfg, mainRepoPath, workTreePath); err != nil {
		if _, warnErr := fmt.Fprintf(w, "Warning: Hook execution failed: %v\n", err); warnErr != nil {
			return warnErr
		}
	}

	if err := executePostCreateCommand(w, cmdExec, cmd.String("exec"), workTreePath); err != nil {
		return fmt.Errorf("worktree was created at '%s', but --exec command failed: %w", workTreePath, err)
	}

	if err := displaySuccessMessage(w, branchName, workTreePath, cfg, mainRepoPath); err != nil {
		return err
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

// CompositeWorktreeError holds multiple worktree-related errors to report all applicable causes.
type CompositeWorktreeError struct {
	Errors []error
}

func (e *CompositeWorktreeError) Error() string {
	var blocks []string
	var solutions []string
	var technical error
	for _, err := range e.Errors {
		var branchErr *BranchAlreadyExistsError
		var pathErr *PathAlreadyExistsError
		switch {
		case stderrors.As(err, &branchErr):
			blocks = append(blocks, fmt.Sprintf("branch '%s' already exists in this repository.", branchErr.BranchName))
			if technical == nil && branchErr.GitError != nil {
				technical = branchErr.GitError
			}
			solutions = appendSolution(solutions,
				fmt.Sprintf("Run 'wtp add %s' to create a worktree for the existing branch", branchErr.BranchName))
			solutions = appendSolution(solutions, "Choose a different branch name with '--branch'")
			solutions = appendSolution(solutions, "Delete the existing branch if it's no longer needed")
		case stderrors.As(err, &pathErr):
			blocks = append(blocks, fmt.Sprintf("destination path already exists: %s", pathErr.Path))
			if technical == nil && pathErr.GitError != nil {
				technical = pathErr.GitError
			}
			solutions = appendSolution(solutions, "Use --force flag to overwrite existing directory")
			solutions = appendSolution(solutions, "Remove the existing directory")
			solutions = appendSolution(solutions, "Use a different branch name")
		default:
			blocks = append(blocks, err.Error())
		}
	}
	var b strings.Builder
	for i, block := range blocks {
		if i > 0 {
			b.WriteString("\n\n")
		}
		b.WriteString(block)
	}
	if len(solutions) > 0 {
		b.WriteString("\n\nSolutions:\n")
		for _, s := range solutions {
			b.WriteString("  • ")
			b.WriteString(s)
			b.WriteString("\n")
		}
	}
	if technical != nil {
		b.WriteString("\n\n")
		b.WriteString(fmt.Sprint(technical))
	}
	return b.String()
}

func appendSolution(list []string, s string) []string {
	for _, existing := range list {
		if existing == s {
			return list
		}
	}
	return append(list, s)
}

// commandLineString returns the full command line for display (e.g. "git worktree add -b test /path").
func commandLineString(c command.Command) string {
	if len(c.Args) == 0 {
		return c.Name
	}
	return c.Name + " " + strings.Join(c.Args, " ")
}

// formatWorktreeAddError returns an error whose message is "Error command: <full command>\nExit status: <code>".
func formatWorktreeAddError(err error, cmdLine string) error {
	if err == nil {
		return nil
	}
	if cmdLine == "" {
		cmdLine = "git worktree add"
	}
	var exitErr *exec.ExitError
	if stderrors.As(err, &exitErr) {
		return fmt.Errorf("Error command: %s\nExit status: %d", cmdLine, exitErr.ExitCode())
	}
	return fmt.Errorf("Error command: %s\n%w", cmdLine, err)
}

// analyzeGitWorktreeError analyzes git worktree errors and provides specific error messages.
// When git reports an "already exists" style error, it checks all overlapping conditions
// (path, branch, worktree, multiple remotes) and returns all that apply (single or composite).
func analyzeGitWorktreeError(
	workTreePath, branchName string, gitError error, gitOutput, mainRepoPath string, creatingNewBranch bool,
) error {
	errorOutput := strings.ToLower(gitOutput)

	if isBranchNotFoundError(errorOutput) {
		return errors.BranchNotFound(branchName)
	}

	// Collect all applicable "already exists" style errors instead of returning the first match
	if isWorktreeAlreadyExistsError(errorOutput) ||
		isBranchAlreadyExistsError(errorOutput) ||
		isPathAlreadyExistsError(errorOutput) ||
		isMultipleBranchesError(errorOutput) {
		collected := collectAlreadyExistsErrors(
			workTreePath, branchName, gitError, errorOutput, mainRepoPath, creatingNewBranch)
		if len(collected) == 1 {
			return collected[0]
		}
		return &CompositeWorktreeError{Errors: collected}
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

%w`, workTreePath, gitOutput, gitError)
	}

	// Default error with helpful context
	return fmt.Errorf(`worktree creation failed for path '%s'

The git command encountered an error while creating the worktree.

Details: %s

Tip: Run 'git worktree list' to see existing worktrees, or check git documentation for valid worktree paths.

%w`, workTreePath, gitOutput, gitError)
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

func isBranchAlreadyExistsError(errorOutput string) bool {
	return strings.Contains(errorOutput, "branch") &&
		strings.Contains(errorOutput, "already exists")
}

func isPathAlreadyExistsError(errorOutput string) bool {
	// Path/destination errors, not branch-named errors (avoid matching "branch X already exists")
	if !strings.Contains(errorOutput, "already exists") {
		return false
	}
	if strings.Contains(errorOutput, "destination path") {
		return true
	}
	// Path-only message e.g. "fatal: '/path' already exists" (no "branch" in it)
	return !strings.Contains(errorOutput, "branch")
}

func isMultipleBranchesError(errorOutput string) bool {
	return strings.Contains(errorOutput, "more than one remote") || strings.Contains(errorOutput, "ambiguous")
}

// collectAlreadyExistsErrors gathers all applicable "already exists" errors from git output and repo state.
func collectAlreadyExistsErrors(
	workTreePath, branchName string, gitError error, errorOutput, mainRepoPath string, creatingNewBranch bool,
) []error {
	collected, hasBranch, hasPath := collectErrorsFromGitOutput(workTreePath, branchName, gitError, errorOutput)
	collected = appendRepoStateErrors(
		collected, workTreePath, branchName, gitError, mainRepoPath, creatingNewBranch, hasBranch, hasPath)
	return collected
}

func collectErrorsFromGitOutput(
	workTreePath, branchName string, gitError error, errorOutput string,
) ([]error, bool, bool) {
	var collected []error
	hasBranchAlreadyExists := false
	hasPathAlreadyExists := false
	if isWorktreeAlreadyExistsError(errorOutput) {
		collected = append(collected, &WorktreeAlreadyExistsError{
			BranchName: branchName, Path: workTreePath, GitError: gitError,
		})
	}
	if isBranchAlreadyExistsError(errorOutput) {
		hasBranchAlreadyExists = true
		collected = append(collected, &BranchAlreadyExistsError{BranchName: branchName, GitError: gitError})
	}
	if isPathAlreadyExistsError(errorOutput) {
		hasPathAlreadyExists = true
		collected = append(collected, &PathAlreadyExistsError{Path: workTreePath, GitError: gitError})
	}
	if isMultipleBranchesError(errorOutput) {
		collected = append(collected, &MultipleBranchesError{BranchName: branchName, GitError: gitError})
	}
	return collected, hasBranchAlreadyExists, hasPathAlreadyExists
}

func appendRepoStateErrors(
	collected []error, workTreePath, branchName string, gitError error, mainRepoPath string,
	creatingNewBranch, hasBranchAlreadyExists, hasPathAlreadyExists bool,
) []error {
	if mainRepoPath != "" && creatingNewBranch && branchName != "" && !hasBranchAlreadyExists {
		if repo, err := git.NewRepository(mainRepoPath); err == nil {
			if exists, err := repo.BranchExists(branchName); err == nil && exists {
				collected = append(collected, &BranchAlreadyExistsError{BranchName: branchName, GitError: gitError})
			}
		}
	}
	if workTreePath != "" && !hasPathAlreadyExists {
		if fi, err := os.Stat(workTreePath); err == nil && fi.IsDir() {
			collected = append(collected, &PathAlreadyExistsError{Path: workTreePath, GitError: gitError})
		}
	}
	return collected
}

func isInvalidPathError(errorOutput, workTreePath, gitOutput string) bool {
	return strings.Contains(errorOutput, "could not create directory") ||
		strings.Contains(errorOutput, "unable to create") ||
		strings.Contains(errorOutput, "is not a directory") ||
		strings.Contains(workTreePath, "/dev/") ||
		gitOutput == ""
}

func formatGitError(err error) string {
	if err == nil {
		return ""
	}
	return "\n\n" + err.Error()
}

// WorktreeAlreadyExistsError reports that a branch already has an attached worktree.
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

%s`, e.BranchName, e.BranchName, formatGitError(e.GitError))
}

// BranchAlreadyExistsError indicates that a branch creation request conflicts with an existing branch.
type BranchAlreadyExistsError struct {
	BranchName string
	GitError   error
}

func (e *BranchAlreadyExistsError) Error() string {
	return fmt.Sprintf(`branch '%s' already exists in this repository.

Solutions:
  • Run 'wtp add %s' to create a worktree for the existing branch
  • Choose a different branch name with '--branch'
  • Delete the existing branch if it's no longer needed

%s`, e.BranchName, e.BranchName, formatGitError(e.GitError))
}

// PathAlreadyExistsError indicates that the destination directory already exists.
type PathAlreadyExistsError struct {
	Path     string
	GitError error
}

func (e *PathAlreadyExistsError) Error() string {
	return fmt.Sprintf(`destination path already exists: %s

Solutions:
  • Use --force flag to overwrite existing directory
  • Remove the existing directory
  • Use a different branch name

%s`, e.Path, formatGitError(e.GitError))
}

// MultipleBranchesError reports that a branch name resolves to multiple remotes and needs disambiguation.
type MultipleBranchesError struct {
	BranchName string
	GitError   error
}

func (e *MultipleBranchesError) Error() string {
	return fmt.Sprintf(`branch '%s' exists in multiple remotes

Use the --track flag to specify which remote to use:
  • wtp add --track origin/%s %s
  • wtp add --track upstream/%s %s

%s`, e.BranchName, e.BranchName, e.BranchName, e.BranchName, e.BranchName, formatGitError(e.GitError))
}

func executePostCreateHooks(w io.Writer, cfg *config.Config, repoPath, workTreePath string) error {
	if cfg.HasHooks() {
		if _, err := fmt.Fprintln(w, "\nExecuting post-create hooks..."); err != nil {
			return err
		}

		executor := hooks.NewExecutor(cfg, repoPath)
		if err := executor.ExecutePostCreateHooks(w, workTreePath); err != nil {
			return err
		}

		if _, err := fmt.Fprintln(w, "✓ All hooks executed successfully"); err != nil {
			return err
		}
	}
	return nil
}

func executePostCreateCommand(w io.Writer, cmdExec command.Executor, execCommand, workTreePath string) error {
	if strings.TrimSpace(execCommand) == "" {
		return nil
	}

	if _, err := fmt.Fprintf(w, "\nExecuting --exec command: %s\n", execCommand); err != nil {
		return err
	}

	commandToRun := command.Command{
		WorkDir:     workTreePath,
		Interactive: true,
	}
	if runtime.GOOS == "windows" {
		commandToRun.Name = "cmd"
		commandToRun.Args = []string{"/c", execCommand}
	} else {
		commandToRun.Name = "sh"
		commandToRun.Args = []string{"-c", execCommand}
	}

	result, err := cmdExec.Execute([]command.Command{commandToRun})
	if err != nil {
		return err
	}

	if len(result.Results) == 0 {
		return fmt.Errorf("empty execution result")
	}

	commandResult := result.Results[0]
	if commandResult.Output != "" {
		if _, writeErr := fmt.Fprintln(w, commandResult.Output); writeErr != nil {
			return writeErr
		}
	}
	if commandResult.Error != nil {
		return commandResult.Error
	}

	if _, err := fmt.Fprintln(w, "✓ --exec command completed"); err != nil {
		return err
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
		if _, err := fmt.Fprintln(w, displayName); err != nil {
			return err
		}
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
		if _, err := fmt.Println(scanner.Text()); err != nil {
			return
		}
	}
}

// displaySuccessMessage is a convenience wrapper for displaySuccessMessageWithCommitish
func displaySuccessMessage(
	w io.Writer,
	branchName, workTreePath string,
	cfg *config.Config,
	mainRepoPath string,
) error {
	return displaySuccessMessageWithCommitish(w, branchName, workTreePath, "", cfg, mainRepoPath)
}

func displaySuccessMessageWithCommitish(
	w io.Writer, branchName, workTreePath, commitish string, cfg *config.Config, mainRepoPath string,
) error {
	if _, err := fmt.Fprintln(w, "✅ Worktree created successfully!"); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintf(w, "📁 Location: %s\n", workTreePath); err != nil {
		return err
	}

	if branchName != "" {
		if _, err := fmt.Fprintf(w, "🌿 Branch: %s\n", branchName); err != nil {
			return err
		}
	} else if commitish != "" {
		if _, err := fmt.Fprintf(w, "🏷️  Commit: %s\n", commitish); err != nil {
			return err
		}
	}

	if _, err := fmt.Fprintln(w); err != nil {
		return err
	}
	if _, err := fmt.Fprintln(w, "💡 To switch to the new worktree, run:"); err != nil {
		return err
	}

	// Use the consistent worktree naming logic
	isMain := isMainWorktree(workTreePath, mainRepoPath)
	worktreeName := getWorktreeNameFromPath(workTreePath, cfg, mainRepoPath, isMain)
	if _, err := fmt.Fprintf(w, "   wtp cd %s\n", worktreeName); err != nil {
		return err
	}

	return nil
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
