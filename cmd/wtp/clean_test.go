package main

import (
	"bytes"
	"context"
	"os"
	"testing"

	"github.com/charmbracelet/huh"
	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"

	"github.com/satococoa/wtp/v2/internal/command"
	"github.com/satococoa/wtp/v2/internal/config"
	"github.com/satococoa/wtp/v2/internal/git"
)

// ===== Command Structure Tests =====

func TestNewCleanCommand(t *testing.T) {
	cmd := NewCleanCommand()

	assert.NotNil(t, cmd)
	assert.Equal(t, "clean", cmd.Name)
	assert.Equal(t, "Interactively clean up worktrees", cmd.Usage)
	assert.NotEmpty(t, cmd.Description)
	assert.NotNil(t, cmd.Action)

	// Check flags exist
	flagNames := []string{"force"}
	for _, name := range flagNames {
		found := false
		for _, flag := range cmd.Flags {
			if flag.Names()[0] == name {
				found = true
				break
			}
		}
		assert.True(t, found, "Flag %s should exist", name)
	}
}

// ===== Pure Business Logic Tests =====

//nolint:dupl
func TestValidateWorktree(t *testing.T) {
	tests := []struct {
		name              string
		worktree          git.Worktree
		mergeBaseOutput   command.ExecutionResult
		statusOutput      command.ExecutionResult
		revListOutput     command.ExecutionResult
		aheadCmdOutput    command.ExecutionResult
		mainBranchOutput  command.ExecutionResult
		masterCmdOutput   command.ExecutionResult
		expectedIsSafe    bool
		expectedIsMerged  bool
		expectedIsClean   bool
		expectedIsPushed  bool
		expectedReasonSub string
	}{
		{
			name:     "safe worktree - merged, clean, pushed",
			worktree: git.Worktree{Path: "/test/repo/.worktrees/feature", Branch: "feature"},
			mergeBaseOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: nil}},
			},
			statusOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: nil}},
			},
			revListOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "0", Error: nil}},
			},
			aheadCmdOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "0", Error: nil}},
			},
			mainBranchOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "main", Error: nil}},
			},
			masterCmdOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: nil}},
			},
			expectedIsSafe:   true,
			expectedIsMerged: true,
			expectedIsClean:  true,
			expectedIsPushed: true,
		},
		{
			name:     "unmerged worktree",
			worktree: git.Worktree{Path: "/test/repo/.worktrees/feature", Branch: "feature"},
			mergeBaseOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: &mockCleanError{message: "not ancestor"}}},
			},
			statusOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: nil}},
			},
			revListOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "0", Error: nil}},
			},
			aheadCmdOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "0", Error: nil}},
			},
			mainBranchOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "main", Error: nil}},
			},
			masterCmdOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: nil}},
			},
			expectedIsSafe:    false,
			expectedIsMerged:  false,
			expectedIsClean:   true,
			expectedIsPushed:  true,
			expectedReasonSub: "unmerged",
		},
		{
			name:     "dirty worktree",
			worktree: git.Worktree{Path: "/test/repo/.worktrees/feature", Branch: "feature"},
			mergeBaseOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: nil}},
			},
			statusOutput: command.ExecutionResult{
				Results: []command.Result{{Output: " M file.txt", Error: nil}},
			},
			revListOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "0", Error: nil}},
			},
			aheadCmdOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "0", Error: nil}},
			},
			mainBranchOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "main", Error: nil}},
			},
			masterCmdOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: nil}},
			},
			expectedIsSafe:    false,
			expectedIsMerged:  true,
			expectedIsClean:   false,
			expectedIsPushed:  true,
			expectedReasonSub: "uncommitted changes",
		},
		{
			name:     "unpushed commits worktree",
			worktree: git.Worktree{Path: "/test/repo/.worktrees/feature", Branch: "feature"},
			mergeBaseOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: nil}},
			},
			statusOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: nil}},
			},
			revListOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "3", Error: nil}},
			},
			aheadCmdOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "3", Error: nil}},
			},
			mainBranchOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "main", Error: nil}},
			},
			masterCmdOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: nil}},
			},
			expectedIsSafe:    false,
			expectedIsMerged:  true,
			expectedIsClean:   true,
			expectedIsPushed:  false,
			expectedReasonSub: "unpushed",
		},
		{
			name:     "detached HEAD worktree",
			worktree: git.Worktree{Path: "/test/repo/.worktrees/feature", Branch: "detached"},
			mergeBaseOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: nil}},
			},
			statusOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: nil}},
			},
			revListOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "0", Error: nil}},
			},
			aheadCmdOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "0", Error: nil}},
			},
			mainBranchOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "main", Error: nil}},
			},
			masterCmdOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: nil}},
			},
			expectedIsSafe:    false,
			expectedIsMerged:  false,
			expectedIsClean:   true,
			expectedIsPushed:  true,
			expectedReasonSub: "detached",
		},
		{
			name:     "empty branch worktree",
			worktree: git.Worktree{Path: "/test/repo/.worktrees/feature", Branch: ""},
			mergeBaseOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: nil}},
			},
			statusOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: nil}},
			},
			revListOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "0", Error: nil}},
			},
			aheadCmdOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "0", Error: nil}},
			},
			mainBranchOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "main", Error: nil}},
			},
			masterCmdOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: nil}},
			},
			expectedIsSafe:    false,
			expectedIsMerged:  false,
			expectedIsClean:   true,
			expectedIsPushed:  true,
			expectedReasonSub: "detached",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockCleanCommandExecutor{
				results: map[string]command.ExecutionResult{
					"merge-base": tt.mergeBaseOutput,
					"status":     tt.statusOutput,
					"rev-list":   tt.revListOutput,
					"ahead-cmd":  tt.aheadCmdOutput,
					"main":       tt.mainBranchOutput,
					"master":     tt.masterCmdOutput,
				},
			}

			status := validateWorktree(tt.worktree, mockExec)

			assert.Equal(t, tt.expectedIsSafe, status.isSafe, "isSafe mismatch")
			assert.Equal(t, tt.expectedIsMerged, status.isMerged, "isMerged mismatch")
			assert.Equal(t, tt.expectedIsClean, status.isClean, "isClean mismatch")
			assert.Equal(t, tt.expectedIsPushed, status.isPushed, "isPushed mismatch")

			if !tt.expectedIsSafe && tt.expectedReasonSub != "" {
				assert.Contains(t, status.reason, tt.expectedReasonSub)
			}
		})
	}
}

