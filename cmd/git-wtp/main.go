package main

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
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
			"functionality with automated setup, branch tracking, and project-specific hooks.\n\n" +
			"Examples:\n" +
			"  git-wtp add feature/new-feature     # Create worktree from branch\n" +
			"  git-wtp add -b hotfix/urgent main   # Create new branch from main\n" +
			"  git-wtp remove feature/old-feature  # Remove worktree\n" +
			"  git-wtp list                        # List all worktrees",
		EnableShellCompletion: true,
		Commands: []*cli.Command{
			{
				Name:          "add",
				Usage:         "Create a new worktree",
				UsageText:     "git-wtp add [--path <path>] [git-worktree-options...] <branch-name> [<commit-ish>]",
				Description: "Creates a new worktree for the specified branch. If the branch doesn't exist locally " +
					"but exists on a remote, it will be automatically tracked. Supports all git worktree flags.\n\n" +
					"Examples:\n" +
					"  git-wtp add feature/auth                    # Auto-generate path: ../worktrees/feature/auth\n" +
					"  git-wtp add --path /tmp/test feature/auth   # Use explicit path\n" +
					"  git-wtp add -b new-feature main             # Create new branch from main\n" +
					"  git-wtp add --detach abc1234                # Detached HEAD at commit",
				ShellComplete: completeBranches,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "path",
						Usage: "Specify explicit path for worktree (instead of auto-generation)",
					},
					&cli.BoolFlag{
						Name:    "force",
						Usage:   "Checkout <commit-ish> even if already checked out in other worktree",
						Aliases: []string{"f"},
					},
					&cli.BoolFlag{
						Name:  "detach",
						Usage: "Make the new worktree's HEAD detached",
					},
					&cli.BoolFlag{
						Name:  "checkout",
						Usage: "Populate the new worktree (default)",
					},
					&cli.BoolFlag{
						Name:  "lock",
						Usage: "Keep the new worktree locked",
					},
					&cli.StringFlag{
						Name:  "reason",
						Usage: "Reason for locking",
					},
					&cli.BoolFlag{
						Name:  "orphan",
						Usage: "Create orphan branch in new worktree",
					},
					&cli.StringFlag{
						Name:    "branch",
						Usage:   "Create new branch",
						Aliases: []string{"b"},
					},
					&cli.StringFlag{
						Name:    "track",
						Usage:   "Set upstream branch",
						Aliases: []string{"t"},
					},
				},
				Action: addCommand,
			},
			{
				Name:   "list",
				Usage:  "List all worktrees",
				Description: "Shows all worktrees with their paths, branches, and HEAD commits.",
				Action: listCommand,
			},
			{
				Name:          "remove",
				Usage:         "Remove a worktree",
				UsageText:     "git-wtp remove <branch-name>",
				Description: "Removes the worktree associated with the specified branch.\n\n" +
					"Examples:\n" +
					"  git-wtp remove feature/old                  # Remove worktree\n" +
					"  git-wtp remove -f feature/dirty             # Force remove dirty worktree\n" +
					"  git-wtp remove --with-branch feature/done   # Also delete the branch",
				ShellComplete: completeWorktrees,
				Flags: []cli.Flag{
					&cli.BoolFlag{
						Name:    "force",
						Usage:   "Force removal even if worktree is dirty",
						Aliases: []string{"f"},
					},
					&cli.BoolFlag{
						Name:  "with-branch",
						Usage: "Also remove the branch after removing worktree",
					},
					&cli.BoolFlag{
						Name:  "force-branch",
						Usage: "Force branch deletion even if not merged (requires --with-branch)",
					},
				},
				Action: removeCommand,
			},
			{
				Name:   "init",
				Usage:  "Initialize configuration file",
				Description: "Creates a .git-worktree-plus.yml configuration file in the repository root " +
					"with example hooks and settings.",
				Action: initCommand,
			},
			{
				Name:   "shell-init",
				Usage:  "Initialize shell completion for current session",
				Action: shellInit,
			},
			{
				Name:  "completion",
				Usage: "Generate shell completion script",
				Commands: []*cli.Command{
					{
						Name:   "bash",
						Usage:  "Generate bash completion script",
						Action: completionBash,
					},
					{
						Name:   "zsh",
						Usage:  "Generate zsh completion script",
						Action: completionZsh,
					},
					{
						Name:   "fish",
						Usage:  "Generate fish completion script",
						Action: completionFish,
					},
				},
			},
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		log.Fatal(err)
	}
}

