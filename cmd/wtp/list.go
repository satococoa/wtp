package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/satococoa/wtp/internal/errors"
	"github.com/satococoa/wtp/internal/git"
	"github.com/urfave/cli/v3"
)

// Display constants
const (
	pathHeaderDashes   = 4
	branchHeaderDashes = 6
	headDisplayLength  = 8
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

	// Initialize repository
	repo, err := listNewRepository(cwd)
	if err != nil {
		return errors.NotInGitRepository()
	}

	// Get worktrees
	worktrees, err := repo.GetWorktrees()
	if err != nil {
		return errors.GitCommandFailed("git worktree list", err.Error())
	}

	// Get the writer from cli.Command
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	if len(worktrees) == 0 {
		fmt.Fprintln(w, "No worktrees found")
		return nil
	}

	// Display worktrees
	displayWorktrees(w, worktrees)
	return nil
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
		if len(wt.Branch) > maxBranchLen {
			maxBranchLen = len(wt.Branch)
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
		fmt.Fprintf(w, "%-*s %-*s %s\n", maxPathLen, wt.Path, maxBranchLen, wt.Branch, headShort)
	}
}
