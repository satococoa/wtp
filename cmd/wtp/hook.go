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
			"  Fish (~/.config/fish/config.fish): wtp hook fish | source\n" +
			"  PowerShell ($PROFILE):    Invoke-Expression -Command (& wtp hook pwsh | Out-String)",
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
			{
				Name:        "pwsh",
				Usage:       "Generate PowerShell hook script",
				Description: "Generate PowerShell hook script for cd functionality",
				Action:      hookPowerShell,
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

func hookPowerShell(_ context.Context, cmd *cli.Command) error {
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}
	return printPowerShellHook(w)
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
    else
        command wtp $argv
    end
end`)

	return err
}

func printPowerShellHook(w io.Writer) error {
	_, err := fmt.Fprintln(w, `# wtp cd command hook for PowerShell
# Store reference to the actual wtp executable
$__wtpPath = $null

# Try to find wtp in PATH first
$__wtpCmd = Get-Command wtp.exe -CommandType Application -ErrorAction SilentlyContinue
if ($__wtpCmd) {
    $__wtpPath = $__wtpCmd.Source
} else {
    $__wtpCmd = Get-Command wtp -CommandType Application -ErrorAction SilentlyContinue
    if ($__wtpCmd) {
        $__wtpPath = $__wtpCmd.Source
    }
}

# If not in PATH, check current directory (for development/testing)
if (-not $__wtpPath) {
    if (Test-Path ".\wtp.exe") {
        $__wtpPath = (Resolve-Path ".\wtp.exe").Path
    } elseif (Test-Path ".\wtp") {
        $__wtpPath = (Resolve-Path ".\wtp").Path
    }
}

function wtp {
    if (-not $__wtpPath) {
        Write-Error "wtp executable not found. Please ensure wtp is in your PATH or current directory."
        return 1
    }

    # Check for completion flag
    foreach ($arg in $args) {
        if ($arg -eq "--generate-shell-completion") {
            & $__wtpPath @args
            return $LASTEXITCODE
        }
    }

    if ($args[0] -eq "cd") {
        if (-not $args[1]) {
            Write-Error "Usage: wtp cd <worktree>"
            return 1
        }
        $targetDir = & $__wtpPath cd $args[1] 2>$null
        if ($LASTEXITCODE -eq 0 -and $targetDir) {
            Set-Location $targetDir
        } else {
            & $__wtpPath cd $args[1]
        }
    } else {
        & $__wtpPath @args
    }
}`)

	return err
}
