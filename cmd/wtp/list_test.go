package main

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"runtime"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"

	"github.com/satococoa/wtp/v2/internal/command"
	"github.com/satococoa/wtp/v2/internal/config"
)

func defaultListDisplayOptionsForTests() listDisplayOptions {
	return listDisplayOptions{
		MaxPathWidth: defaultMaxPathWidth,
		OutputIsTTY:  true,
	}
}

func extractPathColumnWidth(t *testing.T, output string) int {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(output), "\n")
	if len(lines) == 0 {
		t.Fatalf("no output produced")
	}
	header := lines[0]
	idx := strings.Index(header, "BRANCH")
	if idx == -1 {
		t.Fatalf("BRANCH column missing in header: %q", header)
	}
	if idx == 0 {
		return 0
	}
	return idx - 1
}

// ===== Command Structure Tests =====

func TestNewListCommand(t *testing.T) {
	cmd := NewListCommand()

	assert.NotNil(t, cmd)
	assert.Equal(t, "list", cmd.Name)
	assert.Contains(t, cmd.Aliases, "ls")
	assert.Equal(t, "List all worktrees", cmd.Usage)
	assert.NotEmpty(t, cmd.Description)
	assert.NotNil(t, cmd.Action)
	assert.NotNil(t, cmd.ShellComplete)
}

// ===== Pure Business Logic Tests =====

func TestDisplayConstants(t *testing.T) {
	assert.Equal(t, 4, pathHeaderDashes)
	assert.Equal(t, 6, branchHeaderDashes)
	assert.Equal(t, 8, headDisplayLength)
}

func TestWorktreeFormatting(t *testing.T) {
	tests := []struct {
		name           string
		path           string
		branch         string
		head           string
		expectedFormat string
	}{
		{
			name:           "basic worktree",
			path:           "/path/to/worktree",
			branch:         "main",
			head:           "abcd1234",
			expectedFormat: "/path/to/worktree",
		},
		{
			name:           "long path",
			path:           "/very/long/path/to/worktree/that/might/need/truncation",
			branch:         "feature/test",
			head:           "efgh5678",
			expectedFormat: "/very/long/path/to/worktree/that/might/need/truncation",
		},
		{
			name:           "short head",
			path:           "/path",
			branch:         "main",
			head:           "abc123",
			expectedFormat: "/path",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test that formatting doesn't break with various inputs
			// The actual formatting logic can be tested here if needed
			assert.NotEmpty(t, tt.path)
			assert.NotEmpty(t, tt.branch)
			assert.NotEmpty(t, tt.head)
		})
	}
}

// ===== Command Execution Tests =====

func TestListCommand_CommandConstruction(t *testing.T) {
	tests := []struct {
		name             string
		mockOutput       string
		expectedCommands []command.Command
	}{
		{
			name:       "list worktrees command",
			mockOutput: "worktree /path/to/worktree\nHEAD abc123\nbranch refs/heads/main\n\n",
			expectedCommands: []command.Command{{
				Name: "git",
				Args: []string{"worktree", "list", "--porcelain"},
			}},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockListCommandExecutor{
				results: []command.Result{
					{
						Output: tt.mockOutput,
						Error:  nil,
					},
				},
			}

			var buf bytes.Buffer
			cmd := &cli.Command{}

			cfg := &config.Config{
				Defaults: config.Defaults{BaseDir: "../worktrees"},
			}
			err := listCommandWithCommandExecutor(
				cmd,
				&buf,
				mockExec,
				cfg,
				"/test/repo",
				false,
				defaultListDisplayOptionsForTests(),
			)

			assert.NoError(t, err)
			// Verify the correct git command was executed
			assert.Equal(t, tt.expectedCommands, mockExec.executedCommands)
		})
	}
}