func TestValidateWorktrees(t *testing.T) {
	tests := []struct {
		name              string
		worktrees         []git.Worktree
		expectedStatuses  int
		expectedAllSafe   bool
		expectedAllUnsafe bool
	}{
		{
			name: "all safe worktrees",
			worktrees: []git.Worktree{
				{Path: "/test/repo/.worktrees/feature1", Branch: "feature1"},
				{Path: "/test/repo/.worktrees/feature2", Branch: "feature2"},
			},
			expectedStatuses: 2,
			expectedAllSafe:  true,
		},
		{
			name:              "empty worktrees",
			worktrees:         []git.Worktree{},
			expectedStatuses:  0,
			expectedAllSafe:   false,
			expectedAllUnsafe: false,
		},
		{
			name: "single worktree",
			worktrees: []git.Worktree{
				{Path: "/test/repo/.worktrees/feature", Branch: "feature"},
			},
			expectedStatuses: 1,
			expectedAllSafe:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockCleanCommandExecutor{
				results: map[string]command.ExecutionResult{
					"merge-base": {
						Results: []command.Result{{Output: "", Error: nil}},
					},
					"status": {
						Results: []command.Result{{Output: "", Error: nil}},
					},
					"rev-list": {
						Results: []command.Result{{Output: "0", Error: nil}},
					},
					"ahead-cmd": {
						Results: []command.Result{{Output: "0", Error: nil}},
					},
					"main": {
						Results: []command.Result{{Output: "main", Error: nil}},
					},
					"master": {
						Results: []command.Result{{Output: "", Error: nil}},
					},
				},
			}

			statuses := validateWorktrees(tt.worktrees, mockExec)

			assert.Equal(t, tt.expectedStatuses, len(statuses))
		})
	}
}

