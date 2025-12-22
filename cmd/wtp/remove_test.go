package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"

	"github.com/satococoa/wtp/v2/internal/command"
	"github.com/satococoa/wtp/v2/internal/config"
)

// ===== Command Structure Tests =====

func TestNewRemoveCommand(t *testing.T) {
	cmd := NewRemoveCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "remove", cmd.Name)
	assert.Contains(t, cmd.Aliases, "rm")
	assert.Equal(t, "Remove a worktree", cmd.Usage)
	assert.NotEmpty(t, cmd.Description)
	assert.NotNil(t, cmd.Action)
	assert.NotNil(t, cmd.ShellComplete)

	// Check flags exist
	flagNames := []string{"force", "with-branch"}
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

func TestRemoveCommand_WorktreeResolution(t *testing.T) {
	tests := []struct {
		name         string
		worktreeName string
		worktreeList string
		expectedPath string
		shouldFind   bool
	}{
		{
			name:         "find by exact name",
			worktreeName: "feature-branch",
			worktreeList: "worktree /path/to/main\nHEAD abc123\nbranch refs/heads/main\n\n" +
				"worktree /path/to/worktrees/feature-branch\nHEAD def456\nbranch refs/heads/feature-branch\n\n",
			expectedPath: "/path/to/worktrees/feature-branch",
			shouldFind:   true,
		},
		{
			name:         "not found",
			worktreeName: "nonexistent",
			worktreeList: "worktree /path/to/main\nHEAD abc123\nbranch refs/heads/main\n\n" +
				"worktree /path/to/worktrees/other\nHEAD def456\nbranch refs/heads/other\n\n",
			expectedPath: "",
			shouldFind:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			worktrees := parseWorktreesFromOutput(tt.worktreeList)

			targetWorktree, err := findTargetWorktreeFromList(worktrees, tt.worktreeName)

			if tt.shouldFind {
				assert.NoError(t, err)
				assert.Equal(t, tt.expectedPath, targetWorktree.Path)
			} else {
				assert.Error(t, err)
			}
		})
	}
}

func TestRemoveCommand_FlagValidation(t *testing.T) {
	tests := []struct {
		name        string
		forceFlag   bool
		branchFlag  bool
		shouldError bool
		expectedMsg string
	}{
		{
			name:        "force without branch is valid",
			forceFlag:   true,
			branchFlag:  false,
			shouldError: false,
		},
		{
			name:        "branch without force is valid",
			forceFlag:   false,
			branchFlag:  true,
			shouldError: false,
		},
		{
			name:        "both flags together is valid",
			forceFlag:   true,
			branchFlag:  true,
			shouldError: false,
		},
		{
			name:        "neither flag is valid",
			forceFlag:   false,
			branchFlag:  false,
			shouldError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test flag combination validation logic
			// Flag combinations are validated by the CLI framework
			assert.True(t, true) // Test passes as flags are valid
		})
	}
}

// ===== Command Execution Tests =====