func TestListCommand_Output(t *testing.T) {
	tests := []struct {
		name           string
		mockOutput     string
		expectedOutput []string
	}{
		{
			name:       "single worktree",
			mockOutput: "worktree /path/to/worktree\nHEAD abc123\nbranch refs/heads/main\n\n",
			expectedOutput: []string{
				"PATH",
				"BRANCH",
				"HEAD",
				"@", // Main worktree always shows as @
				"main",
				"abc123",
			},
		},
		{
			name: "multiple worktrees",
			mockOutput: "worktree /path/to/main\nHEAD abc123\nbranch refs/heads/main\n\n" +
				"worktree /path/to/feature\nHEAD def456\nbranch refs/heads/feature/test\n\n",
			expectedOutput: []string{
				"PATH",
				"BRANCH",
				"HEAD",
				"@",
				"main",
				"feature", // Relative path from current directory
				"feature/test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock current directory as /path/to
			oldGetwd := listGetwd
			listGetwd = func() (string, error) {
				return "/path/to", nil
			}
			defer func() {
				listGetwd = oldGetwd
			}()

			mockExec := &mockListCommandExecutor{
				results: []command.Result{
					{
						Output: tt.mockOutput,
						Error:  nil,
					},
				},
			}

			var buf bytes.Buffer
			cmd := &cli.Command{}

			cfg := &config.Config{
				Defaults: config.Defaults{BaseDir: "../worktrees"},
			}
			err := listCommandWithCommandExecutor(
				cmd,
				&buf,
				mockExec,
				cfg,
				"/test/repo",
				false,
				defaultListDisplayOptionsForTests(),
			)

			assert.NoError(t, err)
			output := buf.String()
			for _, expected := range tt.expectedOutput {
				assert.Contains(t, output, expected)
			}
		})
	}
}

// ===== Error Handling Tests =====

func TestListCommand_NotInGitRepo(t *testing.T) {
	// Create a temporary directory that is not a git repo
	tempDir := t.TempDir()
	oldDir, _ := os.Getwd()
	defer func() { _ = os.Chdir(oldDir) }()
	err := os.Chdir(tempDir)
	assert.NoError(t, err)

	app := &cli.Command{
		Commands: []*cli.Command{
			NewListCommand(),
		},
	}

	ctx := context.Background()
	err = app.Run(ctx, []string{"wtp", "list"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not in a git repository")
}

func TestListCommand_ExecutionError(t *testing.T) {
	mockExec := &mockListCommandExecutor{
		shouldFail: true,
		errorMsg:   "git command failed",
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	cfg := &config.Config{
		Defaults: config.Defaults{BaseDir: "../worktrees"},
	}
	err := listCommandWithCommandExecutor(
		cmd,
		&buf,
		mockExec,
		cfg,
		"/test/repo",
		false,
		defaultListDisplayOptionsForTests(),
	)

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "git command failed")
}

func TestListCommand_NoWorktrees(t *testing.T) {
	mockExec := &mockListCommandExecutor{
		results: []command.Result{
			{
				Output: "",
				Error:  nil,
			},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	cfg := &config.Config{
		Defaults: config.Defaults{BaseDir: "../worktrees"},
	}
	err := listCommandWithCommandExecutor(
		cmd,
		&buf,
		mockExec,
		cfg,
		"/test/repo",
		false,
		defaultListDisplayOptionsForTests(),
	)

	assert.NoError(t, err)
	output := buf.String()
	// Should show "No worktrees found" message when no worktrees
	assert.Contains(t, output, "No worktrees found")
}

// ===== Edge Cases Tests =====

func TestListCommand_InternationalCharacters(t *testing.T) {
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
			// Mock current directory
			oldGetwd := listGetwd
			listGetwd = func() (string, error) {
				return "/tmp", nil
			}
			defer func() {
				listGetwd = oldGetwd
			}()

			mockOutput := "worktree " + tt.worktreePath + "\nHEAD abc123\nbranch refs/heads/" + tt.branchName + "\n\n"

			mockExec := &mockListCommandExecutor{
				results: []command.Result{
					{
						Output: mockOutput,
						Error:  nil,
					},
				},
			}

			var buf bytes.Buffer
			cmd := &cli.Command{}

			cfg := &config.Config{
				Defaults: config.Defaults{BaseDir: "../worktrees"},
			}
			err := listCommandWithCommandExecutor(
				cmd,
				&buf,
				mockExec,
				cfg,
				"/test/repo",
				false,
				defaultListDisplayOptionsForTests(),
			)

			assert.NoError(t, err)
			output := buf.String()
			// Check that the branch name is displayed correctly
			assert.Contains(t, output, tt.branchName)
			// Main worktree should show as @
			assert.Contains(t, output, "@")
		})
	}
}

func TestListCommand_LongPaths(t *testing.T) {
	tests := []struct {
		name string
		path string
	}{
		{
			name: "very long path",
			path: "/very/long/path/to/worktree/that/might/cause/display/issues/in/terminal/environments/with/limited/width",
		},
		{
			name: "path with spaces",
			path: "/path/with spaces/in the/directory names",
		},
		{
			name: "path with special characters",
			path: "/path/with-special_characters.and.dots",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock a wide terminal so paths aren't truncated
			oldGetTerminalWidth := getTerminalWidth
			getTerminalWidth = func() int {
				return 200 // Wide enough to show full paths
			}
			defer func() {
				getTerminalWidth = oldGetTerminalWidth
			}()

			// Mock current directory
			oldGetwd := listGetwd
			listGetwd = func() (string, error) {
				return "/tmp", nil
			}
			defer func() {
				listGetwd = oldGetwd
			}()

			mockOutput := "worktree " + tt.path + "\nHEAD abc123\nbranch refs/heads/main\n\n"

			mockExec := &mockListCommandExecutor{
				results: []command.Result{
					{
						Output: mockOutput,
						Error:  nil,
					},
				},
			}

			var buf bytes.Buffer
			cmd := &cli.Command{}

			cfg := &config.Config{
				Defaults: config.Defaults{BaseDir: "../worktrees"},
			}
			err := listCommandWithCommandExecutor(
				cmd,
				&buf,
				mockExec,
				cfg,
				"/test/repo",
				false,
				defaultListDisplayOptionsForTests(),
			)

			assert.NoError(t, err)
			output := buf.String()
			// Main worktree should show as @
			assert.Contains(t, output, "@")
		})
	}
}

