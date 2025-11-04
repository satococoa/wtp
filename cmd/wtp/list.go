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

	"github.com/satococoa/wtp/v2/internal/command"
	"github.com/satococoa/wtp/v2/internal/config"
	"github.com/satococoa/wtp/v2/internal/errors"
	"github.com/satococoa/wtp/v2/internal/git"
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
	pathPadding         = 2
	minPathWidth        = 20
	columnSpacing       = 3
	columnSpacingSlots  = 3
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
	if termWidth <= 0 {
		termWidth = 80
	}

	items, metrics := collectListDisplayData(worktrees, currentPath, cfg, mainRepoPath)
	if len(items) == 0 {
		return
	}

	pathWidth, branchWidth, statusWidth := computeListColumnWidths(metrics, termWidth, opts)

	fmt.Fprintf(w, "%-*s %-*s %-*s %s\n", pathWidth, "PATH", branchWidth, "BRANCH", statusWidth, "STATUS", "HEAD")
	fmt.Fprintf(w, "%-*s %-*s %-*s %s\n",
		pathWidth, strings.Repeat("-", pathHeaderDashes),
		branchWidth, strings.Repeat("-", branchHeaderDashes),
		statusWidth, strings.Repeat("-", len("STATUS")),
		"----")

	for _, item := range items {
		headShort := item.head
		if len(headShort) > headDisplayLength {
			headShort = headShort[:headDisplayLength]
		}

		fmt.Fprintf(w, "%-*s %-*s %-*s %s\n",
			pathWidth, truncatePath(item.path, pathWidth),
			branchWidth, truncatePath(item.branch, branchWidth),
			statusWidth, truncatePath(item.status, statusWidth),
			headShort)
	}
}

type listDisplayData struct {
	path   string
	branch string
	head   string
	status string
}

type listColumnMetrics struct {
	maxPathLen   int
	maxBranchLen int
	maxStatusLen int
}

func collectListDisplayData(
	worktrees []git.Worktree, currentPath string, cfg *config.Config, mainRepoPath string,
) ([]listDisplayData, listColumnMetrics) {
	metrics := listColumnMetrics{
		maxPathLen:   len("PATH"),
		maxBranchLen: len("BRANCH"),
		maxStatusLen: len("STATUS"),
	}

	items := make([]listDisplayData, 0, len(worktrees))

	for _, wt := range worktrees {
		pathDisplay := getWorktreeDisplayName(wt, cfg, mainRepoPath)
		if wt.Path == currentPath {
			pathDisplay += "*"
		}

		branchDisplay := formatBranchDisplay(wt.Branch)

		statusDisplay := "unmanaged"
		if isWorktreeManagedList(wt.Path, cfg, mainRepoPath, wt.IsMain) {
			statusDisplay = "managed"
		}

		if len(pathDisplay) > metrics.maxPathLen {
			metrics.maxPathLen = len(pathDisplay)
		}
		if len(branchDisplay) > metrics.maxBranchLen {
			metrics.maxBranchLen = len(branchDisplay)
		}
		if len(statusDisplay) > metrics.maxStatusLen {
			metrics.maxStatusLen = len(statusDisplay)
		}

		items = append(items, listDisplayData{
			path:   pathDisplay,
			branch: branchDisplay,
			head:   wt.HEAD,
			status: statusDisplay,
		})
	}

	return items, metrics
}

func computeListColumnWidths(
	metrics listColumnMetrics,
	termWidth int,
	opts listDisplayOptions,
) (pathWidth, branchWidth, statusWidth int) {
	branchWidth, statusWidth = clampBranchAndStatusWidths(metrics.maxBranchLen, metrics.maxStatusLen, termWidth)
	pathWidth = derivePathWidth(metrics.maxPathLen, branchWidth, statusWidth, termWidth, opts)
	return pathWidth, branchWidth, statusWidth
}

func clampBranchAndStatusWidths(
	maxBranchLen, maxStatusLen, termWidth int,
) (branchWidth, statusWidth int) {
	branchWidth = maxBranchLen
	statusWidth = maxStatusLen

	branchHeaderWidth := len("BRANCH")
	statusHeaderWidth := len("STATUS")

	if branchWidth < branchHeaderWidth {
		branchWidth = branchHeaderWidth
	}
	if statusWidth < statusHeaderWidth {
		statusWidth = statusHeaderWidth
	}

	spacingTotal := columnSpacing * columnSpacingSlots
	maxAvailableForBranch := termWidth - minPathWidth - statusWidth - spacingTotal - headDisplayLength
	if branchWidth > maxAvailableForBranch {
		branchWidth = maxAvailableForBranch
		if branchWidth < branchHeaderWidth {
			branchWidth = branchHeaderWidth
		}
	}

	return branchWidth, statusWidth
}

func derivePathWidth(maxPathLen, branchWidth, statusWidth, termWidth int, opts listDisplayOptions) int {
	pathHeaderWidth := len("PATH")
	availableForPath := termWidth - columnSpacing - branchWidth - columnSpacing - statusWidth -
		columnSpacing - headDisplayLength
	availableForPath = max(availableForPath, pathHeaderWidth)

	pathWidth := availableForPath

	if opts.Compact {
		pathWidth = clampCompactPathWidth(pathWidth, maxPathLen)
	} else {
		pathWidth = clampExpandedPathWidth(pathWidth, maxPathLen)
	}

	if opts.MaxPathWidth > 0 {
		pathWidth = min(pathWidth, opts.MaxPathWidth)
	}

	pathWidth = max(min(pathWidth, availableForPath), pathHeaderWidth)

	return pathWidth
}

func clampCompactPathWidth(currentWidth, maxPathLen int) int {
	pathHeaderWidth := len("PATH")
	minCompactWidth := max(maxPathLen, pathHeaderWidth)
	return max(min(currentWidth, minCompactWidth), pathHeaderWidth)
}

func clampExpandedPathWidth(currentWidth, maxPathLen int) int {
	pathHeaderWidth := len("PATH")

	desiredWidth := max(maxPathLen+pathPadding, minPathWidth, pathHeaderWidth)
	return max(min(currentWidth, desiredWidth), minPathWidth)
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
