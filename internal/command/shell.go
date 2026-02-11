package command

import (
	"os"
	"os/exec"
	"strings"

	"golang.org/x/term"
)

// realShellExecutor implements ShellExecutor using os/exec
type realShellExecutor struct{}

// NewRealShellExecutor creates a new shell executor that executes real commands
func NewRealShellExecutor() ShellExecutor {
	return &realShellExecutor{}
}

// Execute runs the command using os/exec
func (*realShellExecutor) Execute(name string, args []string, workDir string, interactive bool) (string, error) {
	cmd := exec.Command(name, args...)

	if workDir != "" {
		cmd.Dir = workDir
	}

	if interactive && hasTerminalIO() {
		cmd.Stdin = os.Stdin
		cmd.Stdout = os.Stdout
		cmd.Stderr = os.Stderr
		return "", cmd.Run()
	}

	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}

func hasTerminalIO() bool {
	return term.IsTerminal(int(os.Stdin.Fd())) &&
		term.IsTerminal(int(os.Stdout.Fd())) &&
		term.IsTerminal(int(os.Stderr.Fd()))
}
