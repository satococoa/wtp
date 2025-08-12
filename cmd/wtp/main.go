package main

import (
	"context"
	"fmt"
	"os"

	"github.com/urfave/cli/v3"
)

// Version information (set by GoReleaser)
var (
	version = "1.1.0"
	_       = "none"    // commit - set by GoReleaser but not used
	_       = "unknown" // date - set by GoReleaser but not used
)

func main() {
	app := &cli.Command{
		Name:  "wtp",
		Usage: "Enhanced Git worktree management",
		Description: "wtp (Worktree Plus) simplifies Git worktree creation with automatic branch tracking, " +
			"project-specific setup hooks, and convenient defaults.",
		Version:               version,
		EnableShellCompletion: true,
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
			NewInitCommand(),
			NewCdCommand(),
			NewShellCommand(),
		},
	}

	if err := app.Run(context.Background(), os.Args); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