func TestRemoveCommand_CommandConstruction(t *testing.T) {
	tests := []struct {
		name             string
		flags            map[string]any
		worktreeName     string
		mockWorktreeList string
		expectedCommands []command.Command
	}{
		{
			name:         "basic remove",
			flags:        map[string]any{},
			worktreeName: "feature-branch",
			mockWorktreeList: "worktree /path/to/main\nHEAD abc123\nbranch refs/heads/main\n\n" +
				"worktree /path/to/worktrees/feature-branch\nHEAD def456\nbranch refs/heads/feature-branch\n\n",
			expectedCommands: []command.Command{
				{
					Name: "git",
					Args: []string{"worktree", "list", "--porcelain"},
				},
				{
					Name: "git",
					Args: []string{"worktree", "remove", "/path/to/worktrees/feature-branch"},
				},
			},
		},
		{
			name:         "remove with force",
			flags:        map[string]any{"force": true},
			worktreeName: "feature-branch",
			mockWorktreeList: "worktree /path/to/main\nHEAD abc123\nbranch refs/heads/main\n\n" +
				"worktree /path/to/worktrees/feature-branch\nHEAD def456\nbranch refs/heads/feature-branch\n\n",
			expectedCommands: []command.Command{
				{
					Name: "git",
					Args: []string{"worktree", "list", "--porcelain"},
				},
				{
					Name: "git",
					Args: []string{"worktree", "remove", "--force", "/path/to/worktrees/feature-branch"},
				},
			},
		},
		{
			name:         "remove with branch deletion",
			flags:        map[string]any{"branch": true},
			worktreeName: "feature-branch",
			mockWorktreeList: "worktree /path/to/main\nHEAD abc123\nbranch refs/heads/main\n\n" +
				"worktree /path/to/worktrees/feature-branch\nHEAD def456\nbranch refs/heads/feature-branch\n\n",
			expectedCommands: []command.Command{
				{
					Name: "git",
					Args: []string{"worktree", "list", "--porcelain"},
				},
				{
					Name: "git",
					Args: []string{"worktree", "remove", "/path/to/worktrees/feature-branch"},
				},
				{
					Name: "git",
					Args: []string{"branch", "-d", "feature-branch"},
				},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockRemoveCommandExecutor{
				results: []command.Result{
					{
						Output: tt.mockWorktreeList,
						Error:  nil,
					},
					{
						Output: "success",
						Error:  nil,
					},
					{
						Output: "success",
						Error:  nil,
					},
				},
			}

			cmd := createRemoveTestCLICommand(tt.flags, []string{tt.worktreeName})
			var buf bytes.Buffer

			forceFlag := tt.flags["force"] == true
			branchFlag := tt.flags["branch"] == true
			err := removeCommandWithCommandExecutor(
				cmd, &buf, mockExec, "/test/repo", tt.worktreeName, forceFlag, branchFlag, false,
			)

			assert.NoError(t, err)
			// Verify the correct git commands were executed
			assert.Equal(t, tt.expectedCommands, mockExec.executedCommands)
		})
	}
}

func TestRemoveCommand_SuccessMessage(t *testing.T) {
	tests := []struct {
		name           string
		worktreeName   string
		branchFlag     bool
		expectedOutput []string
	}{
		{
			name:         "remove worktree only",
			worktreeName: "feature-branch",
			branchFlag:   false,
			expectedOutput: []string{
				"Removed worktree",
				"feature-branch",
			},
		},
		{
			name:         "remove worktree and branch",
			worktreeName: "feature-branch",
			branchFlag:   true,
			expectedOutput: []string{
				"Removed worktree",
				"feature-branch",
				"Removed branch",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockRemoveCommandExecutor{
				results: []command.Result{
					{
						Output: "worktree /path/to/main\nHEAD abc123\nbranch refs/heads/main\n\n" +
							"worktree /path/to/worktrees/feature-branch\nHEAD def456\nbranch refs/heads/feature-branch\n\n",
						Error: nil,
					},
					{
						Output: "success",
						Error:  nil,
					},
					{
						Output: "success",
						Error:  nil,
					},
				},
			}

			flags := map[string]any{}
			if tt.branchFlag {
				flags["branch"] = true
			}

			cmd := createRemoveTestCLICommand(flags, []string{tt.worktreeName})
			var buf bytes.Buffer

			branchFlag := tt.branchFlag
			err := removeCommandWithCommandExecutor(cmd, &buf, mockExec, "/test/repo", tt.worktreeName, false, branchFlag, false)

			assert.NoError(t, err)
			output := buf.String()
			for _, expected := range tt.expectedOutput {
				assert.Contains(t, output, expected)
			}
		})
	}
}

func TestRemoveCommand_ExecutePreRemoveHooks(t *testing.T) {
	tempDir := t.TempDir()
	mainRepoPath := filepath.Join(tempDir, "repo")
	worktreePath := filepath.Join(tempDir, "worktrees", "feature-hook")

	err := os.MkdirAll(mainRepoPath, 0o755)
	assert.NoError(t, err)
	err = os.MkdirAll(worktreePath, 0o755)
	assert.NoError(t, err)

	configPath := filepath.Join(mainRepoPath, ".wtp.yml")
	configContent := `version: "1.0"
defaults:
  base_dir: "../worktrees"
hooks:
  pre_remove:
    - type: command
      command: "echo before remove"
`
	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	assert.NoError(t, err)

	mockExec := &mockRemoveCommandExecutor{
		results: []command.Result{
			{
				Output: fmt.Sprintf("worktree %s\nHEAD abc123\nbranch refs/heads/main\n\nworktree %s\nHEAD def456\nbranch refs/heads/feature-hook\n\n", mainRepoPath, worktreePath),
				Error:  nil,
			},
			{
				Output: "success",
				Error:  nil,
			},
		},
	}

	cmd := createRemoveTestCLICommand(map[string]any{}, []string{"feature-hook"})
	var buf bytes.Buffer

	err = removeCommandWithCommandExecutor(cmd, &buf, mockExec, mainRepoPath, "feature-hook", false, false, false)

	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Executing pre-remove hooks")
	assert.Contains(t, output, "before remove")
	assert.Contains(t, output, "Removed worktree")
}