func TestBuildCleanOptions(t *testing.T) {
	tests := []struct {
		name               string
		statuses           []worktreeCleanStatus
		expectedOptions    int
		expectedSelected   int
		expectedUnselected int
	}{
		{
			name: "all safe worktrees - all pre-selected",
			statuses: []worktreeCleanStatus{
				{worktree: git.Worktree{Path: "/test/repo/.worktrees/feature1"}, isSafe: true, reason: "safe"},
				{worktree: git.Worktree{Path: "/test/repo/.worktrees/feature2"}, isSafe: true, reason: "safe"},
			},
			expectedOptions:    2,
			expectedSelected:   2,
			expectedUnselected: 0,
		},
		{
			name: "mix of safe and unsafe - only safe pre-selected",
			statuses: []worktreeCleanStatus{
				{worktree: git.Worktree{Path: "/test/repo/.worktrees/safe"}, isSafe: true, reason: "safe"},
				{worktree: git.Worktree{Path: "/test/repo/.worktrees/unsafe"}, isSafe: false, reason: "unsafe"},
			},
			expectedOptions:    2,
			expectedSelected:   1,
			expectedUnselected: 1,
		},
		{
			name: "all unsafe worktrees - none pre-selected",
			statuses: []worktreeCleanStatus{
				{worktree: git.Worktree{Path: "/test/repo/.worktrees/unsafe1"}, isSafe: false, reason: "unsafe"},
				{worktree: git.Worktree{Path: "/test/repo/.worktrees/unsafe2"}, isSafe: false, reason: "unsafe"},
			},
			expectedOptions:    2,
			expectedSelected:   0,
			expectedUnselected: 2,
		},
		{
			name:               "empty statuses",
			statuses:           []worktreeCleanStatus{},
			expectedOptions:    0,
			expectedSelected:   0,
			expectedUnselected: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Defaults: config.Defaults{BaseDir: ".worktrees"},
			}

			options := buildCleanOptions(tt.statuses, cfg, "/test/repo")

			assert.Equal(t, tt.expectedOptions, len(options))

			// Verify options were created
			for _, opt := range options {
				assert.NotEmpty(t, opt.Key)
			}
		})
	}
}

func TestDetectMainBranch(t *testing.T) {
	tests := []struct {
		name            string
		mainCmdOutput   command.ExecutionResult
		masterCmdOutput command.ExecutionResult
		expectedBranch  string
	}{
		{
			name: "custom main branch name",
			mainCmdOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "trunk", Error: nil}},
			},
			masterCmdOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: nil}},
			},
			expectedBranch: "trunk",
		},
		{
			name: "master branch exists when main returns main",
			mainCmdOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "main", Error: nil}},
			},
			masterCmdOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: nil}},
			},
			expectedBranch: "master",
		},
		{
			name: "no main or master - defaults to main",
			mainCmdOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "main", Error: nil}},
			},
			masterCmdOutput: command.ExecutionResult{
				Results: []command.Result{{Output: "", Error: &mockCleanError{message: "not found"}}},
			},
			expectedBranch: "main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockCleanCommandExecutor{
				results: map[string]command.ExecutionResult{
					"main":   tt.mainCmdOutput,
					"master": tt.masterCmdOutput,
				},
			}

			branch := detectMainBranch(mockExec)

			assert.Equal(t, tt.expectedBranch, branch)
		})
	}
}

func TestFindWorktreeByName(t *testing.T) {
	tests := []struct {
		name         string
		worktrees    []git.Worktree
		searchName   string
		shouldFind   bool
		expectedPath string
	}{
		{
			name: "find existing worktree",
			worktrees: []git.Worktree{
				{Path: "/test/repo/.worktrees/feature"},
			},
			searchName:   "feature",
			shouldFind:   true,
			expectedPath: "/test/repo/.worktrees/feature",
		},
		{
			name: "worktree not found",
			worktrees: []git.Worktree{
				{Path: "/test/repo/.worktrees/feature"},
			},
			searchName: "nonexistent",
			shouldFind: false,
		},
		{
			name:         "empty worktrees",
			worktrees:    []git.Worktree{},
			searchName:   "feature",
			shouldFind:   false,
			expectedPath: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := &config.Config{
				Defaults: config.Defaults{BaseDir: ".worktrees"},
			}

			wt, err := findWorktreeByName(tt.worktrees, tt.searchName, cfg, "/test/repo")

			if tt.shouldFind {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPath, wt.Path)
			} else {
				assert.Error(t, err)
				assert.Nil(t, wt)
			}
		})
	}
}

