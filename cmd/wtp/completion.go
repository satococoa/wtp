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

func completionBash(_ context.Context, _ *cli.Command) error {
	// For bash, we'll use the built-in completion support
	fmt.Println(`#!/bin/bash
# wtp bash completion script
# Add this to your ~/.bashrc or ~/.bash_profile:
# source <(wtp completion bash)

# Completion for wtp command
_wtp_completions() {
    local cur="${COMP_WORDS[COMP_CWORD]}"
    local prev="${COMP_WORDS[COMP_CWORD-1]}"
    
    # Complete command names
    if [[ ${COMP_CWORD} -eq 1 ]]; then
        COMPREPLY=( $(compgen -W "$(wtp --generate-shell-completion)" -- "$cur") )
        return
    fi
    
    # Complete based on the command
    case "${COMP_WORDS[1]}" in
        add)
            if [[ ${COMP_CWORD} -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "$(wtp add --generate-shell-completion)" -- "$cur") )
            fi
            ;;
        remove)
            if [[ ${COMP_CWORD} -eq 2 ]]; then
                COMPREPLY=( $(compgen -W "$(wtp remove --generate-shell-completion)" -- "$cur") )
            fi
            ;;
    esac
}

# Register completion for wtp
complete -F _wtp_completions wtp

# For 'git wtp' usage, you can create a git alias:
# git config --global alias.wtp '!wtp'`)
	return nil
}

func completionZsh(_ context.Context, _ *cli.Command) error {
	// For zsh, we'll use the built-in completion support
	fmt.Println(`#compdef wtp
# wtp zsh completion script
# Add this to your ~/.zshrc:
# source <(wtp completion zsh)

# Main completion function
_wtp() {
    local context state state_descr line
    typeset -A opt_args

    # First argument is the command
    if (( CURRENT == 2 )); then
        local -a commands
        commands=(${(@f)"$(wtp --generate-shell-completion)"})
        _describe 'command' commands
        return
    fi

    # Complete based on the command
    case "${words[2]}" in
        add)
            if (( CURRENT == 3 )); then
                local -a branches
                branches=(${(@f)"$(wtp add --generate-shell-completion)"})
                _describe 'branch' branches
            fi
            ;;
        remove)
            if (( CURRENT == 3 )); then
                local -a worktrees
                worktrees=(${(@f)"$(wtp remove --generate-shell-completion)"})
                _describe 'worktree' worktrees
            fi
            ;;
        *)
            ;;
    esac
}

# Register for wtp command
compdef _wtp wtp

# For 'git wtp' usage, you can create a git alias:
# git config --global alias.wtp '!wtp'`)
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
