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
		Description: "Generate shell completion scripts for bash, zsh, or fish. " +
			"The generated scripts provide comprehensive completion for commands, flags, " +
			"branch names, worktree names, and flag values.",
		Commands: []*cli.Command{
			{
				Name:        "bash",
				Usage:       "Generate bash completion script",
				Description: "Generate bash completion script with full flag and option support",
				Action:      completionBash,
			},
			{
				Name:        "zsh",
				Usage:       "Generate zsh completion script",
				Description: "Generate zsh completion script with full flag and option support",
				Action:      completionZsh,
			},
			{
				Name:        "fish",
				Usage:       "Generate fish completion script",
				Description: "Generate fish completion script using urfave/cli built-in support",
				Action:      completionFish,
			},
			{
				Name:   "__branches",
				Hidden: true,
				Usage:  "List branches for completion",
				Action: func(_ context.Context, _ *cli.Command) error {
					printBranches()
					return nil
				},
			},
			{
				Name:   "__worktrees",
				Hidden: true,
				Usage:  "List worktrees for completion",
				Action: func(_ context.Context, _ *cli.Command) error {
					printWorktrees()
					return nil
				},
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
	fmt.Println(`#!/bin/bash
# wtp bash completion script
# Add this to your ~/.bashrc or ~/.bash_profile:
# source <(wtp completion bash)

_wtp_completion() {
    local cur prev words cword
    
    # Use _init_completion if available, otherwise manual setup
    if declare -F _init_completion >/dev/null 2>&1; then
        _init_completion || return
    else
        # Manual completion setup for broader compatibility
        COMPREPLY=()
        cur="${COMP_WORDS[COMP_CWORD]}"
        prev="${COMP_WORDS[COMP_CWORD-1]}"
        words=("${COMP_WORDS[@]}")
        cword=$COMP_CWORD
    fi

    # Handle flag completion for all commands
    if [[ $cur == -* ]]; then
        case "${words[1]}" in
            add)
                local add_flags="--path --force -f --detach --checkout --lock --reason --orphan"
                add_flags="$add_flags --branch -b --track -t --help -h"
                COMPREPLY=( $(compgen -W "$add_flags" -- "$cur") )
                ;;
            remove)
                COMPREPLY=( $(compgen -W "--force -f --with-branch --force-branch --help -h" -- "$cur") )
                ;;
            list)
                COMPREPLY=( $(compgen -W "--help -h" -- "$cur") )
                ;;
            init)
                COMPREPLY=( $(compgen -W "--help -h" -- "$cur") )
                ;;
            completion)
                COMPREPLY=( $(compgen -W "--help -h" -- "$cur") )
                ;;
            shell-init)
                COMPREPLY=( $(compgen -W "--help -h" -- "$cur") )
                ;;
            *)
                COMPREPLY=( $(compgen -W "--help -h --version" -- "$cur") )
                ;;
        esac
        return
    fi

    # Handle value completion for flags that require arguments
    case "$prev" in
        --path)
            # Complete with directories
            COMPREPLY=( $(compgen -d -- "$cur") )
            return
            ;;
        --reason)
            # Complete with common lock reasons
            COMPREPLY=( $(compgen -W "testing debugging maintenance" -- "$cur") )
            return
            ;;
        --branch|-b)
            # Complete with branch names for new branch creation
            local branches
            branches=$(wtp completion __branches 2>/dev/null)
            COMPREPLY=( $(compgen -W "$branches" -- "$cur") )
            return
            ;;
        --track|-t)
            # Complete with remote branches for tracking
            local remote_branches
            local git_cmd="git for-each-ref --format='%(refname:short)' refs/remotes 2>/dev/null"
            remote_branches=$($git_cmd | grep -v '/HEAD$')
            COMPREPLY=( $(compgen -W "$remote_branches" -- "$cur") )
            return
            ;;
    esac

    case $cword in
        1)
            COMPREPLY=( $(compgen -W "add remove list init completion shell-init help" -- "$cur") )
            ;;
        *)
            case "${words[1]}" in
                add)
                    # For add command, determine what kind of completion is needed
                    local has_branch_flag=false
                    local has_path_flag=false
                    local branch_value_provided=false
                    local i
                    
                    # Parse previous words to understand current context
                    for ((i=2; i<cword; i++)); do
                        case "${words[i]}" in
                            --branch|-b)
                                has_branch_flag=true
                                if [[ $((i+1)) -lt $cword ]]; then
                                    branch_value_provided=true
                                    ((i++)) # Skip the branch name value
                                fi
                                ;;
                            --path)
                                has_path_flag=true
                                if [[ $((i+1)) -lt $cword ]]; then
                                    ((i++)) # Skip the path value
                                fi
                                ;;
                        esac
                    done
                    
                    # If we're immediately after -b/--branch, complete with branch names
                    if [[ "$prev" == "-b" || "$prev" == "--branch" ]]; then
                        local branches
                        branches=$(wtp completion __branches 2>/dev/null)
                        COMPREPLY=( $(compgen -W "$branches" -- "$cur") )
                        return
                    fi
                    
                    # Count non-flag arguments to determine if we should complete
                    local arg_count=0
                    for ((i=2; i<cword; i++)); do
                        if [[ "${words[i]}" != -* ]]; then
                            # Skip values that follow flags
                            local prev_word="${words[i-1]}"
                            if [[ "$prev_word" != "-b" && "$prev_word" != "--branch" && 
                                  "$prev_word" != "--path" && "$prev_word" != "--reason" && 
                                  "$prev_word" != "-t" && "$prev_word" != "--track" ]]; then
                                ((arg_count++))
                            fi
                        fi
                    done
                    
                    # If -b flag was used and value provided, complete with commit-ish (max 1 arg)
                    if [[ $has_branch_flag == true && $branch_value_provided == true ]]; then
                        if [[ $arg_count -eq 0 ]]; then
                            # Complete with branch names as potential commit-ish
                            local branches
                            branches=$(wtp completion __branches 2>/dev/null)
                            COMPREPLY=( $(compgen -W "$branches" -- "$cur") )
                        else
                            # No more completions needed
                            COMPREPLY=()
                        fi
                    else
                        # Normal case: complete with branch names (max 1 arg)
                        if [[ $arg_count -eq 0 ]]; then
                            local branches
                            branches=$(wtp completion __branches 2>/dev/null)
                            COMPREPLY=( $(compgen -W "$branches" -- "$cur") )
                        else
                            # No more completions needed
                            COMPREPLY=()
                        fi
                    fi
                    ;;
                remove)
                    # Get actual worktree branches dynamically
                    local worktrees
                    worktrees=$(wtp completion __worktrees 2>/dev/null)
                    COMPREPLY=( $(compgen -W "$worktrees" -- "$cur") )
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

func completionZsh(_ context.Context, _ *cli.Command) error {
	fmt.Println(`#compdef wtp
# wtp zsh completion script
# Add this to your ~/.zshrc:
# source <(wtp completion zsh)

_wtp() {
    local context state line
    typeset -A opt_args
    
    _arguments -C \
        '1: :_wtp_commands' \
        '*:: :->args'
    
    case $state in
        args)
            case $words[1] in
                add)
                    _arguments -s \
                        '--path[Specify explicit path for worktree]:path:_directories' \
                        '(--force -f)'{--force,-f}'[Checkout even if already checked out in other worktree]' \
                        '--detach[Make the new worktree HEAD detached]' \
                        '--checkout[Populate the new worktree (default)]' \
                        '--lock[Keep the new worktree locked]' \
                        '--reason[Reason for locking]:reason:(testing debugging maintenance)' \
                        '--orphan[Create orphan branch in new worktree]' \
                        '(--branch -b)'{--branch,-b}'[Create new branch]:branch:_wtp_branches' \
                        '(--track -t)'{--track,-t}'[Set upstream branch]:upstream:_wtp_remote_branches' \
                        '(--help -h)'{--help,-h}'[Show help]' \
                        '*: :_wtp_branches_or_commits'
                    ;;
                remove)
                    _arguments -s \
                        '(--force -f)'{--force,-f}'[Force removal even if worktree is dirty]' \
                        '--with-branch[Also remove the branch after removing worktree]' \
                        '--force-branch[Force branch deletion even if not merged (requires --with-branch)]' \
                        '(--help -h)'{--help,-h}'[Show help]' \
                        '1: :_wtp_worktrees'
                    ;;
                list)
                    _arguments -s \
                        '(--help -h)'{--help,-h}'[Show help]'
                    ;;
                init)
                    _arguments -s \
                        '(--help -h)'{--help,-h}'[Show help]'
                    ;;
                completion)
                    _arguments -s \
                        '(--help -h)'{--help,-h}'[Show help]' \
                        '1: :_wtp_shells'
                    ;;
                shell-init)
                    _arguments -s \
                        '(--help -h)'{--help,-h}'[Show help]'
                    ;;
            esac
            ;;
    esac
}

