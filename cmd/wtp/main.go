package main

import (
	"context"
	"fmt"
	"os"
)

// Version information
// - In releases: set by GoReleaser via ldflags
// - In dev builds: set by Taskfile via ldflags from git describe
// - Default: used only when built without ldflags (e.g., go run)
// Note: commit and date are set via ldflags but not currently displayed.
// They are available for future use (e.g., verbose version info).
var (
	version = "dev"
	commit  = "none"    //nolint:unused // Set via ldflags, available for future use
	date    = "unknown" //nolint:unused // Set via ldflags, available for future use
)

func main() {
	app := newApp()

	args := normalizeCompletionArgs(os.Args)
	if err := app.Run(context.Background(), args); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
