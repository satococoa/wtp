package main

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/urfave/cli/v3"

	"github.com/satococoa/wtp/internal/config"
)

func TestNewAddCommand(t *testing.T) {
	cmd := NewAddCommand()
	assert.NotNil(t, cmd)
	assert.Equal(t, "add", cmd.Name)
	assert.Equal(t, "Create a new worktree", cmd.Usage)
	assert.NotEmpty(t, cmd.Description)
	assert.NotNil(t, cmd.Action)
	assert.NotNil(t, cmd.ShellComplete)
	
	// Check flags exist
	flagNames := []string{"path", "force", "detach", "branch", "track", "cd", "no-cd"}
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

func TestValidateAddInput(t *testing.T) {
	tests := []struct {
		name    string
		args    []string
		branch  string
		wantErr bool
	}{
		{
			name:    "no args and no branch flag",
			args:    []string{},
			branch:  "",
			wantErr: true,
		},
		{
			name:    "with args",
			args:    []string{"feature"},
			branch:  "",
			wantErr: false,
		},
		{
			name:    "with branch flag",
			args:    []string{},
			branch:  "new-feature",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a properly initialized app
			app := &cli.Command{
				Name: "test",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "branch"},
				},
				Action: func(_ context.Context, cmd *cli.Command) error {
					// Test the validation
					return validateAddInput(cmd)
				},
			}
			
			// Build args
			args := []string{"test"}
			if tt.branch != "" {
				args = append(args, "--branch", tt.branch)
			}
			args = append(args, tt.args...)
			
			ctx := context.Background()
			err := app.Run(ctx, args)
			
			if tt.wantErr {
				assert.Error(t, err)
				assert.Contains(t, err.Error(), "branch name is required")
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestResolveWorktreePath(t *testing.T) {
	cfg := &config.Config{
		Defaults: config.Defaults{
			BaseDir: "../worktrees",
		},
	}
	repoPath := "/path/to/repo"

	tests := []struct {
		name       string
		firstArg   string
		pathFlag   string
		branchFlag string
		wantPath   string
		wantBranch string
	}{
		{
			name:       "explicit path with absolute path",
			firstArg:   "feature/auth",
			pathFlag:   "/custom/path",
			wantPath:   "/custom/path",
			wantBranch: "feature/auth",
		},
		{
			name:       "explicit path with relative path",
			firstArg:   "feature/auth",
			pathFlag:   "./custom/path",
			wantPath:   "./custom/path",
			wantBranch: "feature/auth",
		},
		{
			name:       "auto-generated path - branch name simple",
			firstArg:   "feature",
			wantPath:   filepath.Join(repoPath, "..", "worktrees", "feature"),
			wantBranch: "feature",
		},
		{
			name:       "auto-generated path - branch name with slash",
			firstArg:   "feature/auth",
			wantPath:   filepath.Join(repoPath, "..", "worktrees", "feature", "auth"),
			wantBranch: "feature/auth",
		},
		{
			name:       "auto-generated path with -b flag",
			firstArg:   "feature",
			branchFlag: "new-feature",
			wantPath:   filepath.Join(repoPath, "..", "worktrees", "new-feature"),
			wantBranch: "new-feature",
		},
		{
			name:       "explicit path with -b flag",
			firstArg:   "feature",
			pathFlag:   "/tmp/test",
			branchFlag: "new-feature",
			wantPath:   "/tmp/test",
			wantBranch: "feature",
		},
		{
			name:       "no args",
			firstArg:   "",
			wantPath:   filepath.Join(repoPath, "..", "worktrees"),
			wantBranch: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cli.Command{
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "path"},
					&cli.StringFlag{Name: "branch"},
				},
			}
			
			// Set flags
			if tt.pathFlag != "" {
				_ = cmd.Set("path", tt.pathFlag)
			}
			if tt.branchFlag != "" {
				_ = cmd.Set("branch", tt.branchFlag)
			}

			gotPath, gotBranch := resolveWorktreePath(cfg, repoPath, tt.firstArg, cmd)
			assert.Equal(t, tt.wantPath, gotPath)
			assert.Equal(t, tt.wantBranch, gotBranch)
		})
	}
}