func TestListCommand_MixedWorktreeStates(t *testing.T) {
	// Mock current directory as /path/to
	oldGetwd := listGetwd
	listGetwd = func() (string, error) {
		return "/path/to", nil
	}
	defer func() {
		listGetwd = oldGetwd
	}()

	// Test with worktrees in different states (detached HEAD, etc.)
	mockOutput := `worktree /path/to/main
HEAD abc123
branch refs/heads/main

worktree /path/to/detached
HEAD def456
detached

worktree /path/to/feature
HEAD ghi789
branch refs/heads/feature/test

`

	mockExec := &mockListCommandExecutor{
		results: []command.Result{
			{
				Output: mockOutput,
				Error:  nil,
			},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	cfg := &config.Config{
		Defaults: config.Defaults{BaseDir: "../worktrees"},
	}
	err := listCommandWithCommandExecutor(
		cmd,
		&buf,
		mockExec,
		cfg,
		"/test/repo",
		false,
		defaultListDisplayOptionsForTests(),
	)

	assert.NoError(t, err)
	output := buf.String()

	// Check that all worktrees are displayed
	assert.Contains(t, output, "@")
	assert.Contains(t, output, "detached")
	assert.Contains(t, output, "feature")
	assert.Contains(t, output, "main")
	assert.Contains(t, output, "feature/test")
	// Should show "(detached HEAD)" for detached HEAD
	assert.Contains(t, output, "(detached HEAD)")
}

func TestListCommand_HeaderFormatting(t *testing.T) {
	mockExec := &mockListCommandExecutor{
		results: []command.Result{
			{
				Output: "worktree /path/to/worktree\nHEAD abc123\nbranch refs/heads/main\n\n",
				Error:  nil,
			},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	cfg := &config.Config{
		Defaults: config.Defaults{BaseDir: "../worktrees"},
	}
	err := listCommandWithCommandExecutor(
		cmd,
		&buf,
		mockExec,
		cfg,
		"/test/repo",
		false,
		defaultListDisplayOptionsForTests(),
	)

	assert.NoError(t, err)
	output := buf.String()

	// Check header formatting
	lines := strings.Split(output, "\n")
	assert.True(t, len(lines) >= 2, "Should have header and separator lines")

	// Should contain header columns
	assert.Contains(t, output, "PATH")
	assert.Contains(t, output, "BRANCH")
	assert.Contains(t, output, "HEAD")

	// Should contain separator dashes
	assert.Contains(t, output, "----")
	assert.Contains(t, output, "------")
}

// ===== Mock Implementations =====

type mockListCommandExecutor struct {
	executedCommands []command.Command
	results          []command.Result
	shouldFail       bool
	errorMsg         string
}

func (m *mockListCommandExecutor) Execute(commands []command.Command) (*command.ExecutionResult, error) {
	m.executedCommands = commands

	if m.shouldFail {
		return nil, &mockError{message: m.errorMsg}
	}

	results := make([]command.Result, len(commands))
	for i, cmd := range commands {
		if i < len(m.results) {
			results[i] = m.results[i]
		} else {
			results[i] = command.Result{
				Command: cmd,
				Output:  "",
				Error:   nil,
			}
		}
	}

	return &command.ExecutionResult{Results: results}, nil
}

func TestListCommand_DetachedHeadFormatting(t *testing.T) {
	tests := []struct {
		name           string
		mockOutput     string
		expectedBranch string
		description    string
	}{
		{
			name: "empty branch should show (no branch)",
			mockOutput: `worktree /path/to/empty
HEAD abc123

`,
			expectedBranch: "(no branch)",
			description:    "Empty branch field should display as (no branch)",
		},
		{
			name: "detached keyword should show (detached HEAD)",
			mockOutput: `worktree /path/to/detached-head
HEAD def456
detached

`,
			expectedBranch: "(detached HEAD)",
			description:    "Detached keyword should display as (detached HEAD)",
		},
		{
			name: "normal branch should show as is",
			mockOutput: `worktree /path/to/normal
HEAD ghi789
branch refs/heads/feature/awesome

`,
			expectedBranch: "feature/awesome",
			description:    "Normal branch should display as is",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mockExec := &mockListCommandExecutor{
				results: []command.Result{
					{Output: tt.mockOutput, Error: nil},
				},
			}

			var buf bytes.Buffer
			cmd := &cli.Command{}

			cfg := &config.Config{
				Defaults: config.Defaults{BaseDir: "../worktrees"},
			}
			err := listCommandWithCommandExecutor(
				cmd,
				&buf,
				mockExec,
				cfg,
				"/repo",
				false,
				defaultListDisplayOptionsForTests(),
			)

			assert.NoError(t, err, tt.description)
			output := buf.String()
			assert.Contains(t, output, tt.expectedBranch, tt.description)
		})
	}
}

type mockError struct {
	message string
}

func (e *mockError) Error() string {
	return e.message
}

func TestListCommand_RelativePathDisplay(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("TODO: Fix for Windows - test uses Unix-specific paths")
	}

	tests := []struct {
		name             string
		mockOutput       string
		currentPath      string
		expectedContains []string
		description      string
	}{
		{
			name: "main worktree should display as @",
			mockOutput: `worktree /Users/satoshi/dev/project
HEAD abc123
branch refs/heads/main

worktree /Users/satoshi/dev/project/.worktrees/feature
HEAD def456
branch refs/heads/feature/test

`,
			currentPath: "/Users/satoshi/dev/project/.worktrees/feature",
			expectedContains: []string{
				"@",
				"feature",
				"*", // Current worktree marker
			},
			description: "Main worktree should show as @ and current should have *",
		},
		{
			name: "relative paths from parent directory",
			mockOutput: `worktree /Users/satoshi/dev/project
HEAD abc123
branch refs/heads/main

worktree /Users/satoshi/dev/project-feature
HEAD def456
branch refs/heads/feature

`,
			currentPath: "/Users/satoshi/dev",
			expectedContains: []string{
				"project",
				"project-feature",
			},
			description: "Should show relative paths from current directory",
		},
		{
			name: "paths outside current directory tree",
			mockOutput: `worktree /Users/satoshi/project1
HEAD abc123
branch refs/heads/main

worktree /Users/alice/project2
HEAD def456
branch refs/heads/feature

`,
			currentPath: "/Users/satoshi/dev",
			expectedContains: []string{
				"@", // Main worktree always shows as @
				"../../alice/project2",
			},
			description: "Should show relative paths with .. for outside paths",
		},
		{
			name: "paths relative to main worktree when in subdirectory",
			mockOutput: `worktree /Users/satoshi/dev/src/github.com/satococoa/giselle
HEAD 043130cca
branch refs/heads/main

worktree /Users/satoshi/dev/src/github.com/satococoa/giselle/.worktrees/foobar
HEAD 043130cca
branch refs/heads/foobar

worktree /Users/satoshi/dev/src/github.com/satococoa/giselle/.worktrees/hoge
HEAD 043130cca
branch refs/heads/hoge

`,
			currentPath: "/Users/satoshi/dev/src/github.com/satococoa/giselle/.worktrees/foobar",
			expectedContains: []string{
				"@",
				"foobar*", // Current worktree with marker
				"hoge",    // Should be "hoge" not "../hoge" when paths are relative to base_dir
			},
			description: "Non-main worktrees should show paths relative to base_dir (.worktrees), not current directory",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock current directory
			oldGetwd := listGetwd
			listGetwd = func() (string, error) {
				return tt.currentPath, nil
			}
			defer func() {
				listGetwd = oldGetwd
			}()

			mockExec := &mockListCommandExecutor{
				results: []command.Result{
					{Output: tt.mockOutput, Error: nil},
				},
			}

			var buf bytes.Buffer
			cmd := &cli.Command{}

			// Extract main repo path from mock output
			lines := strings.Split(tt.mockOutput, "\n")
			mainRepoPath := "/Users/satoshi/dev/project" // default
			if len(lines) > 0 && strings.HasPrefix(lines[0], "worktree ") {
				mainRepoPath = strings.TrimPrefix(lines[0], "worktree ")
			}

			// Special handling for the base_dir test case
			baseDir := "../worktrees"
			if tt.name == "paths relative to main worktree when in subdirectory" {
				baseDir = ".worktrees"
				mainRepoPath = "/Users/satoshi/dev/src/github.com/satococoa/giselle"
			}

			cfg := &config.Config{
				Defaults: config.Defaults{BaseDir: baseDir},
			}
			err := listCommandWithCommandExecutor(
				cmd,
				&buf,
				mockExec,
				cfg,
				mainRepoPath,
				false,
				defaultListDisplayOptionsForTests(),
			)

			assert.NoError(t, err, tt.description)
			output := buf.String()

			// Check expected content is present
			for _, expected := range tt.expectedContains {
				assert.Contains(t, output, expected, "Expected to find: %s", expected)
			}
		})
	}
}

func TestListCommand_TerminalWidthTruncation(t *testing.T) {
	tests := []struct {
		name             string
		mockOutput       string
		terminalWidth    int
		expectedContains []string
		expectedNotFull  []string
		description      string
	}{
		{
			name: "truncate very long paths to fit terminal",
			mockOutput: `worktree /Users/satoshi/dev/src/github.com/giselles-ai/giselle
HEAD 5d46cc7a
branch refs/heads/add-github-pull-request-ingestion-table

worktree /Users/satoshi/dev/src/github.com/giselles-ai/giselle/.worktrees/stripe-basil-update
HEAD 7c81ef4f
branch refs/heads/stripe-basil-migration

`,
			terminalWidth:    80,
			expectedContains: []string{"PATH", "BRANCH", "HEAD"},
			expectedNotFull: []string{
				"/Users/satoshi/dev/src/github.com/giselles-ai/giselle/.worktrees/stripe-basil-update",
			},
			description: "Long paths should be truncated when terminal width is limited",
		},
		{
			name: "handle multiple long paths",
			mockOutput: `worktree /very/long/path/that/exceeds/normal/terminal/width/and/should/be/truncated/somehow
HEAD abc123
branch refs/heads/main

worktree /another/extremely/long/path/that/also/exceeds/terminal/width/limitations/feature
HEAD def456
branch refs/heads/feature/long-branch-name-that-might-also-be-truncated

`,
			terminalWidth:    100,
			expectedContains: []string{"PATH", "BRANCH", "HEAD"},
			expectedNotFull: []string{
				"/very/long/path/that/exceeds/normal/terminal/width/and/should/be/truncated/somehow",
				"/another/extremely/long/path/that/also/exceeds/terminal/width/limitations/feature",
			},
			description: "Multiple long paths should be handled gracefully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Mock terminal width detection
			oldGetTerminalWidth := getTerminalWidth
			getTerminalWidth = func() int {
				return tt.terminalWidth
			}
			defer func() {
				getTerminalWidth = oldGetTerminalWidth
			}()

			mockExec := &mockListCommandExecutor{
				results: []command.Result{
					{Output: tt.mockOutput, Error: nil},
				},
			}

			var buf bytes.Buffer
			cmd := &cli.Command{}

			cfg := &config.Config{
				Defaults: config.Defaults{BaseDir: "../worktrees"},
			}
			err := listCommandWithCommandExecutor(
				cmd,
				&buf,
				mockExec,
				cfg,
				"/repo",
				false,
				defaultListDisplayOptionsForTests(),
			)

			assert.NoError(t, err, tt.description)
			output := buf.String()

			// Check expected content is present
			for _, expected := range tt.expectedContains {
				assert.Contains(t, output, expected)
			}

			// Check that long paths are truncated (not displayed in full)
			for _, longPath := range tt.expectedNotFull {
				assert.NotContains(t, output, longPath, "Long path should be truncated: %s", longPath)
			}

			// Check that output fits within terminal width
			lines := strings.Split(strings.TrimSpace(output), "\n")
			for _, line := range lines {
				assert.LessOrEqual(t, len(line), tt.terminalWidth,
					"Line should not exceed terminal width: %s", line)
			}
		})
	}
}

