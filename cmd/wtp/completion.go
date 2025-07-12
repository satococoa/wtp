package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
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
			"branch names, worktree names, and also include the 'wtp cd' command integration.",
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
				Name:  "powershell",
				Usage: "Generate PowerShell completion script",
				Action: func(_ context.Context, _ *cli.Command) error {
					return fmt.Errorf("PowerShell completion is not supported. Supported shells: bash, zsh, fish")
				},
			},
			{
				Name:   "__branches",
				Hidden: true,
				Usage:  "List branches for completion",
				Action: func(_ context.Context, cmd *cli.Command) error {
					w := cmd.Root().Writer
					if w == nil {
						w = os.Stdout
					}
					printBranches(w)
					return nil
				},
			},
			{
				Name:   "__worktrees",
				Hidden: true,
				Usage:  "List worktrees for completion",
				Action: func(_ context.Context, cmd *cli.Command) error {
					w := cmd.Root().Writer
					if w == nil {
						w = os.Stdout
					}
					printWorktrees(w)
					return nil
				},
			},
		},
	}
}

func completionBash(_ context.Context, cmd *cli.Command) error {
	// Get the writer from cli.Command
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	fmt.Fprintln(w, `#!/bin/bash
# wtp bash completion script with cd integration
# Add this to your ~/.bashrc or ~/.bash_profile:
# eval "$(wtp completion bash)"

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
            cd)
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
            COMPREPLY=( $(compgen -W "add remove list init cd completion help" -- "$cur") )
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
                        # Normal case: first arg is branch, optional second arg is commit-ish
                        if [[ $arg_count -eq 0 ]]; then
                            # First argument: complete with branch names
                            local branches
                            branches=$(wtp completion __branches 2>/dev/null)
                            COMPREPLY=( $(compgen -W "$branches" -- "$cur") )
                        elif [[ $arg_count -eq 1 ]]; then
                            # Second argument: complete with commits/branches/tags
                            # Don't suggest the same branch that was used as first argument
                            local first_arg=""
                            for ((i=2; i<cword; i++)); do
                                if [[ "${words[i]}" != -* ]]; then
                                    local prev_word="${words[i-1]}"
                                    if [[ "$prev_word" != "-b" && "$prev_word" != "--branch" &&
                                          "$prev_word" != "--path" && "$prev_word" != "--reason" &&
                                          "$prev_word" != "-t" && "$prev_word" != "--track" ]]; then
                                        first_arg="${words[i]}"
                                        break
                                    fi
                                fi
                            done

                            local branches
                            branches=$(wtp completion __branches 2>/dev/null | grep -v "^${first_arg}$")
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
                cd)
                    # Get actual worktree branches dynamically
                    local worktrees
                    worktrees=$(wtp completion __worktrees 2>/dev/null)
                    COMPREPLY=( $(compgen -W "$worktrees" -- "$cur") )
                    ;;
            esac
            ;;
    esac
}

complete -F _wtp_completion wtp

# wtp cd command integration
wtp() {
    if [[ "$1" == "cd" ]]; then
        if [[ -z "$2" ]]; then
            command wtp cd
            return
        fi
        local target_dir
        target_dir=$(WTP_SHELL_INTEGRATION=1 command wtp cd "$2" 2>/dev/null)
        if [[ $? -eq 0 && -n "$target_dir" ]]; then
            cd "$target_dir"
        else
            WTP_SHELL_INTEGRATION=1 command wtp cd "$2"
        fi
    elif [[ "$1" == "add" ]]; then
        # Run the add command and capture output
        local output
        output=$(WTP_SHELL_INTEGRATION=1 command wtp "$@" 2>&1)
        local exit_code=$?

        # Check if we should cd (last line is a path)
        if [[ $exit_code -eq 0 ]]; then
            local last_line=$(echo "$output" | tail -n1)
            if [[ -d "$last_line" ]]; then
                # Last line is a directory path - cd to it
                echo "${output%$'\n'*}"  # Print all but last line
                cd "$last_line"
            else
                # Normal output
                echo "$output"
            fi
        else
            # Error occurred
            echo "$output" >&2
            return $exit_code
        fi
    else
        command wtp "$@"
    fi
}`)
	return nil
}

