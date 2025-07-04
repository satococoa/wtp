package main

import "github.com/satococoa/wtp/internal/git"

// GitExecutor interface abstracts git command execution for testing
type GitExecutor interface {
	ExecuteGitCommand(args ...string) error
	ResolveBranch(branch string) (resolvedBranch string, isRemote bool, err error)
	GetMainWorktreePath() (string, error)
	Path() string
}

// repositoryExecutor wraps the real git.Repository for production use
type repositoryExecutor struct {
	repo *git.Repository
}

func newRepositoryExecutor(repo *git.Repository) GitExecutor {
	return &repositoryExecutor{repo: repo}
}

func (r *repositoryExecutor) ExecuteGitCommand(args ...string) error {
	return r.repo.ExecuteGitCommand(args...)
}

func (r *repositoryExecutor) ResolveBranch(branch string) (string, bool, error) {
	return r.repo.ResolveBranch(branch)
}

func (r *repositoryExecutor) GetMainWorktreePath() (string, error) {
	return r.repo.GetMainWorktreePath()
}

func (r *repositoryExecutor) Path() string {
	return r.repo.Path()
}

// mockGitExecutor for testing
type mockGitExecutor struct {
	executedCommands [][]string
	resolvedBranch   string
	isRemoteBranch   bool
	resolveBranchErr error
	mainWorktreePath string
	mainWorktreeErr  error
	path             string
	executeErr       error
}

func newMockGitExecutor() *mockGitExecutor {
	return &mockGitExecutor{
		executedCommands: make([][]string, 0),
		path:             "/test/repo",
		mainWorktreePath: "/test/repo",
	}
}

func (m *mockGitExecutor) ExecuteGitCommand(args ...string) error {
	// Copy args to avoid slice modification issues
	argsCopy := make([]string, len(args))
	copy(argsCopy, args)
	m.executedCommands = append(m.executedCommands, argsCopy)
	return m.executeErr
}

func (m *mockGitExecutor) ResolveBranch(branch string) (string, bool, error) {
	if m.resolveBranchErr != nil {
		return "", false, m.resolveBranchErr
	}
	if m.resolvedBranch != "" {
		return m.resolvedBranch, m.isRemoteBranch, nil
	}
	return branch, false, nil
}

func (m *mockGitExecutor) GetMainWorktreePath() (string, error) {
	return m.mainWorktreePath, m.mainWorktreeErr
}

func (m *mockGitExecutor) Path() string {
	return m.path
}

// Test helpers for mock configuration
func (m *mockGitExecutor) SetExecuteError(err error) {
	m.executeErr = err
}

func (m *mockGitExecutor) SetResolveBranch(branch string, isRemote bool, err error) {
	m.resolvedBranch = branch
	m.isRemoteBranch = isRemote
	m.resolveBranchErr = err
}

func (m *mockGitExecutor) SetMainWorktreePath(path string, err error) {
	m.mainWorktreePath = path
	m.mainWorktreeErr = err
}

func (m *mockGitExecutor) SetPath(path string) {
	m.path = path
}

func (m *mockGitExecutor) GetExecutedCommands() [][]string {
	return m.executedCommands
}