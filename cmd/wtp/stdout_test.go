package main

import (
	"os"
	"testing"
)

// silenceStdout redirects stdout to os.DevNull for the duration of the returned
// restore function to avoid polluting test output while preserving a writable
// file descriptor for go test tooling (e.g. coverage generation).
func silenceStdout(t *testing.T) func() {
	t.Helper()

	old := os.Stdout

	f, err := os.OpenFile(os.DevNull, os.O_WRONLY, 0)
	if err != nil {
		t.Fatalf("failed to open %s: %v", os.DevNull, err)
	}

	os.Stdout = f

	return func() {
		os.Stdout = old
		if err := f.Close(); err != nil {
			t.Fatalf("failed to close %s: %v", os.DevNull, err)
		}
	}
}