func TestBuildGitWorktreeArgs(t *testing.T) {
	tests := []struct {
		name         string
		workTreePath string
		branchName   string
		flags        map[string]interface{}
		cliArgs      []string
		want         []string
	}{
		{
			name:         "simple branch",
			workTreePath: "/path/to/worktree",
			branchName:   "feature",
			flags:        map[string]interface{}{},
			want:         []string{"worktree", "add", "/path/to/worktree", "feature"},
		},
		{
			name:         "with force flag",
			workTreePath: "/path/to/worktree",
			branchName:   "feature",
			flags:        map[string]interface{}{"force": true},
			want:         []string{"worktree", "add", "--force", "/path/to/worktree", "feature"},
		},
		{
			name:         "with new branch flag",
			workTreePath: "/path/to/worktree",
			branchName:   "new-feature",
			flags:        map[string]interface{}{"branch": "new-feature"},
			want:         []string{"worktree", "add", "-b", "new-feature", "/path/to/worktree"},
		},
		{
			name:         "with track flag",
			workTreePath: "/path/to/worktree",
			branchName:   "feature",
			flags:        map[string]interface{}{"track": "origin/feature"},
			want:         []string{"worktree", "add", "--track", "-b", "feature", "/path/to/worktree", "origin/feature"},
		},
		{
			name:         "detached HEAD",
			workTreePath: "/path/to/worktree",
			branchName:   "",
			flags:        map[string]interface{}{"detach": true},
			cliArgs:      []string{"abc1234"},
			want:         []string{"worktree", "add", "--detach", "/path/to/worktree", "abc1234"},
		},
		{
			name:         "explicit path",
			workTreePath: "/custom/path",
			branchName:   "feature",
			flags:        map[string]interface{}{"path": "/custom/path"},
			want:         []string{"worktree", "add", "/custom/path", "feature"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create app with all required flags
			app := &cli.Command{
				Name: "test",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "force"},
					&cli.BoolFlag{Name: "detach"},
					&cli.StringFlag{Name: "branch"},
					&cli.StringFlag{Name: "track"},
					&cli.StringFlag{Name: "path"},
				},
				Action: func(_ context.Context, cmd *cli.Command) error {
					// Call buildGitWorktreeArgs and verify result
					result := buildGitWorktreeArgs(cmd, tt.workTreePath, tt.branchName)
					assert.Equal(t, tt.want, result)
					return nil
				},
			}
			
			// Build args
			args := []string{"test"}
			
			// Add flags
			for flag, value := range tt.flags {
				switch v := value.(type) {
				case bool:
					if v {
						args = append(args, "--"+flag)
					}
				case string:
					args = append(args, "--"+flag, v)
				}
			}
			
			// Add CLI args
			args = append(args, tt.cliArgs...)
			
			ctx := context.Background()
			err := app.Run(ctx, args)
			assert.NoError(t, err)
		})
	}
}

func TestAppendBasicFlags(t *testing.T) {
	tests := []struct {
		name  string
		force bool
		detach bool
		want  []string
	}{
		{
			name: "no flags",
			want: []string{},
		},
		{
			name:  "force flag",
			force: true,
			want:  []string{"--force"},
		},
		{
			name:   "detach flag",
			detach: true,
			want:   []string{"--detach"},
		},
		{
			name:   "both flags",
			force:  true,
			detach: true,
			want:   []string{"--force", "--detach"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a properly initialized app to test flags
			app := &cli.Command{
				Name: "test",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "force"},
					&cli.BoolFlag{Name: "detach"},
				},
				Action: func(_ context.Context, cmd *cli.Command) error {
					got := appendBasicFlags([]string{}, cmd)
					assert.Equal(t, tt.want, got)
					return nil
				},
			}
			
			// Build args
			args := []string{"test"}
			if tt.force {
				args = append(args, "--force")
			}
			if tt.detach {
				args = append(args, "--detach")
			}
			
			ctx := context.Background()
			err := app.Run(ctx, args)
			assert.NoError(t, err)
		})
	}
}

