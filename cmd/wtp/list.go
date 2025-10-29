package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"github.com/urfave/cli/v3"
	"golang.org/x/term"

	"github.com/satococoa/wtp/internal/command"
	"github.com/satococoa/wtp/internal/config"
	"github.com/satococoa/wtp/internal/errors"
	"github.com/satococoa/wtp/internal/git"
)

// Display constants
const (
	pathHeaderDashes   = 4
	branchHeaderDashes = 6
	headDisplayLength  = 8
	detachedKeyword    = "detached"
)

const (
	defaultMaxPathWidth = 56
	superWideThreshold  = 160
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
		Name:          "list",
		Aliases:       []string{"ls"},
		Usage:         "List all worktrees",
		Description:   "Shows all worktrees with their paths, branches, and HEAD commits.",
		ShellComplete: completeList,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "compact",
				Aliases: []string{"c"},
				Usage:   "Minimize column widths for narrow or redirected output",
			},
			&cli.IntFlag{
				Name:  "max-path-width",
				Usage: fmt.Sprintf("Maximum width for PATH column (default %d)", defaultMaxPathWidth),
				Value: defaultMaxPathWidth,
			},
			&cli.BoolFlag{
				Name:    "quiet",
				Aliases: []string{"q"},
				Usage:   "Only display worktree paths",
			},
		},
		Action: listCommand,
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

	// Resolve display options
	opts := resolveListDisplayOptions(cmd, w)

	// Get quiet flag
	quiet := cmd.Bool("quiet")

	// Use CommandExecutor-based implementation
	executor := listNewExecutor()
	return listCommandWithCommandExecutor(cmd, w, executor, cfg, mainRepoPath, quiet, opts)
}

func listCommandWithCommandExecutor(
	_ *cli.Command, w io.Writer, executor command.Executor, cfg *config.Config, mainRepoPath string, quiet bool,
	opts listDisplayOptions,
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
		if !quiet {
			fmt.Fprintln(w, "No worktrees found")
		}
		return nil
	}

	// Display worktrees
	if quiet {
		displayWorktreesQuiet(w, worktrees, cfg, mainRepoPath)
	} else {
		termWidth := getTerminalWidth()
		if opts.MaxPathWidth <= 0 {
			opts.MaxPathWidth = defaultMaxPathWidth
		}
		if !opts.Compact {
			if !opts.OutputIsTTY {
				opts.Compact = true
			} else if termWidth >= superWideThreshold {
				opts.Compact = true
			}
		}
		displayWorktreesRelative(w, worktrees, cwd, cfg, mainRepoPath, termWidth, opts)
	}
	return nil
}

