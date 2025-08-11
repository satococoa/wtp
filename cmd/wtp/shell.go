package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

// NewShellCommand creates the shell integration command
func NewShellCommand() *cli.Command {
	return &cli.Command{
		Name:  "shell",
		Usage: "Generate shell integration script (includes completion + cd functionality)",
		Description: "Generate shell integration scripts that provide both tab completion and " +
			"cd functionality with auto-cd support for wtp add. This is the recommended way " +
			"to enable full wtp functionality.",
		Commands: []*cli.Command{
			{
				Name:        "bash",
				Usage:       "Generate bash integration script",
				Description: "Generate bash integration script with tab completion and cd functionality",
				Action:      shellBash,
			},
			{
				Name:        "zsh",
				Usage:       "Generate zsh integration script",
				Description: "Generate zsh integration script with tab completion and cd functionality",
				Action:      shellZsh,
			},
			{
				Name:        "fish",
				Usage:       "Generate fish integration script",
				Description: "Generate fish integration script with tab completion and cd functionality",
				Action:      shellFish,
			},
		},
	}
}

func shellBash(_ context.Context, cmd *cli.Command) error {
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	bashScript := `#!/bin/bash
# wtp shell integration script for bash
# Add this to your ~/.bashrc:
# eval "$(wtp shell bash)"

# Tab completion
_wtp_completion() {
    local cur prev words cword

    # Use _init_completion if available, otherwise manual setup
    if declare -F _init_completion >/dev/null 2>&1; then
        _init_completion || return
    else
        COMPREPLY=()
        cur="${COMP_WORDS[COMP_CWORD]}"
        prev="${COMP_WORDS[COMP_CWORD-1]}"
        words=("${COMP_WORDS[@]}")
        cword=$COMP_CWORD
    fi

    case $cword in
        1)
            COMPREPLY=( $(compgen -W "add remove list init cd completion shell help" -- "$cur") )
            ;;
        *)
            case "${words[1]}" in
                add)
                    local branches
                    branches=$(wtp completion __branches 2>/dev/null)
                    COMPREPLY=( $(compgen -W "$branches" -- "$cur") )
                    ;;
                remove)
                    local worktrees
                    worktrees=$(wtp completion __worktrees 2>/dev/null)
                    COMPREPLY=( $(compgen -W "$worktrees" -- "$cur") )
                    ;;
                completion)
                    COMPREPLY=( $(compgen -W "bash zsh fish" -- "$cur") )
                    ;;
                shell)
                    COMPREPLY=( $(compgen -W "bash zsh fish" -- "$cur") )
                    ;;
                cd)
                    local worktrees
                    worktrees=$(wtp completion __worktrees_cd 2>/dev/null)
                    COMPREPLY=( $(compgen -W "$worktrees" -- "$cur") )
                    ;;
            esac
            ;;
    esac
}

complete -F _wtp_completion wtp

# Shell integration with WTP_CD_FILE protocol
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
        # Create temporary file for cd protocol
        local _wtp_tmp
        _wtp_tmp=$(mktemp 2>/dev/null) || {
            _wtp_tmp="/tmp/wtp_cd_$$_$(date +%s)"
            touch "$_wtp_tmp" 2>/dev/null || {
                echo "Warning: Could not create temporary file for auto-cd" >&2
                command wtp "$@"
                return $?
            }
        }
        
        # Run wtp add with WTP_CD_FILE
        WTP_CD_FILE="$_wtp_tmp" command wtp "$@"
        local exit_code=$?
        
        # If successful and cd file has content, change directory
        if [[ $exit_code -eq 0 && -s "$_wtp_tmp" ]]; then
            local target_dir
            target_dir=$(cat "$_wtp_tmp" 2>/dev/null)
            if [[ -n "$target_dir" && -d "$target_dir" ]]; then
                cd "$target_dir"
            fi
        fi
        
        # Cleanup
        rm -f "$_wtp_tmp" 2>/dev/null
        
        return $exit_code
    else
        command wtp "$@"
    fi
}
`
	fmt.Fprint(w, bashScript)
	return nil
}

