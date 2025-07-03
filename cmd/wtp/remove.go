package main

import (
	"context"
	"fmt"
	"io"
	"os"

	"github.com/satococoa/wtp/internal/errors"
	"github.com/satococoa/wtp/internal/git"
	"github.com/urfave/cli/v3"
)

// Variable to allow mocking in tests
var removeGetwd = os.Getwd

// NewRemoveCommand creates the remove command definition
func NewRemoveCommand() *cli.Command {
	return &cli.Command{
		Name:      "remove",
		Usage:     "Remove a worktree",
		UsageText: "wtp remove <branch-name>",
		Description: "Removes the worktree associated with the specified branch.\n\n" +
			"Examples:\n" +
			"  wtp remove feature/old                  # Remove worktree\n" +
			"  wtp remove -f feature/dirty             # Force remove dirty worktree\n" +
			"  wtp remove --with-branch feature/done   # Also delete the branch",
		ShellComplete: completeWorktrees,
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name:    "force",
				Usage:   "Force removal even if worktree is dirty",
				Aliases: []string{"f"},
			},
			&cli.BoolFlag{
				Name:  "with-branch",
				Usage: "Also remove the branch after removing worktree",
			},
			&cli.BoolFlag{
				Name:  "force-branch",
				Usage: "Force branch deletion even if not merged (requires --with-branch)",
			},
		},
		Action: removeCommand,
	}
}

func removeCommand(_ context.Context, cmd *cli.Command) error {
	// Get the writer from cli.Command
	w := cmd.Root().Writer
	if w == nil {
		w = os.Stdout
	}

	// Extract and validate inputs
	branchName := cmd.Args().Get(0)
	force := cmd.Bool("force")
	withBranch := cmd.Bool("with-branch")
	forceBranch := cmd.Bool("force-branch")

	if err := validateRemoveInput(branchName, withBranch, forceBranch); err != nil {
		return err
	}

	// Get repository
	repo, err := getRepository()
	if err != nil {
		return err
	}

	// Find target worktree
	targetWorktree, err := findTargetWorktree(repo, branchName)
	if err != nil {
		return err
	}

	// Remove worktree
	if err := repo.RemoveWorktree(targetWorktree.Path, force); err != nil {
		return errors.WorktreeRemovalFailed(targetWorktree.Path, err)
	}
	fmt.Fprintf(w, "Removed worktree '%s' at %s\n", branchName, targetWorktree.Path)

	// Remove branch if requested
	if withBranch {
		if err := removeBranch(w, repo, branchName, forceBranch); err != nil {
			return err
		}
	}

	return nil
}

func validateRemoveInput(branchName string, withBranch, forceBranch bool) error {
	if branchName == "" {
		return errors.BranchNameRequired("wtp remove <branch-name>")
	}
	if forceBranch && !withBranch {
		return fmt.Errorf("--force-branch requires --with-branch")
	}
	return nil
}

func getRepository() (*git.Repository, error) {
	cwd, err := removeGetwd()
	if err != nil {
		return nil, errors.DirectoryAccessFailed("access current", ".", err)
	}
	repo, err := git.NewRepository(cwd)
	if err != nil {
		return nil, errors.NotInGitRepository()
	}
	return repo, nil
}

func findTargetWorktree(repo *git.Repository, branchName string) (*git.Worktree, error) {
	worktrees, err := repo.GetWorktrees()
	if err != nil {
		return nil, errors.GitCommandFailed("git worktree list", err.Error())
	}

	var targetWorktree *git.Worktree
	var availableBranches []string
	for _, wt := range worktrees {
		if wt.Branch != "" {
			availableBranches = append(availableBranches, wt.Branch)
		}
		if wt.Branch == branchName {
			targetWorktree = &wt
			break
		}
	}

	if targetWorktree == nil {
		return nil, errors.WorktreeNotFound(branchName, availableBranches)
	}
	return targetWorktree, nil
}

func removeBranch(w io.Writer, repo *git.Repository, branchName string, forceBranch bool) error {
	args := []string{"branch"}
	if forceBranch {
		args = append(args, "-D")
	} else {
		args = append(args, "-d")
	}
	args = append(args, branchName)

	if err := repo.ExecuteGitCommand(args...); err != nil {
		return errors.BranchRemovalFailed(branchName, err, forceBranch)
	}
	fmt.Fprintf(w, "Removed branch '%s'\n", branchName)
	return nil
}
