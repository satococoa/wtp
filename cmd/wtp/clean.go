package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"

	"github.com/charmbracelet/huh"
	"github.com/urfave/cli/v3"

	"github.com/satococoa/wtp/v2/internal/command"
	"github.com/satococoa/wtp/v2/internal/config"
	"github.com/satococoa/wtp/v2/internal/errors"
	"github.com/satococoa/wtp/v2/internal/git"
)

// worktreeCleanStatus holds the validation results for a worktree
type worktreeCleanStatus struct {
	worktree git.Worktree
	isMerged bool
	isClean  bool
	isPushed bool
	isSafe   bool
	reason   string
	reasons  []string
}

// Variable to allow mocking in tests
var cleanGetwd = os.Getwd

// NewCleanCommand creates the clean command definition
func NewCleanCommand() *cli.Command {
	return &cli.Command{
		Name:  "clean",
		Usage: "Interactively clean up worktrees",
		Description: "Shows an interactive checklist of worktrees that can be safely removed.\n\n" +
			"Worktrees are pre-selected if they are:\n" +
			"  • Fully merged into the main branch\n" +
			"  • Have no uncommitted changes\n" +
			"  • Have no unpushed commits\n\n" +
			"Examples:\n" +
			"  wtp clean                 # Show interactive clean UI\n" +
			"  wtp clean --force         # Force remove even with uncommitted changes",
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Usage:   "Force removal even if worktree is dirty",
				Aliases: []string{"f"},
			},
		},
		Action: cleanCommand,
	}
}

func cleanCommand(_ context.Context, cmd *cli.Command) error {
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	force := cmd.Bool("force")

	cwd, err := cleanGetwd()
	if err != nil {
		return errors.DirectoryAccessFailed("access current", ".", err)
	}

	_, err = git.NewRepository(cwd)
	if err != nil {
		return errors.NotInGitRepository()
	}

	executor := command.NewRealExecutor()
	return cleanCommandWithExecutor(w, executor, force)
}

func cleanCommandWithExecutor(
	w io.Writer,
	executor command.Executor,
	force bool,
) error {
	worktrees, mainWorktreePath, err := getWorktreesForClean(executor)
	if err != nil {
		return err
	}

	cfg := loadCleanConfig(mainWorktreePath)
	managedWorktrees := filterManagedWorktrees(worktrees, cfg, mainWorktreePath)

	if len(managedWorktrees) == 0 {
		_, ferr := fmt.Fprintln(w, "No managed worktrees found")
		return ferr
	}

	statuses := validateWorktrees(managedWorktrees, executor)
	opts := buildCleanOptions(statuses, cfg, mainWorktreePath)

	selected, err := runCleanForm(opts)
	if err != nil {
		return err
	}

	if len(selected) == 0 {
		_, ferr := fmt.Fprintln(w, "No worktrees selected for removal")
		return ferr
	}

	_, err = fmt.Fprintf(w, "\nRemoving %d worktree(s)...\n", len(selected))
	if err != nil {
		return err
	}

	removeSelectedWorktrees(w, executor, managedWorktrees, selected, cfg, mainWorktreePath, force)
	return nil
}

func getWorktreesForClean(executor command.Executor) ([]git.Worktree, string, error) {
	listCmd := command.GitWorktreeList()
	result, err := executor.Execute([]command.Command{listCmd})
	if err != nil {
		return nil, "", errors.GitCommandFailed("git worktree list", err.Error())
	}
	if result == nil || len(result.Results) == 0 {
		return nil, "", errors.GitCommandFailed("git worktree list", "no output")
	}

	worktrees := parseWorktreesFromOutput(result.Results[0].Output)

	mainWorktreePath := ""
	for _, wt := range worktrees {
		if wt.IsMain {
			mainWorktreePath = wt.Path
			break
		}
	}

	return worktrees, mainWorktreePath, nil
}

func loadCleanConfig(mainWorktreePath string) *config.Config {
	cfg, err := config.LoadConfig(mainWorktreePath)
	if err != nil {
		cfg = &config.Config{
			Defaults: config.Defaults{
				BaseDir: config.DefaultBaseDir,
			},
		}
	}
	return cfg
}