func shellZsh(_ context.Context, cmd *cli.Command) error {
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	zshScript := `#compdef wtp
# wtp shell integration script for zsh
# Add this to your ~/.zshrc:
# eval "$(wtp shell zsh)"

# Tab completion
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
                        '(--help -h)'{--help,-h}'[Show help]' \
                        '*:::arg:_wtp_branches'
                    ;;
                remove)
                    _arguments -s \
                        '(--help -h)'{--help,-h}'[Show help]' \
                        '1: :_wtp_worktrees'
                    ;;
                completion)
                    _arguments -s \
                        '1: :_wtp_shells'
                    ;;
                shell)
                    _arguments -s \
                        '1: :_wtp_shells'
                    ;;
                cd)
                    _arguments -s \
                        '1: :_wtp_worktrees_cd'
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
        'shell:Generate shell integration script'
        'help:Show help'
    )
    _describe 'commands' commands
}

_wtp_branches() {
    local branches
    branches=(${(f)"$(wtp completion __branches 2>/dev/null)"})
    if [[ ${#branches[@]} -gt 0 ]]; then
        _describe 'branches' branches
    fi
}

_wtp_worktrees() {
    local worktrees
    worktrees=(${(f)"$(wtp completion __worktrees 2>/dev/null)"})
    if [[ ${#worktrees[@]} -gt 0 ]]; then
        _describe 'worktrees' worktrees
    fi
}

_wtp_worktrees_cd() {
    local worktrees
    worktrees=(${(f)"$(wtp completion __worktrees_cd 2>/dev/null)"})
    if [[ ${#worktrees[@]} -gt 0 ]]; then
        _describe 'worktrees' worktrees
    fi
}

_wtp_shells() {
    local shells
    shells=(
        'bash:Bash integration'
        'zsh:Zsh integration'
        'fish:Fish integration'
    )
    _describe 'shells' shells
}

if [ -n "$ZSH_VERSION" ]; then
    compdef _wtp wtp
fi

# Shell integration with WTP_CD_FILE protocol
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
        # Create temporary file for cd protocol
        local _wtp_tmp
        _wtp_tmp=$(mktemp 2>/dev/null) || {
            _wtp_tmp="/tmp/wtp_cd_$$_$(date +%s)"
            touch "$_wtp_tmp" 2>/dev/null || {
                echo "Warning: Could not create temporary file for auto-cd" >&2
                command wtp "$@"
                return $?
            }
        }
        
        # Run wtp add with WTP_CD_FILE
        WTP_CD_FILE="$_wtp_tmp" command wtp "$@"
        local exit_code=$?
        
        # If successful and cd file has content, change directory
        if [[ $exit_code -eq 0 && -s "$_wtp_tmp" ]]; then
            local target_dir
            target_dir=$(cat "$_wtp_tmp" 2>/dev/null)
            if [[ -n "$target_dir" && -d "$target_dir" ]]; then
                cd "$target_dir"
            fi
        fi
        
        # Cleanup
        rm -f "$_wtp_tmp" 2>/dev/null
        
        return $exit_code
    else
        command wtp "$@"
    fi
}
`
	fmt.Fprint(w, zshScript)
	return nil
}

func shellFish(_ context.Context, cmd *cli.Command) error {
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	// Use the built-in fish completion generation for base completion
	fish, err := cmd.Root().ToFishCompletion()
	if err != nil {
		return err
	}
	fmt.Fprintln(w, fish)

	// Add shell integration
	fishIntegration := `
# wtp shell integration for fish
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
        # Create temporary file for cd protocol
        set -l _wtp_tmp (mktemp 2>/dev/null)
        if test $status -ne 0
            set _wtp_tmp "/tmp/wtp_cd_"(random)"_"(date +%s)
            touch $_wtp_tmp 2>/dev/null
            if test $status -ne 0
                echo "Warning: Could not create temporary file for auto-cd" >&2
                command wtp $argv
                return $status
            end
        end
        
        # Run wtp add with WTP_CD_FILE
        env WTP_CD_FILE=$_wtp_tmp command wtp $argv
        set -l exit_code $status
        
        # If successful and cd file has content, change directory
        if test $exit_code -eq 0 -a -s "$_wtp_tmp"
            set -l target_dir (cat $_wtp_tmp 2>/dev/null)
            if test -n "$target_dir" -a -d "$target_dir"
                cd $target_dir
            end
        end
        
        # Cleanup
        rm -f $_wtp_tmp 2>/dev/null
        
        return $exit_code
    else
        command wtp $argv
    end
end
`
	fmt.Fprint(w, fishIntegration)
	return nil
}
