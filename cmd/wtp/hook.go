package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/urfave/cli/v3"
)

// NewHookCommand creates the hook command definition
func NewHookCommand() *cli.Command {
	return &cli.Command{
		Name:  "hook",
		Usage: "Generate shell hook for cd functionality",
		Description: "Generate shell hook scripts that enable the 'wtp cd' command to change directories. " +
			"This provides a seamless navigation experience without needing subshells.\n\n" +
			"To enable the hook, add the following to your shell config:\n" +
			"  Bash (~/.bashrc):         eval \"$(wtp hook bash)\"\n" +
			"  Zsh (~/.zshrc):           eval \"$(wtp hook zsh)\"\n" +
			"  Fish (~/.config/fish/config.fish): wtp hook fish | source",
		Commands: []*cli.Command{
			{
				Name:        "bash",
				Usage:       "Generate bash hook script",
				Description: "Generate bash hook script for cd functionality",
				Action:      hookBash,
			},
			{
				Name:        "zsh",
				Usage:       "Generate zsh hook script",
				Description: "Generate zsh hook script for cd functionality",
				Action:      hookZsh,
			},
			{
				Name:        "fish",
				Usage:       "Generate fish hook script",
				Description: "Generate fish hook script for cd functionality",
				Action:      hookFish,
			},
		},
	}
}

func hookBash(_ context.Context, cmd *cli.Command) error {
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}
	return printBashHook(w)
}

func hookZsh(_ context.Context, cmd *cli.Command) error {
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}
	return printZshHook(w)
}

func hookFish(_ context.Context, cmd *cli.Command) error {
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}
	return printFishHook(w)
}

func printBashHook(w io.Writer) error {
	_, err := fmt.Fprintln(w, `# wtp cd command hook for bash
wtp() {
    for arg in "$@"; do
        if [[ "$arg" == "--generate-shell-completion" ]]; then
            command wtp "$@"
            return $?
        fi
    done
    if [[ "$1" == "cd" ]]; then
        local target_dir
        if [[ -z "$2" ]]; then
            target_dir=$(command wtp cd 2>/dev/null)
        else
            target_dir=$(command wtp cd "$2" 2>/dev/null)
        fi
        if [[ $? -eq 0 && -n "$target_dir" ]]; then
            cd "$target_dir"
        else
            if [[ -z "$2" ]]; then
                command wtp cd
            else
                command wtp cd "$2"
            fi
        fi
    elif [[ "$1" == "add" ]]; then
        if [[ ! -t 1 ]]; then
            command wtp "$@"
            return $?
        fi

        local target_dir
        target_dir=$(command wtp "$@" --quiet)
        local status=$?
        if [[ $status -eq 0 && -n "$target_dir" ]]; then
            cd "$target_dir"
        fi
        return $status
    else
        command wtp "$@"
    fi
}`)

	return err
}

func printZshHook(w io.Writer) error {
	_, err := fmt.Fprintln(w, `# wtp cd command hook for zsh
wtp() {
    for arg in "$@"; do
        if [[ "$arg" == "--generate-shell-completion" ]]; then
            command wtp "$@"
            return $?
        fi
    done
    if [[ "$1" == "cd" ]]; then
        local target_dir
        if [[ -z "$2" ]]; then
            target_dir=$(command wtp cd 2>/dev/null)
        else
            target_dir=$(command wtp cd "$2" 2>/dev/null)
        fi
        if [[ $? -eq 0 && -n "$target_dir" ]]; then
            cd "$target_dir"
        else
            if [[ -z "$2" ]]; then
                command wtp cd
            else
                command wtp cd "$2"
            fi
        fi
    elif [[ "$1" == "add" ]]; then
        if [[ ! -t 1 ]]; then
            command wtp "$@"
            return $?
        fi

        local target_dir
        target_dir=$(command wtp "$@" --quiet)
        local status=$?
        if [[ $status -eq 0 && -n "$target_dir" ]]; then
            cd "$target_dir"
        fi
        return $status
    else
        command wtp "$@"
    fi
}`)

	return err
}

func printFishHook(w io.Writer) error {
	_, err := fmt.Fprintln(w, `# wtp cd command hook for fish
function wtp
    for arg in $argv
        if test "$arg" = "--generate-shell-completion"
            command wtp $argv
            return $status
        end
    end
    if test "$argv[1]" = "cd"
        set -l target_dir
        if test -z "$argv[2]"
            set target_dir (command wtp cd 2>/dev/null)
        else
            set target_dir (command wtp cd $argv[2] 2>/dev/null)
        end
        if test $status -eq 0 -a -n "$target_dir"
            cd "$target_dir"
        else
            if test -z "$argv[2]"
                command wtp cd
            else
                command wtp cd $argv[2]
            end
        end
    else if test "$argv[1]" = "add"
        if not isatty stdout
            command wtp $argv
            return $status
        end

        set -l target_dir (command wtp $argv --quiet)
        set -l status $status
        if test $status -eq 0 -a -n "$target_dir"
            cd "$target_dir"
        end
        return $status
    else
        command wtp $argv
    end
end`)

	return err
}
