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
	"golang.org/x/term"
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
	listNewExecutor  = command.NewRealExecutor // Add this for mocking
	getTerminalWidth = func() int {
		width, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil || width <= 0 {
			return 80 //nolint:mnd // Default terminal width
		}
		return width
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

// truncatePath truncates a path to fit within the given width, showing beginning and end
func truncatePath(path string, maxWidth int) string {
	if len(path) <= maxWidth {
		return path
	}

	// Reserve space for ellipsis "..."
	const ellipsis = "..."
	if maxWidth <= len(ellipsis) {
		return path[:maxWidth]
	}

	availableWidth := maxWidth - len(ellipsis)
	// Show more of the end (file/directory name) than the beginning
	startLen := availableWidth / 3 //nolint:mnd // Show 1/3 of start, 2/3 of end
	endLen := availableWidth - startLen

	return path[:startLen] + ellipsis + path[len(path)-endLen:]
}

// displayWorktrees formats and displays worktree information
func displayWorktrees(w io.Writer, worktrees []git.Worktree) {
	termWidth := getTerminalWidth()

	// Minimum widths for columns
	const minPathWidth = 20
	const headWidth = headDisplayLength
	const spacing = 3 // Spaces between columns

	// Calculate initial column widths
	maxBranchLen := 6 // "BRANCH"
	for _, wt := range worktrees {
		branchDisplay := formatBranchDisplay(wt.Branch)
		if len(branchDisplay) > maxBranchLen {
			maxBranchLen = len(branchDisplay)
		}
	}

	// Calculate available width for path column
	// Total = path + spacing + branch + spacing + head
	availableForPath := termWidth - spacing - maxBranchLen - spacing - headWidth

	// If branch column is too wide, limit it as well
	maxAvailableForBranch := termWidth - minPathWidth - spacing - spacing - headWidth
	if maxBranchLen > maxAvailableForBranch {
		maxBranchLen = maxAvailableForBranch
		// Recalculate path width with truncated branch width
		availableForPath = termWidth - spacing - maxBranchLen - spacing - headWidth
	}

	// Ensure minimum path width
	if availableForPath < minPathWidth {
		availableForPath = minPathWidth
	}

	// Print header
	fmt.Fprintf(w, "%-*s %-*s %s\n", availableForPath, "PATH", maxBranchLen, "BRANCH", "HEAD")
	fmt.Fprintf(w, "%-*s %-*s %s\n",
		availableForPath, strings.Repeat("-", pathHeaderDashes),
		maxBranchLen, strings.Repeat("-", branchHeaderDashes),
		"----")

	// Print worktrees
	for _, wt := range worktrees {
		headShort := wt.HEAD
		if len(headShort) > headDisplayLength {
			headShort = headShort[:headDisplayLength]
		}

		branchDisplay := formatBranchDisplay(wt.Branch)
		pathDisplay := truncatePath(wt.Path, availableForPath)
		branchDisplayTrunc := truncatePath(branchDisplay, maxBranchLen)

		fmt.Fprintf(w, "%-*s %-*s %s\n", availableForPath, pathDisplay, maxBranchLen, branchDisplayTrunc, headShort)
	}
}
