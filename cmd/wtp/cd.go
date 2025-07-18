package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/satococoa/wtp/internal/command"
	"github.com/satococoa/wtp/internal/config"
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

	// Find the main worktree path
	mainWorktreePath := findMainWorktreePath(worktrees)

	// Find the worktree using multiple resolution strategies
	targetPath := resolveCdWorktreePath(worktreeName, worktrees, mainWorktreePath)

	if targetPath == "" {
		// Get available worktree names for suggestions
		availableWorktrees := make([]string, 0, len(worktrees))

		// Load config and main repo path to get proper worktree names
		mainRepoPath := findMainWorktreePath(worktrees)
		cfg, err := config.LoadConfig(mainRepoPath)
		if err != nil {
			// Fallback to directory names if config can't be loaded
			for i := range worktrees {
				availableWorktrees = append(availableWorktrees, filepath.Base(worktrees[i].Path))
			}
		} else {
			// Use consistent worktree names (relative to base_dir)
			for i := range worktrees {
				name := getWorktreeNameFromPath(worktrees[i].Path, cfg, mainRepoPath, worktrees[i].IsMain)
				availableWorktrees = append(availableWorktrees, name)
			}
		}
		return errors.WorktreeNotFound(worktreeName, availableWorktrees)
	}

	// Output the path for the shell function to cd to
	fmt.Fprintln(w, targetPath)
	return nil
}

// findMainWorktreePath finds the main worktree from the list of worktrees
func findMainWorktreePath(worktrees []git.Worktree) string {
	// The first worktree is always the main worktree (git worktree list behavior)
	if len(worktrees) > 0 {
		return worktrees[0].Path
	}
	return ""
}

// resolveCdWorktreePath resolves a worktree name to its path using multiple strategies
func resolveCdWorktreePath(worktreeName string, worktrees []git.Worktree, mainWorktreePath string) string {
	// Remove asterisk marker from completion (e.g., "feature*" → "feature", "@*" → "@")
	worktreeName = strings.TrimSuffix(worktreeName, "*")

	// Load config for unified naming
	cfg, _ := config.LoadConfig(mainWorktreePath)
	var worktreeNameFromPath string

	// The order matters: more specific matches come first
	for i := range worktrees {
		wt := &worktrees[i]

		// Priority 1: Exact branch name match (supports prefixes like "feature/awesome")
		if wt.Branch == worktreeName {
			return wt.Path
		}

		// Priority 2: Unified worktree name match (relative to base_dir)
		if cfg != nil {
			worktreeNameFromPath = getWorktreeNameFromPath(wt.Path, cfg, mainWorktreePath, wt.IsMain)
			if worktreeNameFromPath == worktreeName {
				return wt.Path
			}
		}

		// Priority 3: Worktree directory name match (legacy behavior)
		if filepath.Base(wt.Path) == worktreeName {
			return wt.Path
		}

		// Priority 4: Root worktree alias ("root" → main worktree)
		if worktreeName == "root" && wt.IsMainWorktree(mainWorktreePath) {
			return wt.Path
		}

		// Priority 5: @ symbol for main worktree ("@" → main worktree)
		if worktreeName == "@" && wt.IsMainWorktree(mainWorktreePath) {
			return wt.Path
		}

		// Priority 6: Repository name for root worktree ("giselle" → root worktree)
		if worktreeName == filepath.Base(wt.Path) && wt.IsMainWorktree(mainWorktreePath) {
			return wt.Path
		}

		// Priority 7: Legacy completion display format ("wtp(root worktree)" → root worktree)
		repoRootFormat := filepath.Base(wt.Path) + "(root worktree)"
		if worktreeName == repoRootFormat && wt.IsMainWorktree(mainWorktreePath) {
			return wt.Path
		}

		// Priority 8: Current completion display format ("giselle@fix-nodes(root worktree)" → root worktree)
		if strings.HasSuffix(worktreeName, "(root worktree)") && wt.IsMainWorktree(mainWorktreePath) {
			// Extract repo name and branch from format "repo@branch(root worktree)"
			prefix := strings.TrimSuffix(worktreeName, "(root worktree)")
			// Check if this matches the worktree by comparing repo name and branch
			expectedPrefix := filepath.Base(wt.Path) + "@" + wt.Branch
			if prefix == expectedPrefix {
				return wt.Path
			}
		}
	}

	return ""
}
