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

	if config.Defaults.BaseDir != "../worktrees" {
		t.Errorf("Expected default base_dir '../worktrees', got %s", config.Defaults.BaseDir)
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
					BaseDir: "../worktrees",
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
					BaseDir: "../worktrees",
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
					BaseDir: "../worktrees",
				},
			},
			repoRoot:     "/home/user/project",
			worktreeName: "feature/auth",
			expected:     "/home/user/worktrees/feature/auth",
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
					BaseDir: "../worktrees",
				},
			},
			repoRoot:     "/home/user/project",
			worktreeName: "main",
			expected:     "/home/user/worktrees/main",
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
