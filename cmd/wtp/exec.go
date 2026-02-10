package main

import (
	"context"
	"fmt"
	"io"
	"os"
	"strings"

	"github.com/urfave/cli/v3"

	"github.com/satococoa/wtp/v2/internal/command"
	"github.com/satococoa/wtp/v2/internal/errors"
	"github.com/satococoa/wtp/v2/internal/git"
)

// NewExecCommand creates the exec command definition.
func NewExecCommand() *cli.Command {
	return &cli.Command{
		Name:          "exec",
		Usage:         "Execute a command in a specified worktree",
		UsageText:     "wtp exec <worktree> -- <command> [args...]",
		ArgsUsage:     "<worktree> -- <command> [args...]",
		ShellComplete: completeWorktreesForCd,
		Action:        execCommand,
	}
}

func execCommand(_ context.Context, cmd *cli.Command) error {
	cwd, err := os.Getwd()
	if err != nil {
		return errors.DirectoryAccessFailed("access current", ".", err)
	}

	_, err = git.NewRepository(cwd)
	if err != nil {
		return errors.NotInGitRepository()
	}

	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	executor := command.NewRealExecutor()
	return execCommandWithCommandExecutor(cmd, w, executor)
}

func execCommandWithCommandExecutor(cmd *cli.Command, w io.Writer, executor command.Executor) error {
	worktreeName, commandName, commandArgs, err := parseExecInput(cmd.Args().Slice())
	if err != nil {
		return err
	}

	result, err := executor.Execute([]command.Command{command.GitWorktreeList()})
	if err != nil {
		return errors.GitCommandFailed("git worktree list", err.Error())
	}

	worktrees := parseWorktreesFromOutput(result.Results[0].Output)
	mainWorktreePath := findMainWorktreePath(worktrees)
	targetPath := resolveWorktreePathByName(worktreeName, worktrees, mainWorktreePath)
	if targetPath == "" {
		return errors.WorktreeNotFound(worktreeName, availableManagedWorktreeNames(worktrees, mainWorktreePath))
	}

	execResult, err := executor.Execute([]command.Command{{
		Name:        commandName,
		Args:        commandArgs,
		WorkDir:     targetPath,
		Interactive: true,
	}})
	if err != nil {
		return fmt.Errorf("failed to execute command: %w", err)
	}

	if len(execResult.Results) == 0 {
		return fmt.Errorf("failed to execute command: empty execution result")
	}

	if output := execResult.Results[0].Output; output != "" {
		if _, writeErr := fmt.Fprintln(w, output); writeErr != nil {
			return writeErr
		}
	}

	if execResult.Results[0].Error != nil {
		return fmt.Errorf("command failed in worktree '%s': %w", worktreeName, execResult.Results[0].Error)
	}

	return nil
}

func parseExecInput(args []string) (string, string, []string, error) {
	const usage = "Usage: wtp exec <worktree> -- <command> [args...]"
	if len(args) == 0 {
		return "", "", nil, fmt.Errorf("worktree name is required\n\n%s", usage)
	}

	worktreeName := strings.TrimSpace(args[0])
	if worktreeName == "" {
		return "", "", nil, fmt.Errorf("worktree name is required\n\n%s", usage)
	}

	if len(args) == 1 {
		return "", "", nil, fmt.Errorf("command is required\n\n%s", usage)
	}

	if args[1] == "--" {
		if len(args) < 3 {
			return "", "", nil, fmt.Errorf("command is required\n\n%s", usage)
		}
		return worktreeName, args[2], args[3:], nil
	}

	if len(args) < 2 {
		return "", "", nil, fmt.Errorf("command is required\n\n%s", usage)
	}

	return worktreeName, args[1], args[2:], nil
}
