package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/satococoa/wtp/internal/command"
	"github.com/satococoa/wtp/internal/errors"
	"github.com/satococoa/wtp/internal/git"
	"github.com/urfave/cli/v3"
)

// Display constants
const (
	pathHeaderDashes   = 4
	branchHeaderDashes = 6
	headDisplayLength  = 8
	detachedKeyword    = "detached"
)

// GitRepository interface for mocking
type GitRepository interface {
	GetWorktrees() ([]git.Worktree, error)
}

// Variables to allow mocking in tests
var (
	listGetwd         = os.Getwd
	listNewRepository = func(path string) (GitRepository, error) {
		return git.NewRepository(path)
	}
	listNewExecutor = command.NewRealExecutor // Add this for mocking
)

// NewListCommand creates the list command definition
func NewListCommand() *cli.Command {
	return &cli.Command{
		Name:        "list",
		Usage:       "List all worktrees",
		Description: "Shows all worktrees with their paths, branches, and HEAD commits.",
		Action:      listCommand,
	}
}

func listCommand(_ context.Context, cmd *cli.Command) error {
	// Get current working directory (should be a git repository)
	cwd, err := listGetwd()
	if err != nil {
		return errors.DirectoryAccessFailed("access current", ".", err)
	}

	// Initialize repository to check if we're in a git repo
	_, err = listNewRepository(cwd)
	if err != nil {
		return errors.NotInGitRepository()
	}

	// Get the writer from cli.Command
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	// Use CommandExecutor-based implementation
	executor := listNewExecutor()
	return listCommandWithCommandExecutor(cmd, w, executor, cwd)
}

func listCommandWithCommandExecutor(_ *cli.Command, w io.Writer, executor command.Executor, _ string) error {
	// Get worktrees using CommandExecutor
	listCmd := command.GitWorktreeList()
	result, err := executor.Execute([]command.Command{listCmd})
	if err != nil {
		return errors.GitCommandFailed("git worktree list", err.Error())
	}

	// Parse worktrees from command output
	worktrees := parseWorktreesFromOutput(result.Results[0].Output)

	if len(worktrees) == 0 {
		fmt.Fprintln(w, "No worktrees found")
		return nil
	}

	// Display worktrees
	displayWorktrees(w, worktrees)
	return nil
}

func parseWorktreesFromOutput(output string) []git.Worktree {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var worktrees []git.Worktree
	var currentWorktree git.Worktree

	for _, line := range lines {
		if line == "" {
			if currentWorktree.Path != "" {
				worktrees = append(worktrees, currentWorktree)
				currentWorktree = git.Worktree{}
			}
			continue
		}

		if strings.HasPrefix(line, "worktree ") {
			currentWorktree.Path = strings.TrimPrefix(line, "worktree ")
		} else if strings.HasPrefix(line, "HEAD ") {
			currentWorktree.HEAD = strings.TrimPrefix(line, "HEAD ")
		} else if strings.HasPrefix(line, "branch ") {
			currentWorktree.Branch = strings.TrimPrefix(line, "branch refs/heads/")
		} else if line == detachedKeyword {
			currentWorktree.Branch = detachedKeyword
		}
	}

	if currentWorktree.Path != "" {
		worktrees = append(worktrees, currentWorktree)
	}

	return worktrees
}

// formatBranchDisplay formats branch name for display, following Git conventions
func formatBranchDisplay(branch string) string {
	if branch == detachedKeyword {
		return "(detached HEAD)"
	}
	if branch == "" {
		return "(no branch)"
	}
	return branch
}

// displayWorktrees formats and displays worktree information
func displayWorktrees(w io.Writer, worktrees []git.Worktree) {
	// Calculate column widths dynamically
	maxPathLen := 4   // "PATH"
	maxBranchLen := 6 // "BRANCH"

	for _, wt := range worktrees {
		if len(wt.Path) > maxPathLen {
			maxPathLen = len(wt.Path)
		}

		// Calculate branch display length (considering "(detached)" formatting)
		branchDisplay := formatBranchDisplay(wt.Branch)
		if len(branchDisplay) > maxBranchLen {
			maxBranchLen = len(branchDisplay)
		}
	}

	// Add some padding
	maxPathLen += 2
	maxBranchLen += 2

	// Print header
	fmt.Fprintf(w, "%-*s %-*s %s\n", maxPathLen, "PATH", maxBranchLen, "BRANCH", "HEAD")
	fmt.Fprintf(w, "%-*s %-*s %s\n",
		maxPathLen, strings.Repeat("-", pathHeaderDashes),
		maxBranchLen, strings.Repeat("-", branchHeaderDashes),
		"----")

	// Print worktrees
	for _, wt := range worktrees {
		headShort := wt.HEAD
		if len(headShort) > headDisplayLength {
			headShort = headShort[:headDisplayLength]
		}

		branchDisplay := formatBranchDisplay(wt.Branch)
		fmt.Fprintf(w, "%-*s %-*s %s\n", maxPathLen, wt.Path, maxBranchLen, branchDisplay, headShort)
	}
}