// completeList provides shell completion for the list command (flags only)
func completeList(_ context.Context, cmd *cli.Command) {
	current, previous := completionArgsFromCommand(cmd)
	maybeCompleteFlagSuggestions(cmd, current, previous)
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

// isWorktreeManagedList determines if a worktree is managed by wtp (for list command)
func isWorktreeManagedList(worktreePath string, cfg *config.Config, mainRepoPath string, isMain bool) bool {
	return isWorktreeManagedCommon(worktreePath, cfg, mainRepoPath, isMain)
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

// getWorktreeDisplayName returns the display name for a worktree, with fallback for nil config
func getWorktreeDisplayName(wt git.Worktree, cfg *config.Config, mainRepoPath string) string {
	if cfg != nil {
		return getWorktreeNameFromPath(wt.Path, cfg, mainRepoPath, wt.IsMain)
	}
	// Fallback when config can't be loaded
	if wt.IsMain {
		return "@"
	}
	return filepath.Base(wt.Path)
}

// displayWorktreesQuiet outputs only the worktree names (as shown in PATH column), one per line
func displayWorktreesQuiet(w io.Writer, worktrees []git.Worktree, cfg *config.Config, mainRepoPath string) {
	for _, wt := range worktrees {
		pathDisplay := getWorktreeDisplayName(wt, cfg, mainRepoPath)
		fmt.Fprintln(w, pathDisplay)
	}
}

// displayWorktreesRelative formats and displays worktree information with relative paths
func displayWorktreesRelative(
	w io.Writer, worktrees []git.Worktree, currentPath string, cfg *config.Config, mainRepoPath string,
	termWidth int, opts listDisplayOptions,
) {
	// Minimum widths for columns
	const minPathWidth = 20
	const headWidth = headDisplayLength
	const spacing = 3 // Spaces between columns

	// Calculate initial column widths
	maxBranchLen := 6 // "BRANCH"
	maxPathLen := 4   // "PATH"
	maxStatusLen := 6 // "STATUS"

	// Find main worktree path is no longer needed since we pass it from the caller

	// First pass: calculate max widths and prepare display data
	type displayData struct {
		path      string
		branch    string
		head      string
		status    string
		isCurrent bool
	}

	var displayItems []displayData

	for _, wt := range worktrees {
		var isCurrent bool

		// Get worktree display name
		pathDisplay := getWorktreeDisplayName(wt, cfg, mainRepoPath)

		// Check if this is the current worktree
		if wt.Path == currentPath {
			isCurrent = true
			pathDisplay += "*"
		}

		branchDisplay := formatBranchDisplay(wt.Branch)

		// Determine management status
		var statusDisplay string
		if isWorktreeManagedList(wt.Path, cfg, mainRepoPath, wt.IsMain) {
			statusDisplay = "managed"
		} else {
			statusDisplay = "unmanaged"
		}

		if len(pathDisplay) > maxPathLen {
			maxPathLen = len(pathDisplay)
		}
		if len(branchDisplay) > maxBranchLen {
			maxBranchLen = len(branchDisplay)
		}
		if len(statusDisplay) > maxStatusLen {
			maxStatusLen = len(statusDisplay)
		}

		displayItems = append(displayItems, displayData{
			path:      pathDisplay,
			branch:    branchDisplay,
			head:      wt.HEAD,
			status:    statusDisplay,
			isCurrent: isCurrent,
		})
	}

	// Calculate available width for path column
	// Total = path + spacing + branch + spacing + status + spacing + head
	if termWidth <= 0 {
		termWidth = 80
	}
	// If branch column is too wide, limit it as well
	maxAvailableForBranch := termWidth - minPathWidth - spacing - maxStatusLen - spacing - spacing - headWidth
	if maxBranchLen > maxAvailableForBranch {
		maxBranchLen = maxAvailableForBranch
	}

	pathHeaderWidth := len("PATH")
	branchHeaderWidth := len("BRANCH")
	statusHeaderWidth := len("STATUS")

	if maxBranchLen < branchHeaderWidth {
		maxBranchLen = branchHeaderWidth
	}
	if maxStatusLen < statusHeaderWidth {
		maxStatusLen = statusHeaderWidth
	}

	availableForPath := termWidth - spacing - maxBranchLen - spacing - maxStatusLen - spacing - headWidth

	if availableForPath < pathHeaderWidth {
		availableForPath = pathHeaderWidth
	}

	pathWidth := availableForPath

	if opts.MaxPathWidth > 0 && pathWidth > opts.MaxPathWidth {
		pathWidth = opts.MaxPathWidth
	}

	if opts.Compact {
		minCompactWidth := pathHeaderWidth
		if maxPathLen > minCompactWidth {
			minCompactWidth = maxPathLen
		}
		if pathWidth > minCompactWidth {
			pathWidth = minCompactWidth
		}
		if pathWidth < minCompactWidth {
			pathWidth = minCompactWidth
		}
	} else {
		desiredWidth := maxPathLen + 2
		if desiredWidth < minPathWidth {
			desiredWidth = minPathWidth
		}
		if desiredWidth < pathHeaderWidth {
			desiredWidth = pathHeaderWidth
		}
		if pathWidth > desiredWidth {
			pathWidth = desiredWidth
		}
		if pathWidth < minPathWidth {
			pathWidth = minPathWidth
		}
	}

	if pathWidth > availableForPath {
		pathWidth = availableForPath
	}
	if pathWidth < pathHeaderWidth {
		pathWidth = pathHeaderWidth
	}
	if pathWidth < 1 {
		pathWidth = 1
	}

	// Print header
	fmt.Fprintf(w, "%-*s %-*s %-*s %s\n", pathWidth, "PATH", maxBranchLen, "BRANCH", maxStatusLen, "STATUS", "HEAD")
	fmt.Fprintf(w, "%-*s %-*s %-*s %s\n",
		pathWidth, strings.Repeat("-", pathHeaderDashes),
		maxBranchLen, strings.Repeat("-", branchHeaderDashes),
		maxStatusLen, strings.Repeat("-", statusHeaderWidth),
		"----")

	// Print worktrees
	for _, item := range displayItems {
		headShort := item.head
		if len(headShort) > headDisplayLength {
			headShort = headShort[:headDisplayLength]
		}

		pathDisplay := truncatePath(item.path, pathWidth)
		branchDisplayTrunc := truncatePath(item.branch, maxBranchLen)
		statusDisplayTrunc := truncatePath(item.status, maxStatusLen)

		fmt.Fprintf(w, "%-*s %-*s %-*s %s\n",
			pathWidth, pathDisplay,
			maxBranchLen, branchDisplayTrunc,
			maxStatusLen, statusDisplayTrunc,
			headShort)
	}
}

type listDisplayOptions struct {
	Compact      bool
	MaxPathWidth int
	OutputIsTTY  bool
}

func resolveListDisplayOptions(cmd *cli.Command, w io.Writer) listDisplayOptions {
	maxPathWidth := cmd.Int("max-path-width")
	if maxPathWidth == defaultMaxPathWidth && !cmd.IsSet("max-path-width") {
		if envValue := os.Getenv("WTP_LIST_MAX_PATH"); envValue != "" {
			if parsed, err := strconv.Atoi(envValue); err == nil && parsed > 0 {
				maxPathWidth = parsed
			}
		}
	}
	if maxPathWidth <= 0 {
		maxPathWidth = defaultMaxPathWidth
	}

	compact := cmd.Bool("compact")

	outputIsTTY := false
	if file, ok := w.(*os.File); ok {
		outputIsTTY = term.IsTerminal(int(file.Fd()))
	}

	return listDisplayOptions{
		Compact:      compact,
		MaxPathWidth: maxPathWidth,
		OutputIsTTY:  outputIsTTY,
	}
}