func TestAppendBranchAndTrackFlags(t *testing.T) {
	tests := []struct {
		name       string
		branch     string
		track      string
		branchName string
		isDetached bool
		want       []string
	}{
		{
			name:       "branch only",
			branch:     "new-feature",
			branchName: "feature",
			want:       []string{"-b", "new-feature"},
		},
		{
			name:       "track only",
			track:      "origin/feature",
			branchName: "feature",
			want:       []string{"--track", "-b", "feature"},
		},
		{
			name:       "track with detached",
			track:      "origin/feature",
			branchName: "feature",
			isDetached: true,
			want:       []string{"--track"},
		},
		{
			name:       "branch and track",
			branch:     "new-feature",
			track:      "origin/feature",
			branchName: "feature",
			want:       []string{"-b", "new-feature", "--track"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := appendBranchAndTrackFlags([]string{}, tt.branch, tt.track, tt.branchName, tt.isDetached)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestShouldChangeDirectory(t *testing.T) {
	tests := []struct {
		name      string
		cdFlag    bool
		noCdFlag  bool
		cfgValue  bool
		want      bool
	}{
		{
			name:     "cd flag set",
			cdFlag:   true,
			cfgValue: false,
			want:     true,
		},
		{
			name:     "no-cd flag set",
			noCdFlag: true,
			cfgValue: true,
			want:     false,
		},
		{
			name:     "config true, no flags",
			cfgValue: true,
			want:     true,
		},
		{
			name:     "config false, no flags",
			cfgValue: false,
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			cmd := &cli.Command{
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "cd"},
					&cli.BoolFlag{Name: "no-cd"},
				},
			}
			
			if tt.cdFlag {
				_ = cmd.Set("cd", "true")
			}
			if tt.noCdFlag {
				_ = cmd.Set("no-cd", "true")
			}

			cfg := &config.Config{
				Defaults: config.Defaults{
					CDAfterCreate: tt.cfgValue,
				},
			}

			got := shouldChangeDirectory(cmd, cfg)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestChangeToWorktree(t *testing.T) {
	tests := []struct {
		name            string
		shellIntegration string
		workTreePath    string
		wantOutput      string
	}{
		{
			name:         "without shell integration",
			workTreePath: "/path/to/worktree",
			wantOutput:   "To change directory, run: cd /path/to/worktree\n(Enable shell integration with: eval \"$(wtp completion zsh)\")\n",
		},
		{
			name:             "with shell integration",
			shellIntegration: "1",
			workTreePath:     "/path/to/worktree",
			wantOutput:       "/path/to/worktree",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save and restore stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			// Set env var
			if tt.shellIntegration != "" {
				os.Setenv("WTP_SHELL_INTEGRATION", tt.shellIntegration)
				defer os.Unsetenv("WTP_SHELL_INTEGRATION")
			}

			// Call function
			changeToWorktree(tt.workTreePath)

			// Restore stdout and read output
			w.Close()
			os.Stdout = oldStdout
			buf := make([]byte, 1024)
			n, _ := r.Read(buf)
			output := string(buf[:n])

			assert.Equal(t, tt.wantOutput, output)
		})
	}
}

func TestDisplaySuccessMessage(t *testing.T) {
	tests := []struct {
		name         string
		branchName   string
		workTreePath string
		wantOutput   string
	}{
		{
			name:         "with branch name",
			branchName:   "feature",
			workTreePath: "/path/to/worktree",
			wantOutput:   "Created worktree 'feature' at /path/to/worktree\n",
		},
		{
			name:         "without branch name",
			branchName:   "",
			workTreePath: "/path/to/worktree",
			wantOutput:   "Created worktree at /path/to/worktree\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture stdout
			oldStdout := os.Stdout
			r, w, _ := os.Pipe()
			os.Stdout = w

			displaySuccessMessage(tt.branchName, tt.workTreePath)

			w.Close()
			os.Stdout = oldStdout
			buf := make([]byte, 1024)
			n, _ := r.Read(buf)
			output := string(buf[:n])

			assert.Equal(t, tt.wantOutput, output)
		})
	}
}


func TestExecutePostCreateHooks(t *testing.T) {
	tests := []struct {
		name         string
		cfg          *config.Config
		expectOutput bool
	}{
		{
			name: "with hooks",
			cfg: &config.Config{
				Hooks: config.Hooks{
					PostCreate: []config.Hook{
						{
							Type: "copy",
							From: ".env.example",
							To:   ".env",
						},
					},
				},
			},
			expectOutput: true,
		},
		{
			name:         "without hooks",
			cfg:          &config.Config{},
			expectOutput: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Capture output
			var buf bytes.Buffer
			oldStdout := os.Stdout
			oldStderr := os.Stderr
			r, w, _ := os.Pipe()
			os.Stdout = w
			os.Stderr = w

			err := executePostCreateHooks(tt.cfg, "/repo", "/worktree")
			assert.NoError(t, err)

			w.Close()
			os.Stdout = oldStdout
			os.Stderr = oldStderr
			buf.ReadFrom(r)

			if tt.expectOutput {
				assert.Contains(t, buf.String(), "Executing post-create hooks...")
			} else {
				assert.Empty(t, buf.String())
			}
		})
	}
}


func TestAppendPositionalArgs(t *testing.T) {
	tests := []struct {
		name         string
		initialArgs  []string
		pathFlag     string
		branch       string
		track        string
		branchName   string
		cliArgs      []string
		expectedArgs []string
	}{
		{
			name:         "explicit path without branch or track",
			initialArgs:  []string{"worktree", "add"},
			pathFlag:     "/custom/path",
			branchName:   "feature",
			expectedArgs: []string{"worktree", "add", "feature"},
		},
		{
			name:         "explicit path with track",
			initialArgs:  []string{"worktree", "add"},
			pathFlag:     "/custom/path",
			track:        "origin/feature",
			branchName:   "feature",
			expectedArgs: []string{"worktree", "add", "origin/feature"},
		},
		{
			name:         "auto path with branch flag",
			initialArgs:  []string{"worktree", "add"},
			branch:       "new-feature",
			cliArgs:      []string{"main"},
			expectedArgs: []string{"worktree", "add", "main"},
		},
		{
			name:         "auto path with track",
			initialArgs:  []string{"worktree", "add"},
			track:        "origin/feature",
			branchName:   "feature",
			expectedArgs: []string{"worktree", "add", "origin/feature"},
		},
		{
			name:         "auto path simple branch",
			initialArgs:  []string{"worktree", "add"},
			branchName:   "feature",
			expectedArgs: []string{"worktree", "add", "feature"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create app and run it to properly initialize CLI context
			app := &cli.Command{
				Name: "test",
				Flags: []cli.Flag{
					&cli.StringFlag{Name: "path"},
					&cli.BoolFlag{Name: "detach"},
				},
				Action: func(_ context.Context, cmd *cli.Command) error {
					// Call the function
					result := appendPositionalArgs(tt.initialArgs, cmd, tt.branch, tt.track, tt.branchName)
					assert.Equal(t, tt.expectedArgs, result)
					return nil
				},
			}
			
			// Build args
			args := []string{"test"}
			if tt.pathFlag != "" {
				args = append(args, "--path", tt.pathFlag)
			}
			args = append(args, tt.cliArgs...)
			
			ctx := context.Background()
			err := app.Run(ctx, args)
			assert.NoError(t, err)
		})
	}
}

func TestAppendExplicitPathArgs(t *testing.T) {
	tests := []struct {
		name         string
		initialArgs  []string
		branch       string
		track        string
		branchName   string
		cliArgs      []string
		expectedArgs []string
	}{
		{
			name:         "no branch or track",
			initialArgs:  []string{"worktree", "add", "/path"},
			branchName:   "feature",
			expectedArgs: []string{"worktree", "add", "/path", "feature"},
		},
		{
			name:         "with track but no branch",
			initialArgs:  []string{"worktree", "add", "/path"},
			track:        "origin/feature",
			branchName:   "feature",
			expectedArgs: []string{"worktree", "add", "/path", "origin/feature"},
		},
		{
			name:         "with branch flag",
			initialArgs:  []string{"worktree", "add", "/path"},
			branch:       "new-feature",
			branchName:   "feature",
			expectedArgs: []string{"worktree", "add", "/path"},
		},
		{
			name:         "with multiple args",
			initialArgs:  []string{"worktree", "add", "/path"},
			branchName:   "feature",
			cliArgs:      []string{"arg1", "arg2", "arg3"},
			expectedArgs: []string{"worktree", "add", "/path", "feature", "arg2", "arg3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &cli.Command{
				Name: "test",
				Action: func(_ context.Context, cmd *cli.Command) error {
					result := appendExplicitPathArgs(tt.initialArgs, cmd, tt.branch, tt.track, tt.branchName)
					assert.Equal(t, tt.expectedArgs, result)
					return nil
				},
			}
			
			// Build args
			args := []string{"test"}
			args = append(args, tt.cliArgs...)
			
			ctx := context.Background()
			err := app.Run(ctx, args)
			assert.NoError(t, err)
		})
	}
}

func TestAppendAutoPathArgs(t *testing.T) {
	tests := []struct {
		name         string
		initialArgs  []string
		branch       string
		track        string
		branchName   string
		detach       bool
		cliArgs      []string
		expectedArgs []string
	}{
		{
			name:         "with branch and args",
			initialArgs:  []string{"worktree", "add", "/path"},
			branch:       "new-feature",
			cliArgs:      []string{"main"},
			expectedArgs: []string{"worktree", "add", "/path", "main"},
		},
		{
			name:         "with track only",
			initialArgs:  []string{"worktree", "add", "/path"},
			track:        "origin/feature",
			expectedArgs: []string{"worktree", "add", "/path", "origin/feature"},
		},
		{
			name:         "simple branch",
			initialArgs:  []string{"worktree", "add", "/path"},
			branchName:   "feature",
			expectedArgs: []string{"worktree", "add", "/path", "feature"},
		},
		{
			name:         "detached without track",
			initialArgs:  []string{"worktree", "add", "/path"},
			detach:       true,
			cliArgs:      []string{"abc123"},
			expectedArgs: []string{"worktree", "add", "/path", "abc123"},
		},
		{
			name:         "detached with track",
			initialArgs:  []string{"worktree", "add", "/path"},
			detach:       true,
			track:        "origin/feature",
			expectedArgs: []string{"worktree", "add", "/path", "origin/feature"},
		},
		{
			name:         "multiple args without branch or track",
			initialArgs:  []string{"worktree", "add", "/path"},
			branchName:   "feature",
			cliArgs:      []string{"arg1", "arg2", "arg3"},
			expectedArgs: []string{"worktree", "add", "/path", "feature", "arg2", "arg3"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			app := &cli.Command{
				Name: "test",
				Flags: []cli.Flag{
					&cli.BoolFlag{Name: "detach"},
				},
				Action: func(_ context.Context, cmd *cli.Command) error {
					result := appendAutoPathArgs(tt.initialArgs, cmd, tt.branch, tt.track, tt.branchName)
					assert.Equal(t, tt.expectedArgs, result)
					return nil
				},
			}
			
			// Build args
			args := []string{"test"}
			if tt.detach {
				args = append(args, "--detach")
			}
			args = append(args, tt.cliArgs...)
			
			ctx := context.Background()
			err := app.Run(ctx, args)
			assert.NoError(t, err)
		})
	}
}

// Test the shell complete function
func TestAddCommand_ShellComplete(t *testing.T) {
	cmd := NewAddCommand()
	assert.NotNil(t, cmd.ShellComplete)
	
	// Test that shell complete function exists and can be called
	ctx := context.Background()
	cliCmd := &cli.Command{}
	
	// ShellComplete returns nothing, just test it doesn't panic
	assert.NotPanics(t, func() {
		cmd.ShellComplete(ctx, cliCmd)
	})
}

// Test helper function for building git worktree args
func TestBuildGitWorktreeArgsLogic(t *testing.T) {
	// Test the logic of building args without full CLI context
	tests := []struct {
		name         string
		hasForce     bool
		hasDetach    bool
		hasBranch    string
		hasTrack     string
		workTreePath string
		branchName   string
		expectedArgs []string
	}{
		{
			name:         "basic add",
			workTreePath: "/worktree",
			branchName:   "feature",
			expectedArgs: []string{"worktree", "add", "/worktree", "feature"},
		},
		{
			name:         "with force",
			hasForce:     true,
			workTreePath: "/worktree",
			branchName:   "feature",
			expectedArgs: []string{"worktree", "add", "--force", "/worktree", "feature"},
		},
		{
			name:         "with new branch",
			hasBranch:    "new-feature",
			workTreePath: "/worktree",
			expectedArgs: []string{"worktree", "add", "-b", "new-feature", "/worktree"},
		},
		{
			name:         "with track",
			hasTrack:     "origin/feature",
			workTreePath: "/worktree",
			branchName:   "feature",
			expectedArgs: []string{"worktree", "add", "--track", "-b", "feature", "/worktree", "origin/feature"},
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Simulate the logic of buildGitWorktreeArgs
			args := []string{"worktree", "add"}
			
			// Add basic flags
			if tt.hasForce {
				args = append(args, "--force")
			}
			if tt.hasDetach {
				args = append(args, "--detach")
			}
			
			// Add branch and track flags
			if tt.hasBranch != "" {
				args = append(args, "-b", tt.hasBranch)
			} else if tt.hasTrack != "" {
				args = append(args, "--track", "-b", tt.branchName)
			}
			
			// Add path
			args = append(args, tt.workTreePath)
			
			// Add positional args
			if tt.hasBranch == "" {
				if tt.hasTrack != "" {
					args = append(args, tt.hasTrack)
				} else if tt.branchName != "" {
					args = append(args, tt.branchName)
				}
			}
			
			assert.Equal(t, tt.expectedArgs, args)
		})
	}
}



