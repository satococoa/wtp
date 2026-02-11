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
		ShellComplete: completeWorktreesForExec,
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

	if len(result.Results) == 0 {
		return errors.GitCommandFailed("git worktree list", "no command results")
	}

	gitResult := result.Results[0]
	if gitResult.Error != nil {
		msg := gitResult.Error.Error()
		if gitResult.Output != "" {
			msg = msg + ": " + gitResult.Output
		}
		return errors.GitCommandFailed("git worktree list", msg)
	}

	worktrees := parseWorktreesFromOutput(gitResult.Output)
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

func parseExecInput(args []string) (worktreeName, commandName string, commandArgs []string, err error) {
	const (
		worktreeIndex   = 0
		dashIndex       = 1
		firstCmdIndex   = 2
		minArgsWithDash = 3
	)
	const usage = "Usage: wtp exec <worktree> -- <command> [args...]"
	if len(args) == 0 {
		return "", "", nil, fmt.Errorf("worktree name is required\n\n%s", usage)
	}

	worktreeName = strings.TrimSpace(args[worktreeIndex])
	if worktreeName == "" {
		return "", "", nil, fmt.Errorf("worktree name is required\n\n%s", usage)
	}

	if len(args) == dashIndex {
		return "", "", nil, fmt.Errorf("command is required\n\n%s", usage)
	}

	if args[dashIndex] == "--" {
		if len(args) < minArgsWithDash {
			return "", "", nil, fmt.Errorf("command is required\n\n%s", usage)
		}
		return worktreeName, args[firstCmdIndex], args[firstCmdIndex+1:], nil
	}

	return worktreeName, args[dashIndex], args[firstCmdIndex:], nil
}

func completeWorktreesForExec(ctx context.Context, cmd *cli.Command) {
	current, previous := completionArgsFromCommand(cmd)
	if maybeCompleteFlagSuggestions(cmd, current, previous) {
		return
	}

	for _, arg := range previous {
		if arg == "--" {
			return
		}
	}

	if current == "--" {
		return
	}

	if len(previous) > 0 {
		return
	}

	completeWorktreesForCd(ctx, cmd)
}