func TestListCommand_PathColumnCappedOnWideTerminals(t *testing.T) {
	longName := strings.Repeat("verylongsegment", 5)
	mockOutput := fmt.Sprintf(`worktree /test/repo
HEAD abc123
branch refs/heads/main

worktree /test/repo/.worktrees/%s
HEAD def456
branch refs/heads/feature/long

`, longName)

	oldGetTerminalWidth := getTerminalWidth
	getTerminalWidth = func() int { return 150 }
	t.Cleanup(func() { getTerminalWidth = oldGetTerminalWidth })

	mockExec := &mockListCommandExecutor{
		results: []command.Result{{Output: mockOutput, Error: nil}},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}
	cfg := &config.Config{Defaults: config.Defaults{BaseDir: ".worktrees"}}

	err := listCommandWithCommandExecutor(
		cmd,
		&buf,
		mockExec,
		cfg,
		"/test/repo",
		false,
		defaultListDisplayOptionsForTests(),
	)
	assert.NoError(t, err)

	width := extractPathColumnWidth(t, buf.String())
	assert.Equal(t, defaultMaxPathWidth, width)
}

func TestListCommand_AutoCompactForSuperWideTerminals(t *testing.T) {
	mockOutput := `worktree /test/repo
HEAD abc123
branch refs/heads/main

worktree /test/repo/.worktrees/feature/test
HEAD def456
branch refs/heads/feature/test

`

	oldGetTerminalWidth := getTerminalWidth
	getTerminalWidth = func() int { return 200 }
	t.Cleanup(func() { getTerminalWidth = oldGetTerminalWidth })

	mockExec := &mockListCommandExecutor{
		results: []command.Result{{Output: mockOutput, Error: nil}},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}
	cfg := &config.Config{Defaults: config.Defaults{BaseDir: ".worktrees"}}

	err := listCommandWithCommandExecutor(
		cmd,
		&buf,
		mockExec,
		cfg,
		"/test/repo",
		false,
		defaultListDisplayOptionsForTests(),
	)
	assert.NoError(t, err)

	width := extractPathColumnWidth(t, buf.String())
	assert.Equal(t, len("feature/test"), width)
}

