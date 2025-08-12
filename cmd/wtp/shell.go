package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/satococoa/wtp/internal/config"
	"github.com/satococoa/wtp/internal/git"
	"github.com/urfave/cli/v3"
)

// NewShellCommand creates the shell integration command
func NewShellCommand() *cli.Command {
	return &cli.Command{
		Name:  "shell",
		Usage: "Generate shell integration script (includes completion + cd functionality)",
		Description: "Generate shell integration scripts that provide both tab completion and " +
			"cd functionality. This is the recommended way to enable full wtp functionality.",
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
			{
				Name:   "__worktrees_cd",
				Hidden: true,
				Usage:  "List worktrees for cd completion with markers",
				Action: func(_ context.Context, cmd *cli.Command) error {
					w := cmd.Root().Writer
					if w == nil {
						w = os.Stdout
					}

					// Get current directory
					cwd, err := os.Getwd()
					if err != nil {
						return err
					}

					// Initialize repository
					repo, err := git.NewRepository(cwd)
					if err != nil {
						return err
					}

					// Get main worktree path
					mainRepoPath, err := repo.GetMainWorktreePath()
					if err != nil {
						return err
					}

					// Load config
					cfg, err := config.LoadConfig(mainRepoPath)
					if err != nil {
						return err
					}

					// Get all worktrees
					worktrees, err := repo.GetWorktrees()
					if err != nil {
						return err
					}

					// Print with cd-specific formatting
					printWorktriesForCd(w, worktrees, cwd, cfg, mainRepoPath)
					return nil
				},
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
            COMPREPLY=( $(compgen -W "add remove list init cd shell help" -- "$cur") )
            ;;
        *)
            case "${words[1]}" in
                add)
                    local branches
                    branches=$(wtp shell __branches 2>/dev/null)
                    COMPREPLY=( $(compgen -W "$branches" -- "$cur") )
                    ;;
                remove)
                    local worktrees
                    worktrees=$(wtp shell __worktrees 2>/dev/null)
                    COMPREPLY=( $(compgen -W "$worktrees" -- "$cur") )
                    ;;
                shell)
                    COMPREPLY=( $(compgen -W "bash zsh fish" -- "$cur") )
                    ;;
                cd)
                    local worktrees
                    worktrees=$(wtp shell __worktrees_cd 2>/dev/null)
                    COMPREPLY=( $(compgen -W "$worktrees" -- "$cur") )
                    ;;
            esac
            ;;
    esac
}

complete -F _wtp_completion wtp

# Shell integration for cd command
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
        'shell:Generate shell integration script'
        'help:Show help'
    )
    _describe 'commands' commands
}

