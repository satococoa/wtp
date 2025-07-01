package main

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/satococoa/wtp/internal/git"
	"github.com/urfave/cli/v3"
)

// NewCompletionCommand creates the completion command definition
func NewCompletionCommand() *cli.Command {
	return &cli.Command{
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
	}
}

// NewShellInitCommand creates the shell-init command definition
func NewShellInitCommand() *cli.Command {
	return &cli.Command{
		Name:   "shell-init",
		Usage:  "Initialize shell completion for current session",
		Action: shellInit,
	}
}

func completionBash(_ context.Context, cmd *cli.Command) error {
	fmt.Println(`#!/bin/bash
# wtp bash completion script
# Add this to your ~/.bashrc or ~/.bash_profile:
# source <(wtp completion bash)

_wtp_completion() {
    local cur prev words cword
    _init_completion || return

    case $cword in
        1)
            COMPREPLY=( $(compgen -W "add remove list init completion shell-init help" -- "$cur") )
            ;;
        2)
            case "${words[1]}" in
                add)
                    # Get branch completions by calling our completion function
                    local branches
                    branches=$(COMP_LINE="$COMP_LINE" COMP_POINT="$COMP_POINT" wtp add --help 2>/dev/null | grep -v "USAGE\|FLAGS\|DESCRIPTION" || echo "")
                    # For now, just complete common branch patterns
                    COMPREPLY=( $(compgen -W "main master develop feature/ bugfix/ hotfix/" -- "$cur") )
                    ;;
                remove)
                    # Complete with existing worktree branches
                    COMPREPLY=( $(compgen -W "main master develop feature/ bugfix/ hotfix/" -- "$cur") )
                    ;;
                completion)
                    COMPREPLY=( $(compgen -W "bash zsh fish" -- "$cur") )
                    ;;
            esac
            ;;
    esac
}

complete -F _wtp_completion wtp`)
	return nil
}

func completionZsh(_ context.Context, cmd *cli.Command) error {
	fmt.Println(`#compdef wtp
# wtp zsh completion script
# Add this to your ~/.zshrc:
# source <(wtp completion zsh)

_wtp() {
    local context state line
    
    case $CURRENT in
        2)
            # First argument - complete commands
            _values 'commands' \
                'add[Create a new worktree]' \
                'remove[Remove a worktree]' \
                'list[List all worktrees]' \
                'init[Initialize configuration file]' \
                'completion[Generate shell completion script]' \
                'shell-init[Initialize shell completion for current session]' \
                'help[Show help]'
            ;;
        3)
            # Second argument - context-dependent completion
            case $words[2] in
                add)
                    _values 'branches' \
                        'main[Main branch]' \
                        'master[Master branch]' \
                        'develop[Develop branch]' \
                        'feature/[Feature branch prefix]' \
                        'bugfix/[Bugfix branch prefix]' \
                        'hotfix/[Hotfix branch prefix]'
                    ;;
                remove)
                    _values 'worktrees' \
                        'main[Main branch]' \
                        'master[Master branch]' \
                        'develop[Develop branch]' \
                        'feature/[Feature branch prefix]' \
                        'bugfix/[Bugfix branch prefix]' \
                        'hotfix/[Hotfix branch prefix]'
                    ;;
                completion)
                    _values 'shells' \
                        'bash[Bash completion]' \
                        'zsh[Zsh completion]' \
                        'fish[Fish completion]'
                    ;;
            esac
            ;;
    esac
}

if [ -n "$ZSH_VERSION" ]; then
    compdef _wtp wtp
fi`)
	return nil
}

func completionFish(_ context.Context, cmd *cli.Command) error {
	// Use the built-in fish completion generation
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
		fmt.Println("source <(wtp completion bash)")
	case "zsh":
		fmt.Println("# Run this command to enable completion for current session:")
		fmt.Println("source <(wtp completion zsh)")
	case "fish":
		fmt.Println("# Run this command to enable completion for current session:")
		fmt.Println("wtp completion fish | source")
	default:
		return fmt.Errorf("unsupported shell: %s", shellName)
	}

	fmt.Println("\n# To make it permanent, add the above command to your shell config file")
	return nil
}

// completeBranches provides branch name completion
func completeBranches(_ context.Context, _ *cli.Command) {
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
