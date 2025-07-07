package command

import (
	"os/exec"
	"strings"
)

// realShellExecutor implements ShellExecutor using os/exec
type realShellExecutor struct{}

// NewRealShellExecutor creates a new shell executor that executes real commands
func NewRealShellExecutor() ShellExecutor {
	return &realShellExecutor{}
}

// Execute runs the command using os/exec
func (s *realShellExecutor) Execute(name string, args []string, workDir string) (string, error) {
	cmd := exec.Command(name, args...)

	if workDir != "" {
		cmd.Dir = workDir
	}

	output, err := cmd.CombinedOutput()
	return strings.TrimSpace(string(output)), err
}