func filterManagedWorktrees(
	worktrees []git.Worktree,
	cfg *config.Config,
	mainWorktreePath string,
) []git.Worktree {
	var managed []git.Worktree
	for _, wt := range worktrees {
		if wt.IsMain {
			continue
		}
		if !isWorktreeManagedCommon(wt.Path, cfg, mainWorktreePath, wt.IsMain) {
			continue
		}
		managed = append(managed, wt)
	}
	return managed
}

func validateWorktrees(
	worktrees []git.Worktree,
	executor command.Executor,
) []worktreeCleanStatus {
	mainBranch := detectMainBranch(executor)

	statuses := make([]worktreeCleanStatus, len(worktrees))
	var wg sync.WaitGroup
	for i, wt := range worktrees {
		wg.Add(1)
		go func(idx int, w git.Worktree) {
			defer wg.Done()
			statuses[idx] = validateWorktree(w, executor, mainBranch)
		}(i, wt)
	}
	wg.Wait()
	return statuses
}

func runCleanForm(opts cleanOptions) ([]string, error) {
	var selected []string
	form := huh.NewForm(
		huh.NewGroup(
			huh.NewMultiSelect[string]().
				Title("Select worktrees to remove:").
				Description(opts.columnHeader).
				Options(opts.options...).
				Value(&selected),
		),
	)

	if err := form.Run(); err != nil {
		return nil, err
	}

	return selected, nil
}

func removeSelectedWorktrees(
	w io.Writer,
	executor command.Executor,
	worktrees []git.Worktree,
	selected []string,
	cfg *config.Config,
	mainWorktreePath string,
	force bool,
) {
	for _, name := range selected {
		wt, findErr := findWorktreeByName(worktrees, name, cfg, mainWorktreePath)
		if findErr != nil {
			_, _ = fmt.Fprintf(w, "Failed to find worktree '%s': %v\n", name, findErr)
			continue
		}

		removeCmd := command.GitWorktreeRemove(wt.Path, force)
		result, execErr := executor.Execute([]command.Command{removeCmd})
		if execErr != nil {
			_, _ = fmt.Fprintf(w, "Failed to remove '%s': %v\n", name, execErr)
			continue
		}
		if len(result.Results) > 0 && result.Results[0].Error != nil {
			_, _ = fmt.Fprintf(w, "Failed to remove '%s': %v\n", name, result.Results[0].Error)
			continue
		}

		_, _ = fmt.Fprintf(w, "✓ Removed '%s'\n", name)
	}
}

func validateWorktree(wt git.Worktree, executor command.Executor, mainBranch string) worktreeCleanStatus {
	status := worktreeCleanStatus{
		worktree: wt,
		isMerged: true,
		isClean:  true,
		isPushed: true,
		isSafe:   true,
	}

	status.checkMergeStatus(wt, mainBranch, executor)
	status.checkCleanStatus(wt, executor)
	status.checkPushStatus(wt, mainBranch, executor)
	status.buildReason()

	return status
}

func detectMainBranch(executor command.Executor) string {
	mainCmd := command.Command{
		Name: "git",
		Args: []string{"rev-parse", "--abbrev-ref", "main"},
	}
	result, err := executor.Execute([]command.Command{mainCmd})
	if err == nil && len(result.Results) > 0 {
		branch := strings.TrimSpace(result.Results[0].Output)
		if branch != "" && branch != "main" {
			return branch
		}
	}

	masterCmd := command.Command{
		Name: "git",
		Args: []string{"rev-parse", "--verify", "--quiet", "master"},
	}
	result, err = executor.Execute([]command.Command{masterCmd})
	if err == nil && len(result.Results) > 0 && result.Results[0].Error == nil {
		return "master"
	}

	return "main"
}

func (s *worktreeCleanStatus) checkMergeStatus(
	wt git.Worktree,
	mainBranch string,
	executor command.Executor,
) {
	if wt.Branch == "" || wt.Branch == "detached" {
		s.isMerged = false
		s.isSafe = false
		s.reasons = append(s.reasons, "detached HEAD")
		return
	}

	mergeBaseCmd := command.Command{
		Name: "git",
		Args: []string{"merge-base", "--is-ancestor", wt.Branch, mainBranch},
	}
	result, err := executor.Execute([]command.Command{mergeBaseCmd})
	if err != nil || result == nil || len(result.Results) == 0 || result.Results[0].Error != nil {
		s.isMerged = false
		s.isSafe = false
		s.reasons = append(s.reasons, "unmerged")
	}
}

