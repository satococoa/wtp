package main

import (
	"context"
	"fmt"
	"os"
)

// Version information (set by GoReleaser)
var (
	version = "2.3.1"
	_       = "none"    // commit - set by GoReleaser but not used
	_       = "unknown" // date - set by GoReleaser but not used
)

func main() {
	app := newApp()

	args := normalizeCompletionArgs(os.Args)
	if err := app.Run(context.Background(), args); err != nil {
		fmt.Fprintf(os.Stderr, "%v\n", err)
		os.Exit(1)
	}
}