func addCommand(_ context.Context, cmd *cli.Command) error {
	// Check if we have a branch name from either args or -b flag
	if cmd.Args().Len() == 0 && cmd.String("branch") == "" {
		return fmt.Errorf("branch name is required")
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

	// Resolve worktree path and branch name
	var firstArg string
	if cmd.Args().Len() > 0 {
		firstArg = cmd.Args().Get(0)
	}
	workTreePath, branchName := resolveWorktreePath(cfg, repo.Path(), firstArg, cmd)

	// Build git worktree add command
	args := buildGitWorktreeArgs(cmd, workTreePath, branchName)

	// Execute git worktree add
	if err := repo.ExecuteGitCommand(args...); err != nil {
		return fmt.Errorf("failed to create worktree: %w", err)
	}

	// Display appropriate success message
	if branchName != "" {
		fmt.Printf("Created worktree '%s' at %s\n", branchName, workTreePath)
	} else {
		fmt.Printf("Created worktree at %s\n", workTreePath)
	}

	// Execute post-create hooks
	if cfg.HasHooks() {
		fmt.Println("\nExecuting post-create hooks...")
		executor := hooks.NewExecutor(cfg, repo.Path())
		if err := executor.ExecutePostCreateHooks(workTreePath); err != nil {
			fmt.Fprintf(os.Stderr, "Warning: Hook execution failed: %v\n", err)
		} else {
			fmt.Println("✓ All hooks executed successfully")
		}
	}

	return nil
}

// resolveWorktreePath determines the worktree path and branch name based on flags and arguments
func resolveWorktreePath(
	cfg *config.Config, repoPath, firstArg string, cmd *cli.Command,
) (workTreePath, branchName string) {
	// Check if explicit path is specified via --path flag
	if explicitPath := cmd.String("path"); explicitPath != "" {
		// Explicit path specified - use it as-is, branch name from first argument
		return explicitPath, firstArg
	}

	// No explicit path - generate path automatically from branch name
	branchName = firstArg

	// If -b flag is provided, use that as the branch name for path generation
	if newBranch := cmd.String("branch"); newBranch != "" {
		branchName = newBranch
	}

	// If still no branch name, try to use the first argument
	if branchName == "" && firstArg != "" {
		branchName = firstArg
	}

	workTreePath = cfg.ResolveWorktreePath(repoPath, branchName)
	return workTreePath, branchName
}

func buildGitWorktreeArgs(cmd *cli.Command, workTreePath, branchName string) []string {
	args := []string{"worktree", "add"}

	// Add flags (excluding our custom --path flag)
	if cmd.Bool("force") {
		args = append(args, "--force")
	}
	if cmd.Bool("detach") {
		args = append(args, "--detach")
	}
	if cmd.Bool("checkout") {
		args = append(args, "--checkout")
	}
	if cmd.Bool("lock") {
		args = append(args, "--lock")
	}
	if reason := cmd.String("reason"); reason != "" {
		args = append(args, "--reason", reason)
	}
	if cmd.Bool("orphan") {
		args = append(args, "--orphan")
	}
	if branch := cmd.String("branch"); branch != "" {
		args = append(args, "-b", branch)
	}
	if track := cmd.String("track"); track != "" {
		args = append(args, "--track", track)
	}

	// Add worktree path
	args = append(args, workTreePath)

	// Handle arguments based on whether explicit path was specified
	if cmd.String("path") != "" {
		// Explicit path case: first arg is branch name, add remaining args
		args = append(args, branchName)
		if cmd.Args().Len() > 1 {
			args = append(args, cmd.Args().Slice()[1:]...)
		}
	} else {
		// Auto-generated path case: add branch name if not using -b flag
		if cmd.String("branch") == "" {
			args = append(args, branchName)
		}
		// Add any additional arguments (commit-ish, etc.)
		if cmd.Args().Len() > 1 {
			args = append(args, cmd.Args().Slice()[1:]...)
		}
	}

	return args
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
	branchName := cmd.Args().Get(0)
	force := cmd.Bool("force")
	withBranch := cmd.Bool("with-branch")
	forceBranch := cmd.Bool("force-branch")

	if branchName == "" {
		return fmt.Errorf("branch name is required")
	}

	// Validate flags
	if forceBranch && !withBranch {
		return fmt.Errorf("--force-branch requires --with-branch")
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
	// Use branch name as worktree name
	workTreePath := cfg.ResolveWorktreePath(repo.Path(), branchName)

	// Remove worktree
	if err := repo.RemoveWorktree(workTreePath, force); err != nil {
		return fmt.Errorf("failed to remove worktree: %w", err)
	}

	fmt.Printf("Removed worktree '%s' at %s\n", branchName, workTreePath)

	// Remove branch if requested
	if withBranch {
		// Build git branch delete command
		args := []string{"branch"}
		if forceBranch {
			args = append(args, "-D")
		} else {
			args = append(args, "-d")
		}
		args = append(args, branchName)

		// Execute git branch delete
		if err := repo.ExecuteGitCommand(args...); err != nil {
			// Check if it's an unmerged branch error
			if !forceBranch && strings.Contains(err.Error(), "not fully merged") {
				return fmt.Errorf("branch '%s' is not fully merged. Use --force-branch to delete anyway", branchName)
			}
			return fmt.Errorf("failed to remove branch: %w", err)
		}

		fmt.Printf("Removed branch '%s'\n", branchName)
	}

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

	// Create configuration with comments
	configContent := `# Git Worktree Plus Configuration
version: "1.0"

# Default settings for worktrees
defaults:
  # Base directory for worktrees (relative to repository root)
  base_dir: ../worktrees

# Hooks that run after creating a worktree
hooks:
  post_create:
    # Example: Copy environment file
    - type: copy
      from: .env.example
      to: .env
    
    # Example: Run a command to show all worktrees
    - type: command
      command: git wtp list
    
    # More examples (commented out):
    # - type: command
    #   command: echo "Created new worktree!"
    # - type: command
    #   command: ls -la
    # - type: command
    #   command: npm install
`

	// Write configuration file with comments
	if err := os.WriteFile(configPath, []byte(configContent), 0o600); err != nil {
		return fmt.Errorf("failed to create configuration file: %w", err)
	}

	fmt.Printf("Configuration file created: %s\n", configPath)
	fmt.Println("Edit this file to customize your worktree setup.")
	return nil
}

// Shell completion commands
func completionBash(_ context.Context, _ *cli.Command) error {
	// For bash, we'll use the built-in completion support
	fmt.Println(`#!/bin/bash
# git-wtp bash completion script
# Add this to your ~/.bashrc or ~/.bash_profile:
# source <(git-wtp completion bash)

# Completion for git-wtp command
_git_wtp_completions() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local prev="${COMP_WORDS[COMP_CWORD-1]}"
    
    # Complete command names
    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "$(git-wtp --generate-shell-completion)" -- "$cur") )
        return
    fi
    
    # Complete based on the command
    case "${COMP_WORDS[1]}" in
        add)
            if [[ ${COMP_CWORD} -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "$(git-wtp add --generate-shell-completion)" -- "$cur") )
            fi
            ;;
        remove)
            if [[ ${COMP_CWORD} -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "$(git-wtp remove --generate-shell-completion)" -- "$cur") )
            fi
            ;;
    esac
}

# Register completion for git-wtp
complete -F _git_wtp_completions git-wtp

# For 'git wtp' usage, we recommend using git alias:
# git config --global alias.wtp '!git-wtp'`)
	return nil
}

func completionZsh(_ context.Context, _ *cli.Command) error {
	// For zsh, we'll use the built-in completion support
	fmt.Println(`#compdef git-wtp
# git-wtp zsh completion script
# Add this to your ~/.zshrc:
# source <(git-wtp completion zsh)

# Main completion function
_git_wtp() {
    local context state state_descr line
    typeset -A opt_args

    # First argument is the command
    if (( CURRENT == 2 )); then
        local -a commands
        commands=(${(@f)"$(git-wtp --generate-shell-completion)"})
        _describe 'command' commands
        return
    fi

    # Complete based on the command
    case "${words[2]}" in
        add)
            if (( CURRENT == 3 )); then
                local -a branches
                branches=(${(@f)"$(git-wtp add --generate-shell-completion)"})
                _describe 'branch' branches
            fi
            ;;
        remove)
            if (( CURRENT == 3 )); then
                local -a worktrees
                worktrees=(${(@f)"$(git-wtp remove --generate-shell-completion)"})
                _describe 'worktree' worktrees
            fi
            ;;
        *)
            ;;
    esac
}

# Register for git-wtp command
compdef _git_wtp git-wtp

# For 'git wtp' usage, we recommend using git alias:
# git config --global alias.wtp '!git-wtp'`)
	return nil
}

func completionFish(_ context.Context, cmd *cli.Command) error {
	// For fish, use the built-in method
	fish, err := cmd.Root().ToFishCompletion()
	if err != nil {
		return err
	}
	fmt.Println(fish)
	return nil
}

// shellInit outputs shell initialization commands for the current shell
func shellInit(_ context.Context, _ *cli.Command) error {
	// Detect current shell
	shell := os.Getenv("SHELL")
	if shell == "" {
		return fmt.Errorf("unable to detect shell from $SHELL environment variable")
	}

	// Extract shell name from path
	shellName := filepath.Base(shell)

	switch shellName {
	case "bash":
		fmt.Println("# Run this command to enable completion for current session:")
		fmt.Println("source <(git-wtp completion bash)")
	case "zsh":
		fmt.Println("# Run this command to enable completion for current session:")
		fmt.Println("source <(git-wtp completion zsh)")
	case "fish":
		fmt.Println("# Run this command to enable completion for current session:")
		fmt.Println("git-wtp completion fish | source")
	default:
		return fmt.Errorf("unsupported shell: %s", shellName)
	}

	fmt.Println("\n# To make it permanent, add the above command to your shell config file")
	return nil
}

// completeBranches provides branch name completion
func completeBranches(_ context.Context, cmd *cli.Command) {
	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	// Get all branches using git for-each-ref for better control
	gitCmd := exec.Command("git", "for-each-ref", "--format=%(refname:short)", "refs/heads", "refs/remotes")
	gitCmd.Dir = cwd
	output, err := gitCmd.Output()
	if err != nil {
		return
	}

	branches := strings.Split(strings.TrimSpace(string(output)), "\n")

	// Use a map to avoid duplicates
	seen := make(map[string]bool)
	
	for _, branch := range branches {
		if branch == "" {
			continue
		}
		
		// Skip HEAD references and bare origin
		if branch == "origin/HEAD" || branch == "origin" {
			continue
		}
		
		// Remove remote prefix for display, but keep track of what we've seen
		displayName := branch
		if strings.HasPrefix(branch, "origin/") {
			// For remote branches, show without the origin/ prefix
			displayName = strings.TrimPrefix(branch, "origin/")
		}

		// Skip if already seen (handles case where local and remote have same name)
		if seen[displayName] {
			continue
		}

		seen[displayName] = true
		fmt.Println(displayName)
	}
}

// completeWorktrees provides worktree path completion for remove command
func completeWorktrees(_ context.Context, _ *cli.Command) {
	// Get current directory
	cwd, err := os.Getwd()
	if err != nil {
		return
	}

	// Initialize repository
	repo, err := git.NewRepository(cwd)
	if err != nil {
		return
	}

	// Get all worktrees
	worktrees, err := repo.GetWorktrees()
	if err != nil {
		return
	}

	// Extract branch names from worktrees
	for _, wt := range worktrees {
		if wt.Branch != "" {
			// Branch name is already clean (without refs/heads/)
			fmt.Println(wt.Branch)
		}
	}
}
