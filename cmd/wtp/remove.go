package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/satococoa/wtp/internal/command"
	"github.com/satococoa/wtp/internal/config"
	"github.com/satococoa/wtp/internal/errors"
	"github.com/satococoa/wtp/internal/git"
)

// Variable to allow mocking in tests
var removeGetwd = os.Getwd

// isWorktreeManaged determines if a worktree is managed by wtp
func isWorktreeManaged(worktreePath string, cfg *config.Config, mainRepoPath string, isMain bool) bool {
	// Main worktree is always managed
	if isMain {
		return true
	}

	// Get base directory - use default config if config is not available
	if cfg == nil {
		// Create default config when none is available
		defaultCfg := &config.Config{
			Defaults: config.Defaults{
				BaseDir: "../worktrees",
			},
		}
		cfg = defaultCfg
	}

	baseDir := cfg.ResolveWorktreePath(mainRepoPath, "")
	// Remove trailing slash if it exists
	baseDir = strings.TrimSuffix(baseDir, "/")

	// Check if worktree path is under base directory
	absWorktreePath, err := filepath.Abs(worktreePath)
	if err != nil {
		return false
	}

	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return false
	}

	// Check if worktree is within base directory
	relPath, err := filepath.Rel(absBaseDir, absWorktreePath)
	if err != nil {
		return false
	}

	// If relative path starts with "..", it's outside base directory
	return !strings.HasPrefix(relPath, "..")
}