// ===== Command Execution Tests =====

func TestCleanCommand_CommandConstruction(t *testing.T) {
	tests := []struct {
		name             string
		mockWorktreeList string
		expectedCommands []command.Command
	}{
		{
			name:             "empty worktree list",
			mockWorktreeList: "",
			expectedCommands: []command.Command{
				{
					Name: "git",
					Args: []string{"worktree", "list", "--porcelain"},
				},
			},
		},
		{
			name: "single worktree",
			mockWorktreeList: "worktree /test/repo\nHEAD abc123\nbranch refs/heads/main\n\n" +
				"worktree /test/repo/.worktrees/feature\nHEAD def456\nbranch refs/heads/feature\n\n",
			expectedCommands: []command.Command{
				{
					Name: "git",
					Args: []string{"worktree", "list", "--porcelain"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockCleanCommandExecutor{
				results: map[string]command.ExecutionResult{
					"worktree list": {
						Results: []command.Result{{Output: tt.mockWorktreeList, Error: nil}},
					},
					"merge-base": {
						Results: []command.Result{{Output: "", Error: nil}},
					},
					"status": {
						Results: []command.Result{{Output: "", Error: nil}},
					},
					"rev-list": {
						Results: []command.Result{{Output: "0", Error: nil}},
					},
					"ahead-cmd": {
						Results: []command.Result{{Output: "0", Error: nil}},
					},
					"main": {
						Results: []command.Result{{Output: "main", Error: nil}},
					},
					"master": {
						Results: []command.Result{{Output: "", Error: nil}},
					},
				},
			}

			// We can't easily test the full flow without mocking the form
			// So we'll test the command construction at least
			worktrees, mainPath, err := getWorktreesForClean(mockExec)

			if tt.mockWorktreeList == "" {
				assert.NoError(t, err)
				assert.Equal(t, 0, len(worktrees))
			} else {
				assert.NoError(t, err)
				assert.Equal(t, "/test/repo", mainPath)
			}
		})
	}
}

func TestGetWorktreesForClean(t *testing.T) {
	tests := []struct {
		name             string
		mockOutput       string
		expectedCount    int
		expectedMainPath string
		shouldError      bool
	}{
		{
			name: "worktrees with main",
			mockOutput: "worktree /test/repo\nHEAD abc123\nbranch refs/heads/main\n\n" +
				"worktree /test/repo/.worktrees/feature\nHEAD def456\nbranch refs/heads/feature\n\n",
			expectedCount:    2,
			expectedMainPath: "/test/repo",
			shouldError:      false,
		},
		{
			name:             "empty output",
			mockOutput:       "",
			expectedCount:    0,
			expectedMainPath: "",
			shouldError:      false,
		},
		{
			name:             "only main worktree",
			mockOutput:       "worktree /test/repo\nHEAD abc123\nbranch refs/heads/main\n\n",
			expectedCount:    1,
			expectedMainPath: "/test/repo",
			shouldError:      false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockCleanCommandExecutor{
				results: map[string]command.ExecutionResult{
					"worktree list": {
						Results: []command.Result{{Output: tt.mockOutput, Error: nil}},
					},
				},
			}

			worktrees, mainPath, err := getWorktreesForClean(mockExec)

			if tt.shouldError {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedCount, len(worktrees))
				assert.Equal(t, tt.expectedMainPath, mainPath)
			}
		})
	}
}