func completionZsh(_ context.Context, cmd *cli.Command) error {
	// Get the writer from cli.Command
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	fmt.Fprintln(w, `#compdef wtp
# wtp zsh completion script with cd integration
# Add this to your ~/.zshrc:
# eval "$(wtp completion zsh)"

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
                    _arguments -C -s \
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
                        '*:::arg:->add_args' && return

                    # Handle positional arguments based on context
                    case $state in
                        add_args)
                            # Count non-option arguments
                            local arg_count=0
                            local has_b_flag=false
                            local i
                            for ((i=2; i<=$#line; i++)); do
                                if [[ "${line[i]}" == "-b" || "${line[i]}" == "--branch" ]]; then
                                    has_b_flag=true
                                elif [[ "${line[i]}" != -* ]]; then
                                    ((arg_count++))
                                fi
                            done

                            if [[ $has_b_flag == true ]]; then
                                # With -b flag: max 1 positional arg (commit-ish)
                                if [[ $arg_count -eq 0 ]]; then
                                    _wtp_branches  # Complete with branches for commit-ish
                                fi
                            else
                                # Without -b flag: max 2 positional args
                                if [[ $arg_count -eq 0 ]]; then
                                    _wtp_branches  # First arg is branch name
                                elif [[ $arg_count -eq 1 ]]; then
                                    _wtp_commits   # Second arg is commit-ish
                                fi
                            fi
                            ;;
                    esac
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
                cd)
                    _arguments -s \
                        '(--help -h)'{--help,-h}'[Show help]' \
                        '1: :_wtp_worktrees'
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
        'cd:Change directory to worktree'
        'completion:Generate shell completion script'
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

_wtp_commits() {
    # Try to use git's completion functions if available, otherwise fallback to basic completion
    if (( $+functions[_git_commits] )) && (( $+functions[_git_tags] )); then
        _alternative \
            'commits:commits:_git_commits' \
            'branches:branches:_wtp_branches_except_first' \
            'tags:tags:_git_tags'
    else
        # Fallback to basic branch completion
        _wtp_branches_except_first
    fi
}

_wtp_branches_except_first() {
    # Get the first non-flag argument (branch name)
    local first_branch=""
    local i
    for ((i=2; i<=$#words; i++)); do
        if [[ "${words[i]}" != -* && "${words[i-1]}" != -* ]]; then
            first_branch="${words[i]}"
            break
        fi
    done

    local branches
    if [[ -n "$first_branch" ]]; then
        branches=(${(f)"$(wtp completion __branches 2>/dev/null | grep -v "^${first_branch}$")"})
    else
        branches=(${(f)"$(wtp completion __branches 2>/dev/null)"})
    fi

    if [[ ${#branches[@]} -gt 0 ]]; then
        _describe 'branches' branches
    else
        _values 'branches' 'main' 'master' 'develop'
    fi
}

if [ -n "$ZSH_VERSION" ]; then
    compdef _wtp wtp
fi

# wtp cd command integration
wtp() {
    if [[ "$1" == "cd" ]]; then
        if [[ -z "$2" ]]; then
            command wtp cd
            return
        fi
        local target_dir
        target_dir=$(WTP_SHELL_INTEGRATION=1 command wtp cd "$2" 2>/dev/null)
        if [[ $? -eq 0 && -n "$target_dir" ]]; then
            cd "$target_dir"
        else
            WTP_SHELL_INTEGRATION=1 command wtp cd "$2"
        fi
    elif [[ "$1" == "add" ]]; then
        # Run the add command and capture output
        local output
        output=$(WTP_SHELL_INTEGRATION=1 command wtp "$@" 2>&1)
        local exit_code=$?

        # Check if we should cd (last line is a path)
        if [[ $exit_code -eq 0 ]]; then
            local last_line=$(echo "$output" | tail -n1)
            if [[ -d "$last_line" ]]; then
                # Last line is a directory path - cd to it
                echo "${output%$'\n'*}"  # Print all but last line
                cd "$last_line"
            else
                # Normal output
                echo "$output"
            fi
        else
            # Error occurred
            echo "$output" >&2
            return $exit_code
        fi
    else
        command wtp "$@"
    fi
}`)
	return nil
}

func completionFish(_ context.Context, cmd *cli.Command) error {
	// Get the writer from cli.Command
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	// Use the built-in fish completion generation
	fish, err := cmd.Root().ToFishCompletion()
	if err != nil {
		return err
	}
	fmt.Fprintln(w, fish)

	// Add cd command integration
	fmt.Fprintln(w, `
# wtp cd command integration
function wtp
    if test "$argv[1]" = "cd"
        if test -z "$argv[2]"
            command wtp cd
            return
        end
        set -l target_dir (env WTP_SHELL_INTEGRATION=1 command wtp cd $argv[2] 2>/dev/null)
        if test $status -eq 0 -a -n "$target_dir"
            cd $target_dir
        else
            env WTP_SHELL_INTEGRATION=1 command wtp cd $argv[2]
        end
    else if test "$argv[1]" = "add"
        # Run the add command and capture output
        set -l output (env WTP_SHELL_INTEGRATION=1 command wtp $argv 2>&1)
        set -l exit_code $status

        # Check if we should cd (last line is a path)
        if test $exit_code -eq 0
            set -l last_line (echo "$output" | tail -n1)
            if test -d "$last_line"
                # Last line is a directory path - cd to it
                echo "$output" | head -n -1  # Print all but last line
                cd "$last_line"
            else
                # Normal output
                echo "$output"
            end
        else
            # Error occurred
            echo "$output" >&2
            return $exit_code
        end
    else
        command wtp $argv
    end
end`)
	return nil
}

// completeBranches provides branch name completion
func completeBranches(_ context.Context, cmd *cli.Command) {
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}
	printBranches(w)
}

// printBranches prints available branch names for completion
func printBranches(w io.Writer) {
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
		fmt.Fprintln(w, displayName)
	}
}

// completeWorktrees provides worktree path completion for remove command
func completeWorktrees(_ context.Context, cmd *cli.Command) {
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}
	printWorktrees(w)
}

// printWorktrees prints existing worktree names for completion
func printWorktrees(w io.Writer) {
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

	// Get repository name for completion
	repoName := repo.GetRepositoryName()

	// Extract worktree names using CompletionName for each worktree
	for _, wt := range worktrees {
		// Use CompletionName method for proper display
		completionName := wt.CompletionName(repoName)
		fmt.Fprintln(w, completionName)
	}
}