func TestListCommand_AutoCompactForNonTTY(t *testing.T) {
	mockOutput := `worktree /test/repo
HEAD abc123
branch refs/heads/main

worktree /test/repo/.worktrees/feature/test
HEAD def456
branch refs/heads/feature/test

`

	oldGetTerminalWidth := getTerminalWidth
	getTerminalWidth = func() int { return 120 }
	t.Cleanup(func() { getTerminalWidth = oldGetTerminalWidth })

	mockExec := &mockListCommandExecutor{
		results: []command.Result{{Output: mockOutput, Error: nil}},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}
	cfg := &config.Config{Defaults: config.Defaults{BaseDir: ".worktrees"}}

	opts := defaultListDisplayOptionsForTests()
	opts.OutputIsTTY = false

	err := listCommandWithCommandExecutor(
		cmd,
		&buf,
		mockExec,
		cfg,
		"/test/repo",
		false,
		opts,
	)
	assert.NoError(t, err)

	width := extractPathColumnWidth(t, buf.String())
	assert.Equal(t, len("feature/test"), width)
}

func TestListCommand_CompactFlag(t *testing.T) {
	mockOutput := `worktree /test/repo
HEAD abc123
branch refs/heads/main

worktree /test/repo/.worktrees/feature/test
HEAD def456
branch refs/heads/feature/test

`

	oldGetTerminalWidth := getTerminalWidth
	getTerminalWidth = func() int { return 120 }
	t.Cleanup(func() { getTerminalWidth = oldGetTerminalWidth })

	mockExec := &mockListCommandExecutor{
		results: []command.Result{{Output: mockOutput, Error: nil}},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}
	cfg := &config.Config{Defaults: config.Defaults{BaseDir: ".worktrees"}}

	opts := defaultListDisplayOptionsForTests()
	opts.Compact = true

	err := listCommandWithCommandExecutor(
		cmd,
		&buf,
		mockExec,
		cfg,
		"/test/repo",
		false,
		opts,
	)
	assert.NoError(t, err)

	width := extractPathColumnWidth(t, buf.String())
	assert.Equal(t, len("feature/test"), width)
}

