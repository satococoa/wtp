package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"

	"github.com/urfave/cli/v3"
)

var runCompletionCommand = func(shell string) ([]byte, error) {
	exe, err := os.Executable()
	if err != nil {
		// Fallback to "wtp" if we can't find the executable
		exe = "wtp"
	}

	cmd := exec.Command(exe, "completion", shell)
	return cmd.Output()
}

// NewShellInitCommand creates the shell-init command definition
func NewShellInitCommand() *cli.Command {
	return &cli.Command{
		Name:  "shell-init",
		Usage: "Initialize shell with completion and cd functionality",
		Description: "Generate shell initialization script that sets up both tab completion and cd functionality. " +
			"This is a convenience command that combines 'wtp completion' and 'wtp hook'.\n\n" +
			"To enable full shell integration, add the following to your shell config:\n" +
			"  Bash (~/.bashrc):         eval \"$(wtp shell-init bash)\"\n" +
			"  Zsh (~/.zshrc):           eval \"$(wtp shell-init zsh)\"\n" +
			"  Fish (~/.config/fish/config.fish): wtp shell-init fish | source",
		Commands: []*cli.Command{
			{
				Name:        "bash",
				Usage:       "Generate bash initialization script",
				Description: "Generate bash initialization script with completion and cd functionality",
				Action:      shellInitBash,
			},
			{
				Name:        "zsh",
				Usage:       "Generate zsh initialization script",
				Description: "Generate zsh initialization script with completion and cd functionality",
				Action:      shellInitZsh,
			},
			{
				Name:        "fish",
				Usage:       "Generate fish initialization script",
				Description: "Generate fish initialization script with completion and cd functionality",
				Action:      shellInitFish,
			},
		},
	}
}

func shellInitBash(_ context.Context, cmd *cli.Command) error {
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	// Output completion first
	if err := outputCompletion(w, "bash"); err != nil {
		return err
	}

	// Then output hook
	fmt.Fprintln(w)
	printBashHook(w)

	return nil
}

func shellInitZsh(_ context.Context, cmd *cli.Command) error {
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	// Output completion first
	if err := outputCompletion(w, "zsh"); err != nil {
		return err
	}

	// Then output hook
	fmt.Fprintln(w)
	printZshHook(w)

	return nil
}

func shellInitFish(_ context.Context, cmd *cli.Command) error {
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	// Output completion first
	if err := outputCompletion(w, "fish"); err != nil {
		return err
	}

	// Then output hook
	fmt.Fprintln(w)
	printFishHook(w)

	return nil
}

// outputCompletion executes wtp completion command and writes output to w

func outputCompletion(w io.Writer, shell string) error {
	output, err := runCompletionCommand(shell)
	if err != nil {
		return fmt.Errorf("failed to generate %s completion: %w", shell, err)
	}

	_, err = w.Write(output)
	return err
}
