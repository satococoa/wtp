package e2e

import (
	"testing"

	"github.com/satococoa/wtp/v2/test/e2e/framework"
)

// Living Specifications for Worktree Creation Workflows
// These tests serve as executable documentation of user behavior

// TestUserCreatesWorktree_WithExistingLocalBranch_ShouldCreateWorktreeAtDefaultPath tests
// the most common user workflow: creating a worktree for an existing local branch.
//
// User Story: As a developer working on a feature branch, I want to create a worktree
// for an existing branch so I can quickly switch to working on that feature in isolation.
//
// Business Value: This eliminates the need to stash changes or commit incomplete work
// when switching between features, improving developer productivity.
func TestUserCreatesWorktree_WithExistingLocalBranch_ShouldCreateWorktreeAtDefaultPath(t *testing.T) {
	// Given: User has an existing local branch named "feature/auth"
	// And: User is in a git repository
	// And: No worktree conflicts exist
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	repo := env.CreateTestRepo("user-creates-worktree")
	repo.CreateBranch("feature/auth")

	// When: User runs "wtp add feature/auth"
	output, err := repo.RunWTP("add", "feature/auth")

	// Then: Worktree should be created successfully
	framework.AssertNoError(t, err)
	framework.AssertWorktreeCreated(t, output, "feature/auth")

	// And: Worktree directory should exist
	framework.AssertWorktreeExists(t, repo, "feature/auth")
}

// TestUserCreatesWorktree_WithNewBranchFlag_ShouldCreateBranchAndWorktree tests
// creating a new branch and worktree simultaneously.
//
// User Story: As a developer starting a new feature, I want to create both a new branch
// and its worktree in one command so I can immediately start working on the feature.
//
// Business Value: Streamlines the workflow of starting new features by combining
// branch creation and worktree setup into a single operation.
func TestUserCreatesWorktree_WithNewBranchFlag_ShouldCreateBranchAndWorktree(t *testing.T) {
	// Given: User wants to create a new branch "feature/payment"
	// And: User is in a git repository
	// And: Branch does not exist yet
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	repo := env.CreateTestRepo("user-creates-new-branch")

	// When: User runs "wtp add --branch feature/payment"
	output, err := repo.RunWTP("add", "--branch", "feature/payment")

	// Then: New branch and worktree should be created
	framework.AssertNoError(t, err)
	framework.AssertWorktreeCreated(t, output, "feature/payment")

	// And: Worktree directory should exist
	framework.AssertWorktreeExists(t, repo, "feature/payment")
}

// TestUserCreatesWorktree_WithCustomPath_ShouldCreateAtSpecifiedLocation tests
// the flexibility to specify custom worktree locations.
//
// User Story: As a developer with specific project organization needs, I want to
// specify exactly where my worktree should be created so it fits my workflow.
//
// Business Value: Provides flexibility for different team workflows and project
// structures, accommodating various developer preferences and constraints.

// TestUserCreatesWorktree_WithoutBranchName_ShouldShowBranchRequiredError tests
// input validation from the user's perspective.
//
// User Story: As a developer, when I forget to specify a branch name, I want to
// receive a clear error message so I understand what's required.
//
// Business Value: Clear error messages reduce frustration and improve the user
// experience by guiding users toward correct usage.
func TestUserCreatesWorktree_WithoutBranchName_ShouldShowBranchRequiredError(t *testing.T) {
	// Given: User is in a git repository
	// And: User doesn't specify a branch name or --branch flag
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	repo := env.CreateTestRepo("user-no-branch")

	// When: User runs "wtp add" with no arguments
	output, err := repo.RunWTP("add")

	// Then: User should receive a clear error message
	framework.AssertError(t, err)
	framework.AssertOutputContains(t, output, "branch name is required")
}

// TestUserCreatesWorktree_WhenPathAlreadyExists_ShouldRequireForceFlag tests
// conflict resolution from the user's perspective.
//
// User Story: As a developer, when I try to create a worktree where a directory
// already exists, I want to be warned and given the option to force overwrite.
//
// Business Value: Prevents accidental data loss while providing flexibility for
// experienced users who want to overwrite existing directories.
func TestUserCreatesWorktree_WhenPathAlreadyExists_ShouldRequireForceFlag(t *testing.T) {
	// Given: Directory already exists at the target path
	// And: User tries to create worktree without force flag
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	repo := env.CreateTestRepo("user-path-exists")
	// Create the feature branch
	repo.CreateBranch("feature/auth")

	// Create a conflicting directory first
	_, err := repo.RunWTP("add", "feature/auth")
	framework.AssertNoError(t, err)

	// When: User tries to run "wtp add feature/auth" again
	output, err := repo.RunWTP("add", "feature/auth")

	// Then: User should receive guidance about the conflict
	framework.AssertError(t, err)
	framework.AssertOutputContains(t, output, "already exists")
}

// TestUserCreatesWorktree_WithBranchFromSpecificCommit tests
// creating a new branch from a specific commit or branch.
//
// User Story: As a developer, when I use "wtp add -b new-branch main",
// I want the new branch to be created from the main branch, not from the current branch.
//
// Business Value: Allows developers to create feature branches from specific
// base branches without having to checkout the base branch first.
func TestUserCreatesWorktree_WithBranchFromSpecificCommit(t *testing.T) {
	// Given: User has main branch and is on a different branch
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	repo := env.CreateTestRepo("user-branch-from-commit")

	// Create and checkout a feature branch
	repo.CreateBranch("feature/current")
	repo.CheckoutBranch("feature/current")

	// Make a commit on feature/current to differentiate it from main
	repo.CommitFile("feature.txt", "feature content", "Add feature")

	// Get the commit hashes
	mainCommit := repo.GetBranchCommitHash("main")
	featureCommit := repo.GetCommitHash() // Current HEAD

	// When: User runs "wtp add -b new-feature main"
	output, err := repo.RunWTP("add", "-b", "new-feature", "main")

	// Then: Command should succeed
	framework.AssertNoError(t, err)
	framework.AssertWorktreeCreated(t, output, "new-feature")

	// And: The new branch should be created from main, not from current branch
	newFeatureCommit := repo.GetBranchCommitHash("new-feature")

	framework.AssertTrue(t,
		newFeatureCommit == mainCommit,
		"new-feature branch should be created from main branch")

	framework.AssertFalse(t,
		newFeatureCommit == featureCommit,
		"new-feature branch should NOT be created from current branch (feature/current)")
}