// NewRemoveCommand creates the remove command definition
func NewRemoveCommand() *cli.Command {
	return &cli.Command{
		Name:      "remove",
		Usage:     "Remove a worktree",
		UsageText: "wtp remove <worktree-name>",
		Description: "Removes the worktree with the specified directory name.\n\n" +
			"Examples:\n" +
			"  wtp remove feature-old                  # Remove worktree\n" +
			"  wtp remove -f feature-dirty             # Force remove dirty worktree\n" +
			"  wtp remove --with-branch feature-done   # Also delete the associated branch",
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
	// Get the writer from cli.Command
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	// Extract and validate inputs
	worktreeName := cmd.Args().Get(0)
	force := cmd.Bool("force")
	withBranch := cmd.Bool("with-branch")
	forceBranch := cmd.Bool("force-branch")

	if err := validateRemoveInput(worktreeName, withBranch, forceBranch); err != nil {
		return err
	}

	// Get current working directory
	cwd, err := removeGetwd()
	if err != nil {
		return errors.DirectoryAccessFailed("access current", ".", err)
	}

	// Initialize repository to check if we're in a git repo
	_, err = git.NewRepository(cwd)
	if err != nil {
		return errors.NotInGitRepository()
	}

	// Use CommandExecutor-based implementation
	executor := command.NewRealExecutor()
	return removeCommandWithCommandExecutor(cmd, w, executor, cwd, worktreeName, force, withBranch, forceBranch)
}

func removeCommandWithCommandExecutor(
	_ *cli.Command,
	w io.Writer,
	executor command.Executor,
	cwd string,
	worktreeName string,
	force, withBranch, forceBranch bool,
) error {
	// Get worktrees using CommandExecutor
	listCmd := command.GitWorktreeList()
	result, err := executor.Execute([]command.Command{listCmd})
	if err != nil {
		return errors.GitCommandFailed("git worktree list", err.Error())
	}

	// Parse worktrees from command output
	worktrees := parseWorktreesFromOutput(result.Results[0].Output)

	// Find target worktree
	targetWorktree, err := findTargetWorktreeFromList(worktrees, worktreeName)
	if err != nil {
		return err
	}

	absTargetPath, err := filepath.Abs(targetWorktree.Path)
	if err != nil {
		return errors.WorktreeRemovalFailed(targetWorktree.Path, err)
	}

	absCwd, err := filepath.Abs(cwd)
	if err != nil {
		return errors.DirectoryAccessFailed("access current", cwd, err)
	}

	if isPathWithin(absTargetPath, absCwd) {
		return errors.CannotRemoveCurrentWorktree(worktreeName, absTargetPath)
	}

	// Remove worktree using CommandExecutor
	removeCmd := command.GitWorktreeRemove(targetWorktree.Path, force)
	result, err = executor.Execute([]command.Command{removeCmd})
	if err != nil {
		return errors.WorktreeRemovalFailed(targetWorktree.Path, err)
	}
	if len(result.Results) > 0 && result.Results[0].Error != nil {
		gitOutput := result.Results[0].Output
		if gitOutput != "" {
			combinedError := fmt.Errorf("%v: %s", result.Results[0].Error, gitOutput)
			return errors.WorktreeRemovalFailed(targetWorktree.Path, combinedError)
		}
		return errors.WorktreeRemovalFailed(targetWorktree.Path, result.Results[0].Error)
	}
	fmt.Fprintf(w, "Removed worktree '%s' at %s\n", worktreeName, targetWorktree.Path)

	// Remove branch if requested
	if withBranch && targetWorktree.Branch != "" {
		if err := removeBranchWithCommandExecutor(w, executor, targetWorktree.Branch, forceBranch); err != nil {
			return err
		}
	}

	return nil
}

func validateRemoveInput(worktreeName string, withBranch, forceBranch bool) error {
	if worktreeName == "" {
		return errors.WorktreeNameRequiredForRemove()
	}
	if forceBranch && !withBranch {
		return fmt.Errorf("--force-branch requires --with-branch")
	}
	return nil
}

func isPathWithin(basePath, targetPath string) bool {
	rel, err := filepath.Rel(basePath, targetPath)
	if err != nil {
		return false
	}

	if rel == "." || rel == "" {
		return true
	}

	if rel == ".." || strings.HasPrefix(rel, ".."+string(os.PathSeparator)) {
		return false
	}

	return true
}

func removeBranchWithCommandExecutor(
	w io.Writer,
	executor command.Executor,
	branchName string,
	forceBranch bool,
) error {
	branchCmd := command.GitBranchDelete(branchName, forceBranch)
	result, err := executor.Execute([]command.Command{branchCmd})
	if err != nil {
		return errors.BranchRemovalFailed(branchName, err, forceBranch)
	}
	if len(result.Results) > 0 && result.Results[0].Error != nil {
		gitOutput := result.Results[0].Output
		if gitOutput != "" {
			combinedError := fmt.Errorf("%v: %s", result.Results[0].Error, gitOutput)
			return errors.BranchRemovalFailed(branchName, combinedError, forceBranch)
		}
		return errors.BranchRemovalFailed(branchName, result.Results[0].Error, forceBranch)
	}
	fmt.Fprintf(w, "Removed branch '%s'\n", branchName)
	return nil
}

func findTargetWorktreeFromList(worktrees []git.Worktree, worktreeName string) (*git.Worktree, error) {
	var targetWorktree *git.Worktree
	var availableWorktrees []string

	// Find main worktree path for consistent naming
	mainWorktreePath := ""
	for _, wt := range worktrees {
		if wt.IsMain {
			mainWorktreePath = wt.Path
			break
		}
	}

	// Load config for consistent worktree naming
	cfg, err := config.LoadConfig(mainWorktreePath)
	if err != nil {
		// If config can't be loaded, use default config
		cfg = &config.Config{
			Defaults: config.Defaults{
				BaseDir: config.DefaultBaseDir,
			},
		}
	}

	for _, wt := range worktrees {
		// Skip main worktree - it cannot be removed
		if wt.IsMain {
			continue
		}

		// Skip unmanaged worktrees - they cannot be removed by wtp
		if !isWorktreeManaged(wt.Path, cfg, mainWorktreePath, wt.IsMain) {
			continue
		}

		// Priority 1: Match by branch name (for prefixes like feature/awesome)
		if wt.Branch == worktreeName {
			targetWorktree = &wt
		}

		// Priority 2: Match by directory name (legacy behavior)
		wtName := filepath.Base(wt.Path)
		if wtName == worktreeName {
			targetWorktree = &wt
		}

		// Priority 3: Match by worktree name (relative to base_dir)
		worktreeDisplayName := getWorktreeNameFromPath(wt.Path, cfg, mainWorktreePath, wt.IsMain)
		if worktreeDisplayName == worktreeName {
			targetWorktree = &wt
		}

		// Build available worktrees list using consistent naming (excluding main worktree and unmanaged)
		availableWorktrees = append(availableWorktrees, worktreeDisplayName)

		// Exit early if we found a match
		if targetWorktree != nil {
			break
		}
	}

	if targetWorktree == nil {
		return nil, errors.WorktreeNotFound(worktreeName, availableWorktrees)
	}
	return targetWorktree, nil
}

// getWorktreeNameFromPath calculates the worktree name from its path
// For main worktree, returns "@"
// For other worktrees, returns relative path from base_dir
func getWorktreeNameFromPath(worktreePath string, cfg *config.Config, mainRepoPath string, isMain bool) string {
	if isMain {
		return "@"
	}

	// Get base_dir path
	baseDir := cfg.Defaults.BaseDir
	if !filepath.IsAbs(baseDir) {
		baseDir = filepath.Join(mainRepoPath, baseDir)
	}

	// Calculate relative path from base_dir
	relPath, err := filepath.Rel(baseDir, worktreePath)
	if err != nil {
		// Fallback to directory name
		return filepath.Base(worktreePath)
	}

	return relPath
}

// getWorktreesForRemove gets worktrees for remove command and writes them to writer (testable)
func getWorktreesForRemove(w io.Writer) error {
	// Get current directory
	cwd, err := removeGetwd() // Use mockable function for tests
	if err != nil {
		return err
	}

	// Initialize repository
	repo, err := git.NewRepository(cwd)
	if err != nil {
		return err
	}

	// Get main worktree path
	mainRepoPath, err := repo.GetMainWorktreePath()
	if err != nil {
		return err
	}

	// Load config
	cfg, err := config.LoadConfig(mainRepoPath)
	if err != nil {
		return err
	}

	// Get all worktrees
	worktrees, err := repo.GetWorktrees()
	if err != nil {
		return err
	}

	// Print worktrees for remove command (no main, no markers, managed only)
	for i := range worktrees {
		wt := &worktrees[i]
		if !wt.IsMain && isWorktreeManaged(wt.Path, cfg, mainRepoPath, wt.IsMain) {
			// Calculate worktree name as relative path from base_dir
			name := getWorktreeNameFromPath(wt.Path, cfg, mainRepoPath, wt.IsMain)
			fmt.Fprintln(w, name)
		}
	}

	return nil
}

// completeWorktrees provides worktree name completion for urfave/cli (wrapper for getWorktreesForRemove)
func completeWorktrees(_ context.Context, cmd *cli.Command) {
	current, previous := completionArgsFromCommand(cmd)

	if maybeCompleteFlagSuggestions(cmd, current, previous) {
		return
	}

	currentNormalized := strings.TrimSuffix(current, "*")

	var buf bytes.Buffer
	if err := getWorktreesForRemove(&buf); err != nil {
		return
	}

	used := make(map[string]struct{}, len(previous))
	for _, arg := range previous {
		if arg == "" || strings.HasPrefix(arg, "-") {
			continue
		}
		key := strings.TrimSuffix(arg, "*")
		used[key] = struct{}{}
	}

	// Output each line using fmt.Println for urfave/cli compatibility
	scanner := bufio.NewScanner(&buf)
	for scanner.Scan() {
		name := scanner.Text()
		if _, exists := used[name]; exists {
			continue
		}
		if currentNormalized != "" && name == currentNormalized {
			continue
		}
		fmt.Println(name)
	}
}