func TestListCommand_CustomMaxPathWidth(t *testing.T) {
	longName := strings.Repeat("verylongsegment", 5)
	mockOutput := fmt.Sprintf(`worktree /test/repo
HEAD abc123
branch refs/heads/main

worktree /test/repo/.worktrees/%s
HEAD def456
branch refs/heads/feature/long

`, longName)

	oldGetTerminalWidth := getTerminalWidth
	getTerminalWidth = func() int { return 150 }
	t.Cleanup(func() { getTerminalWidth = oldGetTerminalWidth })

	mockExec := &mockListCommandExecutor{
		results: []command.Result{{Output: mockOutput, Error: nil}},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}
	cfg := &config.Config{Defaults: config.Defaults{BaseDir: ".worktrees"}}

	opts := defaultListDisplayOptionsForTests()
	opts.MaxPathWidth = 30

	err := listCommandWithCommandExecutor(
		cmd,
		&buf,
		mockExec,
		cfg,
		"/test/repo",
		false,
		opts,
	)
	assert.NoError(t, err)

	width := extractPathColumnWidth(t, buf.String())
	assert.Equal(t, 30, width)
}

// ===== Quiet Mode Tests =====

func TestListCommand_QuietMode_SingleWorktree(t *testing.T) {
	mockExec := &mockListCommandExecutor{
		results: []command.Result{
			{
				Output: "worktree /test/repo\nHEAD abc123\nbranch refs/heads/main\n\n",
				Error:  nil,
			},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	cfg := &config.Config{
		Defaults: config.Defaults{BaseDir: "../worktrees"},
	}
	err := listCommandWithCommandExecutor(
		cmd,
		&buf,
		mockExec,
		cfg,
		"/test/repo",
		true,
		defaultListDisplayOptionsForTests(),
	)

	assert.NoError(t, err)
	output := buf.String()

	// Should only contain the worktree name (@), nothing else
	assert.Equal(t, "@\n", output)
	// Should not contain headers
	assert.NotContains(t, output, "PATH")
	assert.NotContains(t, output, "BRANCH")
	assert.NotContains(t, output, "HEAD")
}

func TestListCommand_QuietMode_MultipleWorktrees(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("TODO: Fix for Windows - test uses Unix-specific paths")
	}

	mockOutput := `worktree /test/repo
HEAD abc123
branch refs/heads/main

worktree /test/repo/.worktrees/feature/test
HEAD def456
branch refs/heads/feature/test

worktree /test/repo/.worktrees/feature/another
HEAD ghi789
branch refs/heads/feature/another

`

	mockExec := &mockListCommandExecutor{
		results: []command.Result{
			{Output: mockOutput, Error: nil},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	cfg := &config.Config{
		Defaults: config.Defaults{BaseDir: ".worktrees"},
	}
	err := listCommandWithCommandExecutor(
		cmd,
		&buf,
		mockExec,
		cfg,
		"/test/repo",
		true,
		defaultListDisplayOptionsForTests(),
	)

	assert.NoError(t, err)
	output := buf.String()

	// Should contain all three worktree names, one per line
	expectedOutput := "@\nfeature/test\nfeature/another\n"
	assert.Equal(t, expectedOutput, output)

	// Should not contain headers or formatting
	assert.NotContains(t, output, "PATH")
	assert.NotContains(t, output, "BRANCH")
	assert.NotContains(t, output, "HEAD")
	assert.NotContains(t, output, "----")
}

func TestListCommand_QuietMode_NoWorktrees(t *testing.T) {
	mockExec := &mockListCommandExecutor{
		results: []command.Result{
			{
				Output: "",
				Error:  nil,
			},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	cfg := &config.Config{
		Defaults: config.Defaults{BaseDir: "../worktrees"},
	}
	err := listCommandWithCommandExecutor(
		cmd,
		&buf,
		mockExec,
		cfg,
		"/test/repo",
		true,
		defaultListDisplayOptionsForTests(),
	)

	assert.NoError(t, err)
	output := buf.String()
	// Should produce no output in quiet mode when there are no worktrees
	assert.Equal(t, "", output)
	assert.NotContains(t, output, "No worktrees found")
}

func TestListCommand_QuietMode_DetachedHead(t *testing.T) {
	mockOutput := `worktree /test/repo
HEAD abc123
branch refs/heads/main

worktree /test/repo/.worktrees/detached
HEAD def456
detached

`

	mockExec := &mockListCommandExecutor{
		results: []command.Result{
			{Output: mockOutput, Error: nil},
		},
	}

	var buf bytes.Buffer
	cmd := &cli.Command{}

	cfg := &config.Config{
		Defaults: config.Defaults{BaseDir: ".worktrees"},
	}
	err := listCommandWithCommandExecutor(
		cmd,
		&buf,
		mockExec,
		cfg,
		"/test/repo",
		true,
		defaultListDisplayOptionsForTests(),
	)

	assert.NoError(t, err)
	output := buf.String()

	// Should only contain worktree names, not branch state information
	expectedOutput := "@\ndetached\n"
	assert.Equal(t, expectedOutput, output)
	assert.NotContains(t, output, "detached HEAD")
	assert.NotContains(t, output, "BRANCH")
	assert.NotContains(t, output, "HEAD")
}

func TestCompleteList_SuggestsQuietFlag(t *testing.T) {
	t.Run("suggests quiet flag alias", func(t *testing.T) {
		originalArgs := os.Args
		t.Cleanup(func() { os.Args = originalArgs })

		os.Args = []string{"wtp", "list", "--q", "--generate-shell-completion"}

		var buf bytes.Buffer
		cmd := NewListCommand()
		cmd.Writer = &buf

		cmd.ShellComplete(context.Background(), cmd)

		assert.Contains(t, buf.String(), "--quiet")
	})
}
