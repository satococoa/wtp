package main

import (
	"context"
	"fmt"
	"log"
	"os"

	"github.com/satococoa/git-wtp/internal/config"
	"github.com/satococoa/git-wtp/internal/git"
	"github.com/satococoa/git-wtp/internal/hooks"
	"github.com/urfave/cli/v3"
)

func main() {
	app := &cli.Command{
		Name:  "git-wtp",
		Usage: "Git Worktree Plus - Enhanced worktree management",
		Description: "A powerful Git worktree management tool that extends git's worktree " +
			"functionality with automated setup, branch tracking, and project-specific hooks.",
		Commands: []*cli.Command{
			{
				Name:  "add",
				Usage: "Create a new worktree",
				UsageText: "git-wtp add <name> [branch]",
				Action: addCommand,
			},
			{
				Name:  "list",
				Usage: "List all worktrees",
				Action: listCommand,
			},
			{
				Name:  "remove",
				Usage: "Remove a worktree",
				UsageText: "git-wtp remove <name>",
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:  "force",
						Usage: "Force removal even if worktree is dirty",
						Aliases: []string{"f"},
					},
				},
				Action: removeCommand,
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func addCommand(ctx context.Context, cmd *cli.Command) error {
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

func listCommand(ctx context.Context, cmd *cli.Command) error {
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

	fmt.Printf("%-30s %-20s %s\n", "PATH", "BRANCH", "HEAD")
	fmt.Printf("%-30s %-20s %s\n", "----", "------", "----")
	for _, wt := range worktrees {
		fmt.Printf("%-30s %-20s %s\n", wt.Path, wt.Branch, wt.HEAD[:8])
	}

	return nil
}

func removeCommand(ctx context.Context, cmd *cli.Command) error {
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