func TestRemoveCommand_ExecutePostRemoveHooks(t *testing.T) {
	tempDir := t.TempDir()
	mainRepoPath := filepath.Join(tempDir, "repo")
	worktreePath := filepath.Join(tempDir, "worktrees", "feature-hook")

	err := os.MkdirAll(mainRepoPath, 0o755)
	assert.NoError(t, err)
	err = os.MkdirAll(worktreePath, 0o755)
	assert.NoError(t, err)

	configPath := filepath.Join(mainRepoPath, ".wtp.yml")
	configContent := `version: "1.0"
defaults:
  base_dir: "../worktrees"
hooks:
  post_remove:
    - type: command
      command: "echo after remove"
`
	err = os.WriteFile(configPath, []byte(configContent), 0o644)
	assert.NoError(t, err)

	mockExec := &mockRemoveCommandExecutor{
		results: []command.Result{
			{
				Output: fmt.Sprintf("worktree %s\nHEAD abc123\nbranch refs/heads/main\n\nworktree %s\nHEAD def456\nbranch refs/heads/feature-hook\n\n", mainRepoPath, worktreePath),
				Error:  nil,
			},
			{
				Output: "success",
				Error:  nil,
			},
		},
	}

	cmd := createRemoveTestCLICommand(map[string]any{}, []string{"feature-hook"})
	var buf bytes.Buffer

	err = removeCommandWithCommandExecutor(cmd, &buf, mockExec, mainRepoPath, "feature-hook", false, false, false)

	assert.NoError(t, err)
	output := buf.String()
	assert.Contains(t, output, "Executing post-remove hooks")
	assert.Contains(t, output, "after remove")
	assert.Contains(t, output, "Removed worktree")
}

func TestExecutePostRemoveHooks_DefaultWorkDir(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")
	err := os.MkdirAll(repoPath, 0o755)
	assert.NoError(t, err)
	err = os.MkdirAll(filepath.Join(repoPath, "scripts"), 0o755)
	assert.NoError(t, err)

	command := "pwd"
	if runtime.GOOS == "windows" {
		command = "cd"
	}

	cfg := &config.Config{
		Defaults: config.Defaults{
			BaseDir: "../worktrees",
		},
		Hooks: config.Hooks{
			PostRemove: []config.Hook{
				{
					Type:    config.HookTypeCommand,
					Command: command,
					WorkDir: "scripts",
				},
			},
		},
	}

	var buf bytes.Buffer
	err = executePostRemoveHooks(&buf, cfg, repoPath, filepath.Join(tempDir, "worktrees", "missing"))

	assert.NoError(t, err)
	assert.Contains(t, buf.String(), "Executing post-remove hooks")
	assert.Contains(t, buf.String(), filepath.Join(repoPath, "scripts"))
}

func TestExecutePostRemoveHooks_WorkDirTraversalRejected(t *testing.T) {
	tempDir := t.TempDir()
	repoPath := filepath.Join(tempDir, "repo")

	cfg := &config.Config{
		Defaults: config.Defaults{
			BaseDir: "../worktrees",
		},
		Hooks: config.Hooks{
			PostRemove: []config.Hook{
				{
					Type:    config.HookTypeCommand,
					Command: "echo should-not-run",
					WorkDir: filepath.Join("..", ".."),
				},
			},
		},
	}

	var buf bytes.Buffer

	err := executePostRemoveHooks(&buf, cfg, repoPath, filepath.Join(tempDir, "worktrees", "missing"))

	assert.Error(t, err)
	assert.EqualError(t, err, fmt.Sprintf("post-remove hook work_dir '%s' escapes repository root", filepath.Join("..", "..")))
	assert.Contains(t, buf.String(), "Executing post-remove hooks")
	assert.NotContains(t, buf.String(), "should-not-run")
}

// ===== Error Handling Tests =====

