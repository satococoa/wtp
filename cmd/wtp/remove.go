package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/satococoa/wtp/internal/command"
	"github.com/satococoa/wtp/internal/config"
	"github.com/satococoa/wtp/internal/errors"
	"github.com/satococoa/wtp/internal/git"
	"github.com/urfave/cli/v3"
)

// Variable to allow mocking in tests
var removeGetwd = os.Getwd

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
	_ string,
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

	// Remove worktree using CommandExecutor
	removeCmd := command.GitWorktreeRemove(targetWorktree.Path, force)
	_, err = executor.Execute([]command.Command{removeCmd})
	if err != nil {
		return errors.WorktreeRemovalFailed(targetWorktree.Path, err)
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
		return errors.BranchNameRequired("wtp remove <worktree-name>")
	}
	if forceBranch && !withBranch {
		return fmt.Errorf("--force-branch requires --with-branch")
	}
	return nil
}

func removeBranchWithCommandExecutor(
	w io.Writer,
	executor command.Executor,
	branchName string,
	forceBranch bool,
) error {
	branchCmd := command.GitBranchDelete(branchName, forceBranch)
	_, err := executor.Execute([]command.Command{branchCmd})
	if err != nil {
		return errors.BranchRemovalFailed(branchName, err, forceBranch)
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
		// Fallback to old behavior if config can't be loaded
		return findTargetWorktreeFromListFallback(worktrees, worktreeName)
	}

	for _, wt := range worktrees {
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

		// Build available worktrees list using consistent naming
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

// findTargetWorktreeFromListFallback provides fallback behavior when config can't be loaded
func findTargetWorktreeFromListFallback(worktrees []git.Worktree, worktreeName string) (*git.Worktree, error) {
	var targetWorktree *git.Worktree
	var availableWorktrees []string

	for _, wt := range worktrees {
		// Priority 1: Match by branch name (for prefixes like feature/awesome)
		if wt.Branch == worktreeName {
			targetWorktree = &wt
		}

		// Priority 2: Match by directory name (legacy behavior)
		wtName := filepath.Base(wt.Path)
		if wtName == worktreeName {
			targetWorktree = &wt
		}

		// Build available worktrees list - prefer branch name if available
		if wt.Branch != "" {
			availableWorktrees = append(availableWorktrees, wt.Branch)
		} else {
			availableWorktrees = append(availableWorktrees, wtName)
		}

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