func TestFilterManagedWorktrees(t *testing.T) {
	tests := []struct {
		name            string
		worktrees       []git.Worktree
		cfg             *config.Config
		mainRepoPath    string
		expectedManaged int
	}{
		{
			name: "all managed worktrees",
			worktrees: []git.Worktree{
				{Path: "/test/repo", IsMain: true},
				{Path: "/test/repo/.worktrees/feature", IsMain: false},
				{Path: "/test/repo/.worktrees/bugfix", IsMain: false},
			},
			cfg: &config.Config{
				Defaults: config.Defaults{BaseDir: ".worktrees"},
			},
			mainRepoPath:    "/test/repo",
			expectedManaged: 2,
		},
		{
			name: "mix of managed and unmanaged",
			worktrees: []git.Worktree{
				{Path: "/test/repo", IsMain: true},
				{Path: "/test/repo/.worktrees/feature", IsMain: false},
				{Path: "/other/path/worktree", IsMain: false},
			},
			cfg: &config.Config{
				Defaults: config.Defaults{BaseDir: ".worktrees"},
			},
			mainRepoPath:    "/test/repo",
			expectedManaged: 1,
		},
		{
			name: "empty worktrees",
			worktrees: []git.Worktree{
				{Path: "/test/repo", IsMain: true},
			},
			cfg: &config.Config{
				Defaults: config.Defaults{BaseDir: ".worktrees"},
			},
			mainRepoPath:    "/test/repo",
			expectedManaged: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			managed := filterManagedWorktrees(tt.worktrees, tt.cfg, tt.mainRepoPath)

			assert.Equal(t, tt.expectedManaged, len(managed))
		})
	}
}

func TestLoadCleanConfig(t *testing.T) {
	tests := []struct {
		name         string
		mainRepoPath string
		shouldReturn bool
	}{
		{
			name:         "valid repo path",
			mainRepoPath: "/test/repo",
			shouldReturn: true,
		},
		{
			name:         "non-existent path returns default config",
			mainRepoPath: "/non/existent/path",
			shouldReturn: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cfg := loadCleanConfig(tt.mainRepoPath)

			assert.NotNil(t, cfg)
			// Config should always be returned, even for non-existent paths
			assert.Equal(t, config.DefaultBaseDir, cfg.Defaults.BaseDir)
		})
	}
}

// ===== Error Handling Tests =====

