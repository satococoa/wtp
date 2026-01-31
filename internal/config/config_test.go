package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadConfig_NonExistentFile(t *testing.T) {
	tempDir := t.TempDir()

	config, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config.Version != CurrentVersion {
		t.Errorf("Expected version %s, got %s", CurrentVersion, config.Version)
	}

	if config.Defaults.BaseDir != DefaultBaseDir {
		t.Errorf("Expected default base_dir '%s', got %s", DefaultBaseDir, config.Defaults.BaseDir)
	}
}

func TestLoadConfig_ValidFile(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ConfigFileName)

	configContent := `version: "1.0"
defaults:
  base_dir: "../my-worktrees"
hooks:
  post_create:
    - type: copy
      from: ".env.example"
      to: ".env"
    - type: command
      command: "echo test"
    - type: symlink
      from: ".bin"
      to: ".bin"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if config.Version != "1.0" {
		t.Errorf("Expected version '1.0', got %s", config.Version)
	}

	if config.Defaults.BaseDir != "../my-worktrees" {
		t.Errorf("Expected base_dir '../my-worktrees', got %s", config.Defaults.BaseDir)
	}

	if len(config.Hooks.PostCreate) != 3 {
		t.Errorf("Expected 3 hooks, got %d", len(config.Hooks.PostCreate))
	}

	if config.Hooks.PostCreate[0].Type != HookTypeCopy {
		t.Errorf("Expected first hook type 'copy', got %s", config.Hooks.PostCreate[0].Type)
	}

	if config.Hooks.PostCreate[1].Type != HookTypeCommand {
		t.Errorf("Expected second hook type 'command', got %s", config.Hooks.PostCreate[1].Type)
	}

	if config.Hooks.PostCreate[2].Type != HookTypeSymlink {
		t.Errorf("Expected third hook type 'symlink', got %s", config.Hooks.PostCreate[2].Type)
	}
}

func TestLoadConfig_CopyHookDefaultsToFrom(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ConfigFileName)

	configContent := `version: "1.0"
hooks:
  post_create:
    - type: copy
      from: ".env"
`

	err := os.WriteFile(configPath, []byte(configContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	config, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if len(config.Hooks.PostCreate) != 1 {
		t.Fatalf("Expected 1 hook, got %d", len(config.Hooks.PostCreate))
	}

	if got := config.Hooks.PostCreate[0].To; got != ".env" {
		t.Errorf("Expected hook.To to default to %q, got %q", ".env", got)
	}
}

func TestLoadConfig_InvalidYAML(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ConfigFileName)

	invalidContent := `version: "1.0"
hooks:
  post_create:
    - type: copy
      from: ".env.example"
      # Invalid YAML syntax
      to: ".env"
    invalid_structure
