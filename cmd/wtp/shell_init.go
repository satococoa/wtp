package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

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

func shellInitBash(ctx context.Context, cmd *cli.Command) error {
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	// Output completion first
	if err := completionBash(ctx, cmd); err != nil {
		return err
	}

	// Then output hook
	fmt.Fprintln(w)
	printBashHook(w)

	return nil
}

func shellInitZsh(ctx context.Context, cmd *cli.Command) error {
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	// Output completion first
	if err := completionZsh(ctx, cmd); err != nil {
		return err
	}

	// Then output hook
	fmt.Fprintln(w)
	printZshHook(w)

	return nil
}

func shellInitFish(ctx context.Context, cmd *cli.Command) error {
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	// Output completion first
	if err := completionFish(ctx, cmd); err != nil {
		return err
	}

	// Then output hook
	fmt.Fprintln(w)
	printFishHook(w)

	return nil
}