_wtp_branches() {
    local branches
    branches=(${(f)"$(wtp shell __branches 2>/dev/null)"})
    if [[ ${#branches[@]} -gt 0 ]]; then
        _describe 'branches' branches
    fi
}

_wtp_worktrees() {
    local worktrees
    worktrees=(${(f)"$(wtp shell __worktrees 2>/dev/null)"})
    if [[ ${#worktrees[@]} -gt 0 ]]; then
        _describe 'worktrees' worktrees
    fi
}

_wtp_worktrees_cd() {
    local worktrees
    worktrees=(${(f)"$(wtp shell __worktrees_cd 2>/dev/null)"})
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

# Shell integration for cd command
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
    else
        command wtp $argv
    end
end
`
	fmt.Fprint(w, fishIntegration)
	return nil
}

// Helper functions from completion.go

// completeBranches provides branch name completion
func completeBranches(_ context.Context, cmd *cli.Command) {
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}
	printBranches(w)
}

// completeWorktrees provides worktree path completion for remove command
func completeWorktrees(_ context.Context, cmd *cli.Command) {
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}
	printWorktrees(w)
}

// isWorktreeManagedCompletion determines if a worktree is managed by wtp (for completion)
func isWorktreeManagedCompletion(worktreePath string, cfg *config.Config, mainRepoPath string, isMain bool) bool {
	// Main worktree is always managed
	if isMain {
		return true
	}

	// Get base directory - use default config if config is not available
	if cfg == nil {
		// Create default config when none is available
		defaultCfg := &config.Config{
			Defaults: config.Defaults{
				BaseDir: "../worktrees",
			},
		}
		cfg = defaultCfg
	}

	baseDir := cfg.ResolveWorktreePath(mainRepoPath, "")
	// Remove trailing slash if it exists
	baseDir = strings.TrimSuffix(baseDir, "/")

	// Check if worktree path is under base directory
	absWorktreePath, err := filepath.Abs(worktreePath)
	if err != nil {
		return false
	}

	absBaseDir, err := filepath.Abs(baseDir)
	if err != nil {
		return false
	}

	// Check if worktree is within base directory
	relPath, err := filepath.Rel(absBaseDir, absWorktreePath)
	if err != nil {
		return false
	}

	// If relative path starts with "..", it's outside base directory
	return !strings.HasPrefix(relPath, "..")
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

	// Get main worktree path
	mainRepoPath, err := repo.GetMainWorktreePath()
	if err != nil {
		return
	}

	// Load config
	cfg, err := config.LoadConfig(mainRepoPath)
	if err != nil {
		return
	}

	// Get all worktrees
	worktrees, err := repo.GetWorktrees()
	if err != nil {
		return
	}

	// For now, keep the old behavior for backward compatibility
	// This will be called from hidden __worktrees command
	printWorktriesForRemove(w, worktrees, cfg, mainRepoPath)
}

// printWorktriesForCd prints worktrees for cd command with special formatting
func printWorktriesForCd(
	w io.Writer, worktrees []git.Worktree, currentPath string, cfg *config.Config, mainRepoPath string,
) {
	// Print @ for main worktree
	for i := range worktrees {
		wt := &worktrees[i]
		if wt.IsMain {
			if wt.Path == currentPath {
				fmt.Fprintln(w, "@*")
			} else {
				fmt.Fprintln(w, "@")
			}
			break
		}
	}

	// Print other worktrees with current marker (managed only)
	for i := range worktrees {
		wt := &worktrees[i]
		if !wt.IsMain && isWorktreeManagedCompletion(wt.Path, cfg, mainRepoPath, wt.IsMain) {
			name := getWorktreeNameFromPath(wt.Path, cfg, mainRepoPath, wt.IsMain)
			if wt.Path == currentPath {
				fmt.Fprintf(w, "%s*\n", name)
			} else {
				fmt.Fprintln(w, name)
			}
		}
	}
}

// getWorktreeNameFromPath calculates the worktree name from its path
// For main worktree, returns "@"
// For other worktrees, returns relative path from base_dir
func getWorktreeNameFromPath(worktreePath string, cfg *config.Config, mainRepoPath string, isMain bool) string {
	if isMain {
		return "@"
	}

	// Get base_dir path
	baseDir := cfg.Defaults.BaseDir
	if !filepath.IsAbs(baseDir) {
		baseDir = filepath.Join(mainRepoPath, baseDir)
	}

	// Calculate relative path from base_dir
	relPath, err := filepath.Rel(baseDir, worktreePath)
	if err != nil {
		// Fallback to directory name
		return filepath.Base(worktreePath)
	}

	return relPath
}

// printWorktriesForRemove prints worktrees for remove command (no main, no markers, managed only)
func printWorktriesForRemove(w io.Writer, worktrees []git.Worktree, cfg *config.Config, mainRepoPath string) {
	for i := range worktrees {
		wt := &worktrees[i]
		if !wt.IsMain && isWorktreeManagedCompletion(wt.Path, cfg, mainRepoPath, wt.IsMain) {
			// Calculate worktree name as relative path from base_dir
			name := getWorktreeNameFromPath(wt.Path, cfg, mainRepoPath, wt.IsMain)
			fmt.Fprintln(w, name)
		}
	}
}
