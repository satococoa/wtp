package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/satococoa/wtp/internal/command"
	"github.com/satococoa/wtp/internal/errors"
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
			"  Bash: eval \"$(wtp completion bash)\"\n" +
			"  Zsh:  eval \"$(wtp completion zsh)\"\n" +
			"  Fish: wtp completion fish | source",
		ArgsUsage: "<worktree-name>",
		Action:    cdToWorktree,
	}
}

func cdToWorktree(_ context.Context, cmd *cli.Command) error {
	// Check if we're running inside the shell function
	if os.Getenv("WTP_SHELL_INTEGRATION") != "1" {
		return errors.ShellIntegrationRequired()
	}

	args := cmd.Args()
	if args.Len() == 0 {
		return errors.WorktreeNameRequired()
	}

	worktreeName := args.Get(0)

	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return errors.DirectoryAccessFailed("access current", ".", err)
	}

	// Initialize repository to check if we're in a git repo
	_, err = git.NewRepository(cwd)
	if err != nil {
		return errors.NotInGitRepository()
	}

	// Get the writer from cli.Command
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	// Use CommandExecutor-based implementation
	executor := command.NewRealExecutor()
	return cdCommandWithCommandExecutor(cmd, w, executor, cwd, worktreeName)
}

func cdCommandWithCommandExecutor(
	_ *cli.Command,
	w io.Writer,
	executor command.Executor,
	_ string,
	worktreeName string,
) error {
	// Get worktrees using CommandExecutor
	listCmd := command.GitWorktreeList()
	result, err := executor.Execute([]command.Command{listCmd})
	if err != nil {
		return fmt.Errorf("failed to get worktrees: %w", err)
	}

	// Parse worktrees from command output
	worktrees := parseWorktreesFromOutput(result.Results[0].Output)

	// Find the worktree by name
	var targetPath string
	for _, wt := range worktrees {
		// Priority 1: Match by branch name (for prefixes like feature/awesome)
		if wt.Branch == worktreeName {
			targetPath = wt.Path
			break
		}

		// Priority 2: Match by directory name (legacy behavior)
		if filepath.Base(wt.Path) == worktreeName {
			targetPath = wt.Path
			break
		}

		// Priority 3: Special handling for root worktree aliases
		if worktreeName == "root" && (wt.Branch == git.MainBranch || wt.Branch == git.MasterBranch) {
			targetPath = wt.Path
			break
		}

		// Priority 4: Handle completion display format "reponame(root worktree)"
		repoRootFormat := filepath.Base(wt.Path) + "(root worktree)"
		if worktreeName == repoRootFormat && (wt.Branch == git.MainBranch || wt.Branch == git.MasterBranch) {
			targetPath = wt.Path
			break
		}
	}

	if targetPath == "" {
		// Get available worktree names for suggestions
		availableWorktrees := make([]string, 0, len(worktrees))
		for _, wt := range worktrees {
			availableWorktrees = append(availableWorktrees, filepath.Base(wt.Path))
		}
		return errors.WorktreeNotFound(worktreeName, availableWorktrees)
	}

	// Output the path for the shell function to cd to
	fmt.Fprintln(w, targetPath)
	return nil
}