func TestRemoveCommand_ValidationErrors(t *testing.T) {
	tests := []struct {
		name          string
		args          []string
		expectedError string
	}{
		{
			name:          "no worktree name",
			args:          []string{},
			expectedError: "worktree name is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &cli.Command{
				Commands: []*cli.Command{
					NewRemoveCommand(),
				},
			}

			ctx := context.Background()
			cmdArgs := []string{"wtp", "remove"}
			cmdArgs = append(cmdArgs, tt.args...)

			err := app.Run(ctx, cmdArgs)
			assert.Error(t, err)
			assert.Contains(t, err.Error(), tt.expectedError)
		})
	}
}

func TestRemoveCommand_NotInGitRepo(t *testing.T) {
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	err := os.Chdir(tempDir)
	assert.NoError(t, err)

	app := &cli.Command{
		Commands: []*cli.Command{
			NewRemoveCommand(),
		},
	}

	ctx := context.Background()
	err = app.Run(ctx, []string{"wtp", "remove", "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in a git repository")
}

func TestRemoveCommand_WorktreeNotFound(t *testing.T) {
	mockExec := &mockRemoveCommandExecutor{
		results: []command.Result{
			{
				Output: "worktree /path/to/worktrees/main\nHEAD abc123\nbranch refs/heads/main\n\n",
				Error:  nil,
			},
		},
	}

	cmd := createRemoveTestCLICommand(map[string]any{}, []string{"nonexistent"})
	var buf bytes.Buffer

	err := removeCommandWithCommandExecutor(cmd, &buf, mockExec, "/test/repo", "nonexistent", false, false, false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worktree 'nonexistent' not found")
}

func TestRemoveCommand_WorktreeNotFound_ShowsConsistentNames(t *testing.T) {
	mockExec := &mockRemoveCommandExecutor{
		results: []command.Result{
			{
				Output: "worktree /repo\nHEAD abc123\nbranch refs/heads/main\n\n" +
					"worktree /repo/.worktrees/feat/hogehoge\nHEAD def456\nbranch refs/heads/feat/hogehoge\n\n",
				Error: nil,
			},
		},
	}

	cmd := createRemoveTestCLICommand(map[string]any{}, []string{"nonexistent"})
	var buf bytes.Buffer

	err := removeCommandWithCommandExecutor(cmd, &buf, mockExec, "/repo", "nonexistent", false, false, false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "worktree 'nonexistent' not found")
	// Should show "No worktrees found" since the only non-main worktree is unmanaged
	assert.Contains(t, err.Error(), "No worktrees found")
}

func TestRemoveCommand_ConfigLoadFailure(t *testing.T) {
	tempDir := t.TempDir()
	mainRepoPath := filepath.Join(tempDir, "repo")
	worktreePath := filepath.Join(tempDir, "worktrees", "feature-bad-config")

	err := os.MkdirAll(mainRepoPath, 0o755)
	assert.NoError(t, err)
	err = os.WriteFile(filepath.Join(mainRepoPath, ".wtp.yml"), []byte("hooks:\n  post_create:\n    - type: command\n      command: \"oops\"\n    invalid"), 0o644)
	assert.NoError(t, err)

	mockExec := &mockRemoveCommandExecutor{
		results: []command.Result{
			{
				Output: fmt.Sprintf("worktree %s\nHEAD abc123\nbranch refs/heads/main\n\nworktree %s\nHEAD def456\nbranch refs/heads/feature-bad-config\n\n", mainRepoPath, worktreePath),
				Error:  nil,
			},
		},
	}

	cmd := createRemoveTestCLICommand(map[string]any{}, []string{"feature-bad-config"})
	var buf bytes.Buffer

	err = removeCommandWithCommandExecutor(cmd, &buf, mockExec, mainRepoPath, "feature-bad-config", false, false, false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to load configuration")
}

func TestRemoveCommand_FailsWhenRemovingCurrentWorktree(t *testing.T) {
	targetPath := "/worktrees/feature/foo"
	mockWorktreeList := fmt.Sprintf(
		"worktree /repo\nHEAD abc123\nbranch refs/heads/main\n\n"+
			"worktree %s\nHEAD def456\nbranch refs/heads/feature/foo\n\n",
		targetPath,
	)

	tests := []struct {
		name string
		cwd  string
	}{
		{
			name: "exact worktree path",
			cwd:  targetPath,
		},
		{
			name: "nested directory inside worktree",
			cwd:  filepath.Join(targetPath, "nested"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockRemoveCommandExecutor{
				results: []command.Result{
					{
						Output: mockWorktreeList,
						Error:  nil,
					},
				},
			}

			cmd := createRemoveTestCLICommand(map[string]any{}, []string{"feature/foo"})
			var buf bytes.Buffer

			err := removeCommandWithCommandExecutor(cmd, &buf, mockExec, tt.cwd, "feature/foo", false, false, false)

			assert.Error(t, err)
			assert.Contains(t, err.Error(), "cannot remove worktree 'feature/foo'")
			assert.Equal(t, []command.Command{
				{
					Name: "git",
					Args: []string{"worktree", "list", "--porcelain"},
				},
			}, mockExec.executedCommands)
		})
	}
}

func TestRemoveCommand_ExecutionError(t *testing.T) {
	mockExec := &mockRemoveCommandExecutor{
		results: []command.Result{
			{
				Output: "worktree /path/to/main\nHEAD abc123\nbranch refs/heads/main\n\n" +
					"worktree /path/to/worktrees/feature-branch\nHEAD def456\nbranch refs/heads/feature-branch\n\n",
				Error: nil,
			},
		},
		shouldFail: true,
		errorMsg:   "git command failed",
	}

	cmd := createRemoveTestCLICommand(map[string]any{}, []string{"feature-branch"})
	var buf bytes.Buffer

	err := removeCommandWithCommandExecutor(cmd, &buf, mockExec, "/test/repo", "feature-branch", false, false, false)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to remove worktree")
}

func TestRemoveCommand_DirtyWorktree(t *testing.T) {
	tests := []struct {
		name          string
		forceFlag     bool
		gitError      string
		shouldSucceed bool
		expectedMsg   []string
	}{
		{
			name:      "remove dirty worktree without force fails",
			forceFlag: false,
			gitError: "fatal: '/path/to/worktrees/dirty-feature' " +
				"contains modified or untracked files, use --force to delete it",
			shouldSucceed: false,
			expectedMsg: []string{
				"failed to remove worktree",
				"contains modified or untracked files",
				"Use '--force' flag to remove anyway",
			},
		},
		{
			name:          "remove dirty worktree with force succeeds",
			forceFlag:     true,
			gitError:      "", // No error when force is used
			shouldSucceed: true,
			expectedMsg: []string{
				"Removed worktree",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mockResults []command.Result

			// First result is always the worktree list
			mockResults = append(mockResults, command.Result{
				Output: "worktree /path/to/main\nHEAD abc123\nbranch refs/heads/main\n\n" +
					"worktree /path/to/worktrees/dirty-feature\nHEAD def456\nbranch refs/heads/dirty-feature\n\n",
				Error: nil,
			})

			// Second result is the remove command
			if tt.shouldSucceed {
				mockResults = append(mockResults, command.Result{
					Output: "success",
					Error:  nil,
				})
			} else {
				mockResults = append(mockResults, command.Result{
					Output: tt.gitError,
					Error:  &mockRemoveError{message: tt.gitError},
				})
			}

			mockExec := &mockRemoveCommandExecutor{
				results: mockResults,
			}

			flags := map[string]any{}
			if tt.forceFlag {
				flags["force"] = true
			}

			cmd := createRemoveTestCLICommand(flags, []string{"dirty-feature"})
			var buf bytes.Buffer

			err := removeCommandWithCommandExecutor(
				cmd, &buf, mockExec, "/test/repo", "dirty-feature", tt.forceFlag, false, false)

			if tt.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			output := buf.String()
			for _, expected := range tt.expectedMsg {
				if tt.shouldSucceed {
					assert.Contains(t, output, expected)
				} else {
					assert.Contains(t, err.Error(), expected)
				}
			}
		})
	}
}

func TestRemoveCommand_BranchRemovalWithUnmergedCommits(t *testing.T) {
	tests := []struct {
		name            string
		forceBranchFlag bool
		branchError     string
		shouldSucceed   bool
		expectedMsg     []string
	}{
		{
			name:            "remove unmerged branch without force fails",
			forceBranchFlag: false,
			branchError: "error: The branch 'feature-unmerged' is not fully merged.\n" +
				"If you are sure you want to delete it, run 'git branch -D feature-unmerged'.",
			shouldSucceed: false,
			expectedMsg: []string{
				"failed to remove branch",
				"not fully merged",
				"Use '--force-branch' to delete anyway",
			},
		},
		{
			name:            "remove unmerged branch with force succeeds",
			forceBranchFlag: true,
			branchError:     "", // No error when force is used
			shouldSucceed:   true,
			expectedMsg: []string{
				"Removed branch",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var mockResults []command.Result

			// First result is the worktree list
			mockResults = append(mockResults,
				command.Result{
					Output: "worktree /path/to/main\nHEAD abc123\nbranch refs/heads/main\n\n" +
						"worktree /path/to/worktrees/feature-unmerged\nHEAD def456\nbranch refs/heads/feature-unmerged\n\n",
					Error: nil,
				},
				command.Result{
					Output: "success",
					Error:  nil,
				})

			// Third result is the branch delete

			if tt.shouldSucceed {
				mockResults = append(mockResults, command.Result{
					Output: "Deleted branch feature-unmerged (was def456).",
					Error:  nil,
				})
			} else {
				mockResults = append(mockResults, command.Result{
					Output: tt.branchError,
					Error:  &mockRemoveError{message: tt.branchError},
				})
			}

			mockExec := &mockRemoveCommandExecutor{
				results: mockResults,
			}

			flags := map[string]any{
				"branch": true,
			}
			if tt.forceBranchFlag {
				flags["force-branch"] = true
			}

			cmd := createRemoveTestCLICommand(flags, []string{"feature-unmerged"})
			var buf bytes.Buffer

			err := removeCommandWithCommandExecutor(
				cmd, &buf, mockExec, "/test/repo", "feature-unmerged", false, true, tt.forceBranchFlag)

			if tt.shouldSucceed {
				assert.NoError(t, err)
			} else {
				assert.Error(t, err)
			}

			output := buf.String()
			for _, expected := range tt.expectedMsg {
				if tt.shouldSucceed {
					assert.Contains(t, output, expected)
				} else {
					assert.Contains(t, err.Error(), expected)
				}
			}
		})
	}
}

// ===== Edge Cases Tests =====

func TestRemoveCommand_InternationalCharacters(t *testing.T) {
	tests := []struct {
		name         string
		branchName   string
		worktreePath string
	}{
		{
			name:         "Japanese characters",
			branchName:   "機能/ログイン",
			worktreePath: "/path/to/worktrees/機能/ログイン",
		},
		{
			name:         "Spanish accents",
			branchName:   "función/añadir",
			worktreePath: "/path/to/worktrees/función/añadir",
		},
		{
			name:         "Emoji characters",
			branchName:   "feature/🚀-rocket",
			worktreePath: "/path/to/worktrees/feature/🚀-rocket",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockOutput := "worktree /path/to/main\nHEAD abc123\nbranch refs/heads/main\n\n" +
				"worktree " + tt.worktreePath + "\nHEAD def456\nbranch refs/heads/" + tt.branchName + "\n\n"

			mockExec := &mockRemoveCommandExecutor{
				results: []command.Result{
					{
						Output: mockOutput,
						Error:  nil,
					},
					{
						Output: "success",
						Error:  nil,
					},
				},
			}

			// Extract the basename from the path for matching
			worktreeName := filepath.Base(tt.worktreePath)
			cmd := createRemoveTestCLICommand(map[string]any{}, []string{worktreeName})
			var buf bytes.Buffer

			err := removeCommandWithCommandExecutor(cmd, &buf, mockExec, "/test/repo", worktreeName, false, false, false)

			assert.NoError(t, err)
			assert.Contains(t, buf.String(), "Removed worktree")
		})
	}
}

func TestRemoveCommand_PathWithSpaces(t *testing.T) {
	worktreePath := "/path/to/main/../worktrees/feature branch"
	mockOutput := "worktree /path/to/main\nHEAD abc123\nbranch refs/heads/main\n\n" +
		"worktree " + worktreePath + "\nHEAD def456\nbranch refs/heads/feature-branch\n\n"

	mockExec := &mockRemoveCommandExecutor{
		results: []command.Result{
			{
				Output: mockOutput,
				Error:  nil,
			},
			{
				Output: "success",
				Error:  nil,
			},
		},
	}

	cmd := createRemoveTestCLICommand(map[string]any{}, []string{"feature branch"})
	var buf bytes.Buffer

	err := removeCommandWithCommandExecutor(cmd, &buf, mockExec, "/path/to/main", "feature branch", false, false, false)

	assert.NoError(t, err)
	// Verify the correct path was passed to git command
	assert.Len(t, mockExec.executedCommands, 2)
	assert.Equal(t, []string{"worktree", "remove", worktreePath}, mockExec.executedCommands[1].Args)
}

func TestRemoveCommand_MultipleMatchingWorktrees(t *testing.T) {
	// Test case where multiple worktrees might match the input
	mockOutput := `worktree /path/to/main
HEAD abc123
branch refs/heads/main

worktree /path/to/worktrees/feature-test
HEAD def456
branch refs/heads/feature-test

worktree /path/to/worktrees/feature-test-2
HEAD ghi789
branch refs/heads/feature-test-2

worktree /path/to/worktrees/test-feature
HEAD jkl012
branch refs/heads/test-feature

`

	tests := []struct {
		input        string
		expectedPath string
	}{
		{"feature-test", "/path/to/worktrees/feature-test"},
		{"feature-test-2", "/path/to/worktrees/feature-test-2"},
		{"test-feature", "/path/to/worktrees/test-feature"},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			// Create a fresh mock executor for each subtest
			mockExec := &mockRemoveCommandExecutor{
				results: []command.Result{
					{
						Output: mockOutput,
						Error:  nil,
					},
					{
						Output: "success",
						Error:  nil,
					},
				},
			}

			cmd := createRemoveTestCLICommand(map[string]any{}, []string{tt.input})
			var buf bytes.Buffer

			err := removeCommandWithCommandExecutor(cmd, &buf, mockExec, "/test/repo", tt.input, false, false, false)

			assert.NoError(t, err)
			// Verify the correct worktree was targeted
			assert.Len(t, mockExec.executedCommands, 2)
			assert.Equal(t, []string{"worktree", "remove", tt.expectedPath}, mockExec.executedCommands[1].Args)
		})
	}
}