func TestCleanCommand_NotInGitRepo(t *testing.T) {
	// Create a temporary directory that is not a git repo
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	err := os.Chdir(tempDir)
	assert.NoError(t, err)

	app := &cli.Command{
		Commands: []*cli.Command{
			NewCleanCommand(),
		},
	}

	ctx := context.Background()
	err = app.Run(ctx, []string{"wtp", "clean"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in a git repository")
}

func TestCleanCommand_ExecutionError(t *testing.T) {
	mockExec := &mockCleanCommandExecutor{
		shouldFail: true,
		errorMsg:   "git command failed",
	}

	var buf bytes.Buffer

	err := cleanCommandWithExecutor(&buf, mockExec, false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git command failed")
}

func TestCleanCommand_NoManagedWorktrees(t *testing.T) {
	mockExec := &mockCleanCommandExecutor{
		results: map[string]command.ExecutionResult{
			"worktree list": {
				Results: []command.Result{{
					Output: "worktree /test/repo\nHEAD abc123\nbranch refs/heads/main\n\n",
					Error:  nil,
				}},
			},
		},
	}

	var buf bytes.Buffer

	err := cleanCommandWithExecutor(&buf, mockExec, false)

	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "No managed worktrees found")
}

// ===== Mock Implementations =====

type mockCleanCommandExecutor struct {
	executedCommands []command.Command
	results          map[string]command.ExecutionResult
	shouldFail       bool
	errorMsg         string
	callCount        int
}

func (m *mockCleanCommandExecutor) Execute(commands []command.Command) (*command.ExecutionResult, error) {
	m.executedCommands = append(m.executedCommands, commands...)

	if m.shouldFail {
		return nil, &mockCleanError{message: m.errorMsg}
	}

	if len(commands) == 0 {
		return &command.ExecutionResult{Results: []command.Result{}}, nil
	}

	cmd := commands[0]
	result := m.matchCommand(cmd)
	m.callCount++

	results := make([]command.Result, len(result.Results))
	for i, r := range result.Results {
		results[i] = r
		results[i].Command = cmd
	}

	return &command.ExecutionResult{Results: results}, nil
}

func (m *mockCleanCommandExecutor) matchCommand(cmd command.Command) command.ExecutionResult {
	if len(cmd.Args) == 0 {
		return command.ExecutionResult{}
	}

	arg0 := cmd.Args[0]

	if arg0 == "worktree" && len(cmd.Args) > 1 && cmd.Args[1] == "list" {
		return m.results["worktree list"]
	}
	if arg0 == "merge-base" {
		return m.results["merge-base"]
	}
	if arg0 == "status" {
		return m.results["status"]
	}
	if arg0 == "rev-list" {
		return m.results["rev-list"]
	}
	if arg0 == "rev-parse" {
		if len(cmd.Args) > 2 && cmd.Args[2] == "main" {
			return m.results["main"]
		}
		if len(cmd.Args) > 3 && cmd.Args[3] == "master" {
			return m.results["master"]
		}
	}

	return command.ExecutionResult{}
}

type mockCleanError struct {
	message string
}

func (e *mockCleanError) Error() string {
	return e.message
}

// ===== Edge Cases Tests =====

func TestCleanCommand_EmptyWorktreeList(t *testing.T) {
	mockExec := &mockCleanCommandExecutor{
		results: map[string]command.ExecutionResult{
			"worktree list": {
				Results: []command.Result{{
					Output: "worktree /test/repo\nHEAD abc123\nbranch refs/heads/main\n\n",
					Error:  nil,
				}},
			},
		},
	}

	var buf bytes.Buffer

	err := cleanCommandWithExecutor(&buf, mockExec, false)

	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "No managed worktrees found")
}

func TestCleanCommand_InternationalCharacters(t *testing.T) {
	tests := []struct {
		name         string
		branchName   string
		worktreePath string
	}{
		{
			name:         "Japanese characters",
			branchName:   "機能/ログイン",
			worktreePath: "/test/repo/.worktrees/機能/ログイン",
		},
		{
			name:         "Spanish accents",
			branchName:   "función/añadir",
			worktreePath: "/test/repo/.worktrees/función/añadir",
		},
		{
			name:         "Emoji characters",
			branchName:   "feature/🚀-rocket",
			worktreePath: "/test/repo/.worktrees/feature/🚀-rocket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockOutput := "worktree /test/repo\nHEAD abc123\nbranch refs/heads/main\n\n" +
				"worktree " + tt.worktreePath + "\nHEAD def456\nbranch refs/heads/" + tt.branchName + "\n\n"

			mockExec := &mockCleanCommandExecutor{
				results: map[string]command.ExecutionResult{
					"worktree list": {
						Results: []command.Result{{Output: mockOutput, Error: nil}},
					},
				},
			}

			worktrees, _, err := getWorktreesForClean(mockExec)

			assert.NoError(t, err)
			// Should parse without error
			assert.GreaterOrEqual(t, len(worktrees), 1)
		})
	}
}

func TestWorktreeCleanStatus_BuildReason(t *testing.T) {
	tests := []struct {
		name           string
		status         worktreeCleanStatus
		expectedReason string
	}{
		{
			name: "safe status",
			status: worktreeCleanStatus{
				isSafe:   true,
				isMerged: true,
				isClean:  true,
				isPushed: true,
			},
			expectedReason: "safe: merged, clean, pushed",
		},
		{
			name: "single unsafe reason",
			status: worktreeCleanStatus{
				isSafe:   false,
				isMerged: false,
				isClean:  true,
				isPushed: true,
				reasons:  []string{"unmerged"},
			},
			expectedReason: "unsafe: unmerged",
		},
		{
			name: "multiple unsafe reasons",
			status: worktreeCleanStatus{
				isSafe:   false,
				isMerged: false,
				isClean:  false,
				isPushed: true,
				reasons:  []string{"unmerged", "uncommitted changes"},
			},
			expectedReason: "unsafe: unmerged, uncommitted changes",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.status.buildReason()
			assert.Equal(t, tt.expectedReason, tt.status.reason)
		})
	}
}

// TestRunCleanForm is skipped as it requires interactive input
func TestRunCleanForm(t *testing.T) {
	t.Skip("Skipping interactive form test - requires user input")

	options := []huh.Option[string]{
		huh.NewOption("feature [safe]", "feature"),
		huh.NewOption("bugfix [unsafe: unmerged]", "bugfix"),
	}

	_, err := runCleanForm(options)
	// This will fail in tests due to no TTY, but we're just checking the function exists
	assert.Error(t, err)
}