func (s *worktreeCleanStatus) checkCleanStatus(wt git.Worktree, executor command.Executor) {
	statusCmd := command.Command{
		Name:    "git",
		Args:    []string{"status", "--porcelain"},
		WorkDir: wt.Path,
	}
	result, err := executor.Execute([]command.Command{statusCmd})
	if err != nil || result == nil || len(result.Results) == 0 {
		return
	}

	output := strings.TrimSpace(result.Results[0].Output)
	if output != "" {
		s.isClean = false
		s.isSafe = false
		s.reasons = append(s.reasons, "uncommitted changes")
	}
}

func (s *worktreeCleanStatus) checkPushStatus(
	wt git.Worktree,
	mainBranch string,
	executor command.Executor,
) {
	if wt.Branch == "" || wt.Branch == "detached" {
		return
	}

	pushCheckCmd := command.Command{
		Name: "git",
		Args: []string{"rev-list", "--count", fmt.Sprintf("origin/%s..%s", wt.Branch, wt.Branch)},
	}
	result, err := executor.Execute([]command.Command{pushCheckCmd})
	if err != nil || result == nil || len(result.Results) == 0 {
		return
	}

	count := strings.TrimSpace(result.Results[0].Output)
	if count == "0" || count == "" {
		return
	}

	s.isPushed = false
	s.isSafe = false

	aheadCmd := command.Command{
		Name: "git",
		Args: []string{"rev-list", "--count", fmt.Sprintf("%s..%s", mainBranch, wt.Branch)},
	}
	result, err = executor.Execute([]command.Command{aheadCmd})

	aheadCount := ""
	if err == nil && result != nil && len(result.Results) > 0 {
		aheadCount = strings.TrimSpace(result.Results[0].Output)
	}

	if aheadCount != "" && aheadCount != "0" {
		s.reasons = append(s.reasons, fmt.Sprintf("unpushed commits (%s ahead)", aheadCount))
	} else {
		s.reasons = append(s.reasons, "unpushed commits")
	}
}

func (s *worktreeCleanStatus) buildReason() {
	if s.isSafe {
		s.reason = "safe: merged, clean, pushed"
	} else {
		s.reason = fmt.Sprintf("unsafe: %s", strings.Join(s.reasons, ", "))
	}
}

type cleanOptions struct {
	options      []huh.Option[string]
	columnHeader string
}

func buildCleanOptions(statuses []worktreeCleanStatus, cfg *config.Config, mainRepoPath string) cleanOptions {
	maxNameLen := len("WORKTREE")
	names := make([]string, len(statuses))
	for i, status := range statuses {
		names[i] = getWorktreeNameFromPath(status.worktree.Path, cfg, mainRepoPath, status.worktree.IsMain)
		if len(names[i]) > maxNameLen {
			maxNameLen = len(names[i])
		}
	}

	options := make([]huh.Option[string], 0, len(statuses))
	for i, status := range statuses {
		statusText := "safe"
		note := "merged, clean, pushed"
		if !status.isSafe {
			statusText = "unsafe"
			note = strings.Join(status.reasons, ", ")
		}

		label := fmt.Sprintf("%-*s  %-6s  %s", maxNameLen, names[i], statusText, note)
		option := huh.NewOption(label, names[i])
		if status.isSafe {
			option = option.Selected(true)
		}
		options = append(options, option)
	}

	columnHeader := fmt.Sprintf("    %-*s  %-6s  %s", maxNameLen, "WORKTREE", "STATUS", "NOTE")
	return cleanOptions{
		options:      options,
		columnHeader: columnHeader,
	}
}

func findWorktreeByName(
	worktrees []git.Worktree,
	name string,
	cfg *config.Config,
	mainRepoPath string,
) (*git.Worktree, error) {
	for _, wt := range worktrees {
		displayName := getWorktreeNameFromPath(wt.Path, cfg, mainRepoPath, wt.IsMain)
		if displayName == name {
			return &wt, nil
		}
	}
	return nil, fmt.Errorf("worktree not found: %s", name)
}