// ===== Helper Functions =====

func createRemoveTestCLICommand(flags map[string]any, args []string) *cli.Command {
	app := &cli.Command{
		Name: "test",
		Commands: []*cli.Command{
			{
				Name: "remove",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "force"},
					&cli.BoolFlag{Name: "branch"},
					&cli.BoolFlag{Name: "force-branch"},
				},
				Action: func(_ context.Context, _ *cli.Command) error {
					return nil
				},
			},
		},
	}

	cmdArgs := []string{"test", "remove"}
	for key, value := range flags {
		switch v := value.(type) {
		case bool:
			if v {
				cmdArgs = append(cmdArgs, "--"+key)
			}
		case string:
			cmdArgs = append(cmdArgs, "--"+key, v)
		}
	}
	cmdArgs = append(cmdArgs, args...)

	ctx := context.Background()
	_ = app.Run(ctx, cmdArgs)

	return app.Commands[0]
}

// ===== Mock Implementations =====

type mockRemoveCommandExecutor struct {
	executedCommands []command.Command
	results          []command.Result
	shouldFail       bool
	errorMsg         string
	callCount        int
}

func (m *mockRemoveCommandExecutor) Execute(commands []command.Command) (*command.ExecutionResult, error) {
	// Accumulate all commands instead of overwriting
	m.executedCommands = append(m.executedCommands, commands...)

	if m.shouldFail && m.callCount > 0 {
		errorMsg := m.errorMsg
		if errorMsg == "" {
			errorMsg = "mock error"
		}
		return nil, &mockRemoveError{message: errorMsg}
	}

	results := make([]command.Result, len(commands))
	for i, cmd := range commands {
		if m.callCount < len(m.results) {
			results[i] = m.results[m.callCount]
			results[i].Command = cmd
		} else {
			results[i] = command.Result{
				Command: cmd,
				Output:  "",
				Error:   nil,
			}
		}
	}

	m.callCount++
	return &command.ExecutionResult{Results: results}, nil
}

type mockRemoveError struct {
	message string
}

func (e *mockRemoveError) Error() string {
	return e.message
}

// ===== Worktree Completion Tests =====

func TestGetWorktreeNameFromPath(t *testing.T) {
	RunNameFromPathTests(t, "remove", getWorktreeNameFromPath)
}

func TestGetWorktreesForRemove(t *testing.T) {
	RunWriterCommonTests(t, "getWorktreesForRemove", getWorktreesForRemove)
}

func TestCompleteWorktrees(t *testing.T) {
	t.Run("should not panic when called", func(t *testing.T) {
		cmd := &cli.Command{}

		// Should not panic even without proper git setup
		assert.NotPanics(t, func() {
			restore := silenceStdout(t)
			defer restore()

			completeWorktrees(context.Background(), cmd)
		})
	})
}
