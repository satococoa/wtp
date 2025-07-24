package e2e

import (
	"testing"

	"github.com/satococoa/wtp/test/e2e/framework"
)

func TestRemoteBranchHandling(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("SingleRemoteBranch", func(t *testing.T) {
		repo := env.CreateTestRepo("remote-single")
		repo.AddRemote("origin", "https://example.com/repo.git")
		repo.CreateRemoteBranch("origin", "remote-feature")

		output, err := repo.RunWTP("add", "remote-feature")
		framework.AssertNoError(t, err)
		framework.AssertWorktreeCreated(t, output, "remote-feature")
		framework.AssertWorktreeExists(t, repo, "remote-feature")
	})

	t.Run("MultipleRemotes", func(t *testing.T) {
		repo := env.CreateTestRepo("remote-multiple")
		repo.AddRemote("origin", "https://example.com/repo.git")
		repo.AddRemote("upstream", "https://example.com/upstream.git")
		repo.CreateRemoteBranch("origin", "shared-branch")
		repo.CreateRemoteBranch("upstream", "shared-branch")

		output, err := repo.RunWTP("add", "shared-branch")
		framework.AssertError(t, err)
		framework.AssertOutputContains(t, output, "exists in multiple remotes")
		framework.AssertOutputContains(t, output, "origin")
		framework.AssertOutputContains(t, output, "upstream")
		framework.AssertHelpfulError(t, output)
	})

	t.Run("ExplicitRemoteTracking", func(t *testing.T) {
		repo := env.CreateTestRepo("remote-explicit")
		repo.AddRemote("origin", "https://example.com/repo.git")
		repo.AddRemote("upstream", "https://example.com/upstream.git")
		repo.CreateRemoteBranch("origin", "explicit-branch")
		repo.CreateRemoteBranch("upstream", "explicit-branch")

		// With new simplified interface, create new branch from specific remote
		output, err := repo.RunWTP("add", "-b", "explicit-branch", "origin/explicit-branch")
		framework.AssertNoError(t, err)
		framework.AssertWorktreeCreated(t, output, "explicit-branch")
		framework.AssertWorktreeExists(t, repo, "explicit-branch")
	})

	t.Run("RemoteOnlyBranch", func(t *testing.T) {
		repo := env.CreateTestRepo("remote-only")
		repo.AddRemote("origin", "https://example.com/repo.git")
		repo.CreateRemoteBranch("origin", "remote-only-branch")

		output, err := repo.RunWTP("add", "remote-only-branch")
		framework.AssertNoError(t, err)
		framework.AssertWorktreeCreated(t, output, "remote-only-branch")

		// Check that branch is tracking the remote
		branchOutput, _ := repo.RunWTP("branch", "-vv")
		_ = branchOutput // Branch tracking verification would go here
	})

	t.Run("NonExistentRemoteBranch", func(t *testing.T) {
		repo := env.CreateTestRepo("remote-nonexistent")
		repo.AddRemote("origin", "https://example.com/repo.git")

		output, err := repo.RunWTP("add", "nonexistent-remote-branch")
		framework.AssertError(t, err)
		framework.AssertOutputContains(t, output, "not found in local or remote branches")
		framework.AssertHelpfulError(t, output)
	})

	t.Run("LocalTakesPrecedence", func(t *testing.T) {
		repo := env.CreateTestRepo("remote-precedence")
		repo.AddRemote("origin", "https://example.com/repo.git")
		repo.CreateBranch("precedence-branch")
		repo.CreateRemoteBranch("origin", "precedence-branch")

		output, err := repo.RunWTP("add", "precedence-branch")
		framework.AssertNoError(t, err)
		framework.AssertWorktreeCreated(t, output, "precedence-branch")

		// Should use local branch, not remote
		worktrees := repo.ListWorktrees()
		framework.AssertEqual(t, 2, len(worktrees))
	})

	t.Run("RemoteBranchWithSlashes", func(t *testing.T) {
		repo := env.CreateTestRepo("remote-slashes")
		repo.AddRemote("origin", "https://example.com/repo.git")
		repo.CreateRemoteBranch("origin", "feature/remote/nested")

		output, err := repo.RunWTP("add", "feature/remote/nested")
		framework.AssertNoError(t, err)
		framework.AssertWorktreeCreated(t, output, "feature/remote/nested")
		framework.AssertWorktreeExists(t, repo, "feature/remote/nested")
	})
}

func TestRemoteConfiguration(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("NoRemotes", func(t *testing.T) {
		repo := env.CreateTestRepo("no-remotes")

		output, err := repo.RunWTP("add", "remote-branch")
		framework.AssertError(t, err)
		framework.AssertOutputContains(t, output, "not found in local or remote branches")
	})

	t.Run("InvalidRemoteURL", func(t *testing.T) {
		repo := env.CreateTestRepo("invalid-remote")

		// Add remote with invalid URL format
		_ = env.RunInDir(repo.Path(), "git", "remote", "add", "invalid", "not-a-url")
		// Git might accept this, but it's still invalid

		repo.CreateRemoteBranch("invalid", "test-branch")

		// wtp should still work with the remote branch
		output, err := repo.RunWTP("add", "test-branch")
		framework.AssertNoError(t, err)
		framework.AssertWorktreeCreated(t, output, "test-branch")
	})

	t.Run("CaseSensitivity", func(t *testing.T) {
		repo := env.CreateTestRepo("case-sensitive")
		repo.AddRemote("origin", "https://example.com/repo.git")
		repo.CreateRemoteBranch("origin", "Feature/CaseSensitive")

		// Try with different case
		output, err := repo.RunWTP("add", "feature/casesensitive")
		framework.AssertError(t, err)
		framework.AssertOutputContains(t, output, "not found")

		// Try with correct case
		output, err = repo.RunWTP("add", "Feature/CaseSensitive")
		framework.AssertNoError(t, err)
		framework.AssertWorktreeCreated(t, output, "Feature/CaseSensitive")
	})
}

func TestSimplifiedInterfaceBehavior(t *testing.T) {
	env := framework.NewTestEnvironment(t)
	defer env.Cleanup()

	t.Run("NewBranchFromRemote", func(t *testing.T) {
		repo := env.CreateTestRepo("new-from-remote")
		repo.AddRemote("origin", "https://example.com/repo.git")
		repo.CreateRemoteBranch("origin", "remote-feature")

		// Create new branch from remote using simplified interface
		output, err := repo.RunWTP("add", "-b", "local-feature", "origin/remote-feature")
		framework.AssertNoError(t, err)
		framework.AssertWorktreeCreated(t, output, "local-feature")
		framework.AssertWorktreeExists(t, repo, "local-feature")
	})

	t.Run("NewBranchFromCommit", func(t *testing.T) {
		repo := env.CreateTestRepo("new-from-commit")
		repo.CreateBranch("source-branch")

		// Create new branch from specific commit
		output, err := repo.RunWTP("add", "-b", "new-branch", "main")
		framework.AssertNoError(t, err)
		framework.AssertWorktreeCreated(t, output, "new-branch")
		framework.AssertWorktreeExists(t, repo, "new-branch")
	})
}