`

	err := os.WriteFile(configPath, []byte(invalidContent), 0644)
	if err != nil {
		t.Fatalf("Failed to write test config: %v", err)
	}

	_, err = LoadConfig(tempDir)
	if err == nil {
		t.Error("Expected error for invalid YAML, got nil")
	}
}

func TestFileExists(t *testing.T) {
	tempDir := t.TempDir()

	if FileExists(tempDir) {
		t.Fatal("Expected config file to be missing")
	}

	configPath := filepath.Join(tempDir, ConfigFileName)
	if err := os.WriteFile(configPath, []byte("version: 1.0\n"), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	if !FileExists(tempDir) {
		t.Fatal("Expected config file to exist")
	}
}

func TestSaveConfig(t *testing.T) {
	tempDir := t.TempDir()

	config := &Config{
		Version: "1.0",
		Defaults: Defaults{
			BaseDir: "../test-worktrees",
		},
		Hooks: Hooks{
			PostCreate: []Hook{
				{
					Type: HookTypeCopy,
					From: ".env.example",
					To:   ".env",
				},
			},
		},
	}

	err := SaveConfig(tempDir, config)
	if err != nil {
		t.Fatalf("Failed to save config: %v", err)
	}

	// Verify file was created
	configPath := filepath.Join(tempDir, ConfigFileName)
	if _, statErr := os.Stat(configPath); os.IsNotExist(statErr) {
		t.Error("Config file was not created")
	}

	// Load it back and verify
	loadedConfig, err := LoadConfig(tempDir)
	if err != nil {
		t.Fatalf("Failed to load saved config: %v", err)
	}

	if loadedConfig.Version != config.Version {
		t.Errorf("Expected version %s, got %s", config.Version, loadedConfig.Version)
	}

	if loadedConfig.Defaults.BaseDir != config.Defaults.BaseDir {
		t.Errorf("Expected base_dir %s, got %s", config.Defaults.BaseDir, loadedConfig.Defaults.BaseDir)
	}
}

func TestConfigValidate(t *testing.T) {
	tests := []struct {
		name        string
		config      *Config
		expectError bool
	}{
		{
			name: "valid config",
			config: &Config{
				Version: "1.0",
				Defaults: Defaults{
					BaseDir: DefaultBaseDir,
				},
				Hooks: Hooks{
					PostCreate: []Hook{
						{
							Type: HookTypeCopy,
							From: ".env.example",
							To:   ".env",
						},
					},
				},
			},
			expectError: false,
		},
		{
			name: "empty version gets default",
			config: &Config{
				Defaults: Defaults{
					BaseDir: DefaultBaseDir,
				},
			},
			expectError: false,
		},
		{
			name: "empty base_dir gets default",
			config: &Config{
				Version: "1.0",
			},
			expectError: false,
		},
		{
			name: "invalid copy hook - missing from",
			config: &Config{
				Version: "1.0",
				Hooks: Hooks{
					PostCreate: []Hook{
						{
							Type: HookTypeCopy,
							To:   ".env",
						},
					},
				},
			},
			expectError: true,
		},
		{
			name: "invalid command hook - missing command",
			config: &Config{
				Version: "1.0",
				Hooks: Hooks{
					PostCreate: []Hook{
						{
							Type: HookTypeCommand,
							// Missing Command field - should cause validation error
						},
					},
				},
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.config.ApplyDefaults()
			err := tt.config.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}

			// Check defaults are set
			if !tt.expectError {
				if tt.config.Version == "" {
					t.Error("Version should be set to default")
				}
				if tt.config.Defaults.BaseDir == "" {
					t.Error("BaseDir should be set to default")
				}
			}
		})
	}
}

func TestHookValidate(t *testing.T) {
	tests := []struct {
		name        string
		hook        Hook
		expectError bool
	}{
		{
			name: "valid copy hook",
			hook: Hook{
				Type: HookTypeCopy,
				From: ".env.example",
				To:   ".env",
			},
			expectError: false,
		},
		{
			name: "valid command hook",
			hook: Hook{
				Type:    HookTypeCommand,
				Command: "echo test",
			},
			expectError: false,
		},
		{
			name: "valid symlink hook",
			hook: Hook{
				Type: HookTypeSymlink,
				From: ".bin",
				To:   ".bin",
			},
			expectError: false,
		},
		{
			name: "copy hook missing from",
			hook: Hook{
				Type: HookTypeCopy,
				To:   ".env",
			},
			expectError: true,
		},
		{
			name: "copy hook missing to",
			hook: Hook{
				Type: HookTypeCopy,
				From: ".env.example",
			},
			expectError: false,
		},
		{
			name: "copy hook missing to with absolute from",
			hook: Hook{
				Type: HookTypeCopy,
				From: filepath.Join(string(os.PathSeparator), "tmp", "source.txt"),
			},
			expectError: true,
		},
		{
			name: "copy hook with command field",
			hook: Hook{
				Type:    HookTypeCopy,
				From:    ".env.example",
				To:      ".env",
				Command: "echo", // Should not have command
			},
			expectError: true,
		},
		{
			name: "command hook missing command",
			hook: Hook{
				Type: HookTypeCommand,
			},
			expectError: true,
		},
		{
			name: "symlink hook missing from",
			hook: Hook{
				Type: HookTypeSymlink,
				To:   ".bin",
			},
			expectError: true,
		},
		{
			name: "symlink hook missing to",
			hook: Hook{
				Type: HookTypeSymlink,
				From: ".bin",
			},
			expectError: true,
		},
		{
			name: "symlink hook with command field",
			hook: Hook{
				Type:    HookTypeSymlink,
				From:    ".bin",
				To:      ".bin",
				Command: "echo", // Should not have command
			},
			expectError: true,
		},
		{
			name: "command hook with from/to fields",
			hook: Hook{
				Type:    HookTypeCommand,
				Command: "echo",
				From:    ".env.example", // Should not have from/to
				To:      ".env",
			},
			expectError: true,
		},
		{
			name: "invalid hook type",
			hook: Hook{
				Type: "invalid",
			},
			expectError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.hook.Validate()
			if tt.expectError && err == nil {
				t.Error("Expected error but got nil")
			}
			if !tt.expectError && err != nil {
				t.Errorf("Expected no error but got: %v", err)
			}
		})
	}
}

func TestHookValidate_DoesNotMutateTo(t *testing.T) {
	hook := Hook{
		Type: HookTypeCopy,
		From: ".env",
	}

	if err := hook.Validate(); err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}

	if hook.To != "" {
		t.Errorf("Expected hook.To to remain empty, got %q", hook.To)
	}
}

func TestHookApplyDefaults_CopyToDefaultsToFrom(t *testing.T) {
	hook := Hook{
		Type: HookTypeCopy,
		From: ".env",
	}

	hook.ApplyDefaults()

	if hook.To != hook.From {
		t.Errorf("Expected hook.To to default to %q, got %q", hook.From, hook.To)
	}

	if err := hook.Validate(); err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}
}

func TestConfigApplyDefaults_CopyToDefaultsToFrom(t *testing.T) {
	config := &Config{
		Version: "1.0",
		Hooks: Hooks{
			PostCreate: []Hook{
				{
					Type: HookTypeCopy,
					From: ".env",
				},
			},
		},
	}

	config.ApplyDefaults()

	if err := config.Validate(); err != nil {
		t.Fatalf("Expected no error but got: %v", err)
	}

	if got := config.Hooks.PostCreate[0].To; got != ".env" {
		t.Errorf("Expected hook.To to default to %q, got %q", ".env", got)
	}
}

func TestConfigValidate_CopyAbsoluteFromRequiresTo(t *testing.T) {
	config := &Config{
		Version: "1.0",
		Hooks: Hooks{
			PostCreate: []Hook{
				{
					Type: HookTypeCopy,
					From: filepath.Join(string(os.PathSeparator), "tmp", "source.txt"),
				},
			},
		},
	}

	config.ApplyDefaults()

	if err := config.Validate(); err == nil {
		t.Fatalf("Expected error but got nil")
	}
}

func TestResolveWorktreePath(t *testing.T) {
	tests := []struct {
		name         string
		config       *Config
		repoRoot     string
		worktreeName string
		expected     string
	}{
		{
			name: "relative base_dir",
			config: &Config{
				Defaults: Defaults{
					BaseDir: DefaultBaseDir,
				},
			},
			repoRoot:     "/home/user/project",
			worktreeName: "feature/auth",
			expected:     "/home/user/project/.git/wtp/worktrees/feature/auth",
		},
		{
			name: "absolute base_dir",
			config: &Config{
				Defaults: Defaults{
					BaseDir: "/tmp/worktrees",
				},
			},
			repoRoot:     "/home/user/project",
			worktreeName: "feature/auth",
			expected:     "/tmp/worktrees/feature/auth",
		},
		{
			name: "simple worktree name",
			config: &Config{
				Defaults: Defaults{
					BaseDir: DefaultBaseDir,
				},
			},
			repoRoot:     "/home/user/project",
			worktreeName: "main",
			expected:     "/home/user/project/.git/wtp/worktrees/main",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.ResolveWorktreePath(tt.repoRoot, tt.worktreeName)
			if result != tt.expected {
				t.Errorf("Expected %s, got %s", tt.expected, result)
			}
		})
	}
}

func TestHasHooks(t *testing.T) {
	tests := []struct {
		name     string
		config   *Config
		expected bool
	}{
		{
			name: "config with hooks",
			config: &Config{
				Hooks: Hooks{
					PostCreate: []Hook{
						{Type: HookTypeCopy, From: "a", To: "b"},
					},
				},
			},
			expected: true,
		},
		{
			name: "config without hooks",
			config: &Config{
				Hooks: Hooks{},
			},
			expected: false,
		},
		{
			name: "config with empty hooks slice",
			config: &Config{
				Hooks: Hooks{
					PostCreate: []Hook{},
				},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := tt.config.HasHooks()
			if result != tt.expected {
				t.Errorf("Expected %v, got %v", tt.expected, result)
			}
		})
	}
}
