package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/satococoa/git-wtp/internal/config"
	"github.com/satococoa/git-wtp/internal/git"
	"github.com/satococoa/git-wtp/internal/hooks"
)

// Version information (set by GoReleaser)
var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

// Display constants
const (
	pathHeaderDashes   = 4
	branchHeaderDashes = 6
	headDisplayLength  = 8
)

func main() {
	versionInfo := fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)

	app := &cli.Command{
		Name:    "git-wtp",
		Usage:   "Git Worktree Plus - Enhanced worktree management",
		Version: versionInfo,
		Description: "A powerful Git worktree management tool that extends git's worktree " +
			"functionality with automated setup, branch tracking, and project-specific hooks.",
		Commands: []*cli.Command{
			{
				Name:      "add",
				Usage:     "Create a new worktree",
				UsageText: "git-wtp add <name> [branch] or git-wtp add <name> -b <branch>",
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:    "branch",
						Usage:   "Branch to checkout in the new worktree",
						Aliases: []string{"b"},
					},
				},
				Action: addCommand,
			},
			{
				Name:   "list",
				Usage:  "List all worktrees",
				Action: listCommand,
			},
			{
				Name:      "remove",
				Usage:     "Remove a worktree",
				UsageText: "git-wtp remove <name>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "force",
						Usage:   "Force removal even if worktree is dirty",
						Aliases: []string{"f"},
					},
				},
				Action: removeCommand,
			},
			{
				Name:   "init",
				Usage:  "Initialize configuration file",
				Action: initCommand,
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func addCommand(_ context.Context, cmd *cli.Command) error {
	worktreeName := cmd.Args().Get(0)
	branchName := cmd.Args().Get(1)

	if worktreeName == "" {
		return fmt.Errorf("worktree name is required")
	}

	// Get current working directory (should be a git repository)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Initialize repository
	repo, err := git.NewRepository(cwd)
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Load configuration
	cfg, err := config.LoadConfig(repo.Path())
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Resolve worktree path using configuration
	workTreePath := cfg.ResolveWorktreePath(repo.Path(), worktreeName)

	// Check if branch is specified via flag
	if flagBranch := cmd.String("branch"); flagBranch != "" {
		branchName = flagBranch
	}

	// If no branch specified, use worktree name as branch name
	if branchName == "" {
		branchName = worktreeName
	}

	// Create worktree with automatic remote tracking
	if err := repo.CreateWorktreeFromBranch(workTreePath, branchName); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	fmt.Printf("Created worktree '%s' at %s", worktreeName, workTreePath)
	if branchName != "" {
		fmt.Printf(" on branch %s", branchName)
	}
	fmt.Println()

	// Execute post-create hooks
	if cfg.HasHooks() {
		fmt.Println("Executing post-create hooks...")
		executor := hooks.NewExecutor(cfg, repo.Path())
		if err := executor.ExecutePostCreateHooks(workTreePath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Hook execution failed: %v\n", err)
		}
	}

	return nil
}

func listCommand(_ context.Context, _ *cli.Command) error {
	// Get current working directory (should be a git repository)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Initialize repository
	repo, err := git.NewRepository(cwd)
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Get worktrees
	worktrees, err := repo.GetWorktrees()
	if err != nil {
		return fmt.Errorf("failed to list worktrees: %w", err)
	}

	if len(worktrees) == 0 {
		fmt.Println("No worktrees found")
		return nil
	}

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
	fmt.Printf("%-*s %-*s %s\n", maxPathLen, "PATH", maxBranchLen, "BRANCH", "HEAD")
	fmt.Printf("%-*s %-*s %s\n",
		maxPathLen, strings.Repeat("-", pathHeaderDashes),
		maxBranchLen, strings.Repeat("-", branchHeaderDashes),
		"----")

	// Print worktrees
	for _, wt := range worktrees {
		headShort := wt.HEAD
		if len(headShort) > headDisplayLength {
			headShort = headShort[:headDisplayLength]
		}
		fmt.Printf("%-*s %-*s %s\n", maxPathLen, wt.Path, maxBranchLen, wt.Branch, headShort)
	}

	return nil
}

func removeCommand(_ context.Context, cmd *cli.Command) error {
	worktreeName := cmd.Args().Get(0)
	force := cmd.Bool("force")

	if worktreeName == "" {
		return fmt.Errorf("worktree name is required")
	}

	// Get current working directory (should be a git repository)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Initialize repository
	repo, err := git.NewRepository(cwd)
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Load configuration
	cfg, err := config.LoadConfig(repo.Path())
	if err != nil {
		return fmt.Errorf("failed to load configuration: %w", err)
	}

	// Resolve worktree path using configuration
	workTreePath := cfg.ResolveWorktreePath(repo.Path(), worktreeName)

	// Remove worktree
	if err := repo.RemoveWorktree(workTreePath, force); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	fmt.Printf("Removed worktree '%s' at %s\n", worktreeName, workTreePath)
	return nil
}

func initCommand(_ context.Context, _ *cli.Command) error {
	// Get current working directory (should be a git repository)
	cwd, err := os.Getwd()
	if err != nil {
		return fmt.Errorf("failed to get current directory: %w", err)
	}

	// Initialize repository
	repo, err := git.NewRepository(cwd)
	if err != nil {
		return fmt.Errorf("not in a git repository: %w", err)
	}

	// Check if config file already exists
	configPath := fmt.Sprintf("%s/%s", repo.Path(), config.ConfigFileName)
	if _, err := os.Stat(configPath); err == nil {
		return fmt.Errorf("configuration file already exists: %s", configPath)
	}

	// Create default configuration
	defaultConfig := &config.Config{
		Version: config.CurrentVersion,
		Defaults: config.Defaults{
			BaseDir: "../worktrees",
		},
		Hooks: config.Hooks{
			PostCreate: []config.Hook{
				{
					Type: config.HookTypeCopy,
					From: ".env.example",
					To:   ".env",
				},
			},
		},
	}

	// Save configuration
	if err := config.SaveConfig(repo.Path(), defaultConfig); err != nil {
		return fmt.Errorf("failed to create configuration file: %w", err)
	}

	fmt.Printf("Configuration file created: %s\n", configPath)
	fmt.Println("Edit this file to customize your worktree setup.")
	return nil
}
