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
	repo, err := listNewRepository(cwd)
	if err != nil {
		return errors.NotInGitRepository()
	}

	// Get main worktree path
	mainRepoPath, err := repo.(*git.Repository).GetMainWorktreePath()
	if err != nil {
		return errors.GitCommandFailed("get main worktree path", err.Error())
	}

	// Get the writer from cli.Command
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	// Load config to get base_dir
	cfg, _ := config.LoadConfig(mainRepoPath)

	// Use CommandExecutor-based implementation
	executor := listNewExecutor()
	return listCommandWithCommandExecutor(cmd, w, executor, cfg, mainRepoPath)
}

func listCommandWithCommandExecutor(
	_ *cli.Command, w io.Writer, executor command.Executor, cfg *config.Config, mainRepoPath string,
) error {
	// Get current working directory
	cwd, err := listGetwd()
	if err != nil {
		return errors.DirectoryAccessFailed("access current", ".", err)
	}

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

	// Display worktrees with relative paths
	displayWorktreesRelative(w, worktrees, cwd, cfg, mainRepoPath)
	return nil
}

func parseWorktreesFromOutput(output string) []git.Worktree {
	lines := strings.Split(strings.TrimSpace(output), "\n")
	var worktrees []git.Worktree
	var currentWorktree git.Worktree
	isFirst := true

	for _, line := range lines {
		if line == "" {
			if currentWorktree.Path != "" {
				// First worktree is always the main worktree
				if isFirst {
					currentWorktree.IsMain = true
					isFirst = false
				}
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
		// First worktree is always the main worktree
		if isFirst {
			currentWorktree.IsMain = true
		}
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


// displayWorktreesRelative formats and displays worktree information with relative paths
func displayWorktreesRelative(
	w io.Writer, worktrees []git.Worktree, currentPath string, cfg *config.Config, mainRepoPath string,
) {
	termWidth := getTerminalWidth()

	// Minimum widths for columns
	const minPathWidth = 20
	const headWidth = headDisplayLength
	const spacing = 3 // Spaces between columns

	// Calculate initial column widths
	maxBranchLen := 6 // "BRANCH"
	maxPathLen := 4   // "PATH"

	// Find main worktree path is no longer needed since we pass it from the caller

	// First pass: calculate max widths and prepare display data
	type displayData struct {
		path      string
		branch    string
		head      string
		isCurrent bool
	}

	var displayItems []displayData

	for _, wt := range worktrees {
		var pathDisplay string
		var isCurrent bool

		// Use unified worktree naming function
		if cfg != nil {
			pathDisplay = getWorktreeNameFromPath(wt.Path, cfg, mainRepoPath, wt.IsMain)
		} else {
			// Fallback when config can't be loaded
			if wt.IsMain {
				pathDisplay = "@"
			} else {
				pathDisplay = filepath.Base(wt.Path)
			}
		}

		// Check if this is the current worktree
		if wt.Path == currentPath {
			isCurrent = true
			pathDisplay += "*"
		}

		branchDisplay := formatBranchDisplay(wt.Branch)

		if len(pathDisplay) > maxPathLen {
			maxPathLen = len(pathDisplay)
		}
		if len(branchDisplay) > maxBranchLen {
			maxBranchLen = len(branchDisplay)
		}

		displayItems = append(displayItems, displayData{
			path:      pathDisplay,
			branch:    branchDisplay,
			head:      wt.HEAD,
			isCurrent: isCurrent,
		})
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
	for _, item := range displayItems {
		headShort := item.head
		if len(headShort) > headDisplayLength {
			headShort = headShort[:headDisplayLength]
		}

		pathDisplay := truncatePath(item.path, availableForPath)
		branchDisplayTrunc := truncatePath(item.branch, maxBranchLen)

		fmt.Fprintf(w, "%-*s %-*s %s\n", availableForPath, pathDisplay, maxBranchLen, branchDisplayTrunc, headShort)
	}
}
