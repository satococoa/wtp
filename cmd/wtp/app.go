package main

import "github.com/urfave/cli/v3"

func newApp() *cli.Command {
	return &cli.Command{
		Name:  "wtp",
		Usage: "Enhanced Git worktree management",
		Description: "wtp (Worktree Plus) simplifies Git worktree creation with automatic branch tracking, " +
			"project-specific setup hooks, and convenient defaults.",
		Version:                         version,
		EnableShellCompletion:           true,
		ConfigureShellCompletionCommand: configureCompletionCommand,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:  "version",
				Usage: "Show version information",
			},
		},
		Commands: []*cli.Command{
			NewAddCommand(),
			NewListCommand(),
			NewRemoveCommand(),
			NewCleanCommand(),
			NewInitCommand(),
			NewCdCommand(),
			NewExecCommand(),
			// Built-in completion is automatically provided by urfave/cli
			NewHookCommand(),
			NewShellInitCommand(),
		},
	}
}