_wtp_commands() {
    local commands
    commands=(
        'add:Create a new worktree'
        'remove:Remove a worktree'
        'list:List all worktrees'
        'init:Initialize configuration file'
        'completion:Generate shell completion script'
        'shell-init:Initialize shell completion for current session'
        'help:Show help'
    )
    _describe 'commands' commands
}

_wtp_branches() {
    local branches
    branches=(${(f)"$(wtp completion __branches 2>/dev/null)"})
    if [[ ${#branches[@]} -gt 0 ]]; then
        _describe 'branches' branches
    else
        _values 'branches' 'main' 'master' 'develop'
    fi
}

_wtp_worktrees() {
    local worktrees
    worktrees=(${(f)"$(wtp completion __worktrees 2>/dev/null)"})
    if [[ ${#worktrees[@]} -gt 0 ]]; then
        _describe 'worktrees' worktrees
    fi
}

_wtp_remote_branches() {
    local remote_branches
    local git_cmd="git for-each-ref --format='%(refname:short)' refs/remotes 2>/dev/null"
    remote_branches=(${(f)"$($git_cmd | grep -v '/HEAD$')"})
    if [[ ${#remote_branches[@]} -gt 0 ]]; then
        _describe 'remote branches' remote_branches
    fi
}

_wtp_shells() {
    local shells
    shells=(
        'bash:Bash completion'
        'zsh:Zsh completion'
        'fish:Fish completion'
    )
    _describe 'shells' shells
}

_wtp_branches_or_commits() {
    # For positional arguments, complete with branch names primarily
    _wtp_branches
}

_wtp_commits() {
    # Try to use git's completion functions if available, otherwise fallback to basic completion
    if (( $+functions[_git_commits] )) && (( $+functions[_git_tags] )); then
        _alternative \
            'commits:commits:_git_commits' \
            'branches:branches:_wtp_branches' \
            'tags:tags:_git_tags'
    else
        # Fallback to basic branch completion
        _wtp_branches
    fi
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
	printBranches()
}

// printBranches prints available branch names for completion
func printBranches() {
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
	printWorktrees()
}

// printWorktrees prints existing worktree branch names for completion
func printWorktrees() {
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
