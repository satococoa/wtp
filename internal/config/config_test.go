package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestMain(m *testing.M) {
	tmpDir, err := os.MkdirTemp("", "wtp-config-test-*")
	if err != nil {
		os.Exit(1)
	}
	_ = os.Setenv("XDG_CONFIG_HOME", tmpDir)
	code := m.Run()
	_ = os.Unsetenv("XDG_CONFIG_HOME")
	_ = os.RemoveAll(tmpDir)
	os.Exit(code)
}

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

	if len(config.Hooks.PostCreate) != 2 {
		t.Errorf("Expected 2 hooks, got %d", len(config.Hooks.PostCreate))
	}

	if config.Hooks.PostCreate[0].Type != HookTypeCopy {
		t.Errorf("Expected first hook type 'copy', got %s", config.Hooks.PostCreate[0].Type)
	}

	if config.Hooks.PostCreate[1].Type != HookTypeCommand {
		t.Errorf("Expected second hook type 'command', got %s", config.Hooks.PostCreate[1].Type)
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

func TestLoadGlobalConfig_FromXDGConfigHome(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "wtp")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configContent := `version: "1.0"
defaults:
  base_dir: ".."
hooks:
  post_create:
    - type: command
      command: "echo global"
`
	configPath := filepath.Join(configDir, "config.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", tempDir)

	cfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.Defaults.BaseDir != ".." {
		t.Errorf("Expected base_dir '..', got %s", cfg.Defaults.BaseDir)
	}

	if len(cfg.Hooks.PostCreate) != 1 {
		t.Errorf("Expected 1 hook, got %d", len(cfg.Hooks.PostCreate))
	}
}

func TestLoadGlobalConfig_FallbackToHomeConfig(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, ".config", "wtp")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configContent := `version: "1.0"
defaults:
  base_dir: "../my-worktrees"
`
	configPath := filepath.Join(configDir, "config.yml")
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", "")
	t.Setenv("HOME", tempDir)

	cfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("Expected no error, got %v", err)
	}

	if cfg.Defaults.BaseDir != "../my-worktrees" {
		t.Errorf("Expected base_dir '../my-worktrees', got %s", cfg.Defaults.BaseDir)
	}
}

func TestLoadGlobalConfig_ReturnsEmptyWhenMissing(t *testing.T) {
	tempDir := t.TempDir()
	t.Setenv("XDG_CONFIG_HOME", tempDir)
	t.Setenv("HOME", tempDir)

	cfg, err := LoadGlobalConfig()
	if err != nil {
		t.Fatalf("Expected no error when global config missing, got %v", err)
	}

	if cfg == nil {
		t.Fatal("Expected non-nil config")
	}

	if cfg.Version != "" {
		t.Errorf("Expected empty version, got %s", cfg.Version)
	}
}

func TestMergeConfigs(t *testing.T) {
	tests := []struct {
		name          string
		global        *Config
		project       *Config
		expectedDir   string
		expectedHooks int
		expectedVer   string
	}{
		{
			name: "project overrides global base_dir",
			global: &Config{
				Version:  "1.0",
				Defaults: Defaults{BaseDir: ".."},
			},
			project: &Config{
				Version:  "1.0",
				Defaults: Defaults{BaseDir: ".worktrees"},
			},
			expectedDir:   ".worktrees",
			expectedHooks: 0,
			expectedVer:   "1.0",
		},
		{
			name: "global base_dir used when project empty",
			global: &Config{
				Version:  "1.0",
				Defaults: Defaults{BaseDir: ".."},
			},
			project: &Config{
				Version: "1.0",
			},
			expectedDir:   "..",
			expectedHooks: 0,
			expectedVer:   "1.0",
		},
		{
			name: "hooks concatenate global then project",
			global: &Config{
				Version: "1.0",
				Hooks: Hooks{
					PostCreate: []Hook{
						{Type: HookTypeCopy, From: "a", To: "b"},
					},
				},
			},
			project: &Config{
				Version: "1.0",
				Hooks: Hooks{
					PostCreate: []Hook{
						{Type: HookTypeCommand, Command: "echo test"},
					},
				},
			},
			expectedDir:   "",
			expectedHooks: 2,
			expectedVer:   "1.0",
		},
		{
			name: "project version preferred",
			global: &Config{
				Version: "0.9",
			},
			project: &Config{
				Version: "1.0",
			},
			expectedDir:   "",
			expectedHooks: 0,
			expectedVer:   "1.0",
		},
		{
			name:   "nil global config",
			global: nil,
			project: &Config{
				Version:  "1.0",
				Defaults: Defaults{BaseDir: "../worktrees"},
			},
			expectedDir:   "../worktrees",
			expectedHooks: 0,
			expectedVer:   "1.0",
		},
		{
			name: "nil project config",
			global: &Config{
				Version:  "1.0",
				Defaults: Defaults{BaseDir: ".."},
				Hooks: Hooks{
					PostCreate: []Hook{
						{Type: HookTypeCommand, Command: "npm install"},
					},
				},
			},
			project:       nil,
			expectedDir:   "..",
			expectedHooks: 1,
			expectedVer:   "1.0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := MergeConfigs(tt.global, tt.project)

			if result.Defaults.BaseDir != tt.expectedDir {
				t.Errorf("Expected base_dir %q, got %q", tt.expectedDir, result.Defaults.BaseDir)
			}

			if len(result.Hooks.PostCreate) != tt.expectedHooks {
				t.Errorf("Expected %d hooks, got %d", tt.expectedHooks, len(result.Hooks.PostCreate))
			}

			if result.Version != tt.expectedVer {
				t.Errorf("Expected version %q, got %q", tt.expectedVer, result.Version)
			}
		})
	}
}

func TestMergeConfigs_HookOrder(t *testing.T) {
	global := &Config{
		Hooks: Hooks{
			PostCreate: []Hook{
				{Type: HookTypeCommand, Command: "echo global"},
			},
		},
	}
	project := &Config{
		Hooks: Hooks{
			PostCreate: []Hook{
				{Type: HookTypeCommand, Command: "echo project"},
			},
		},
	}

	result := MergeConfigs(global, project)

	if len(result.Hooks.PostCreate) != 2 {
		t.Fatalf("Expected 2 hooks, got %d", len(result.Hooks.PostCreate))
	}

	if result.Hooks.PostCreate[0].Command != "echo global" {
		t.Errorf("Expected first hook to be global, got %s", result.Hooks.PostCreate[0].Command)
	}

	if result.Hooks.PostCreate[1].Command != "echo project" {
		t.Errorf("Expected second hook to be project, got %s", result.Hooks.PostCreate[1].Command)
	}
}

func TestLoadConfig_StatError(t *testing.T) {
	tempDir := t.TempDir()
	configPath := filepath.Join(tempDir, ConfigFileName)

	// Create a config file
	if err := os.WriteFile(configPath, []byte("version: \"1.0\"\n"), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Make the directory unsearchable (permission denied on stat)
	if err := os.Chmod(tempDir, 0o000); err != nil {
		t.Fatalf("Failed to chmod: %v", err)
	}
	defer func() {
		_ = os.Chmod(tempDir, 0o755)
	}()

	_, err := LoadConfig(tempDir)
	if err == nil {
		t.Error("Expected error for stat failure, got nil")
	}
}

func TestLoadGlobalConfig_StatError(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "wtp")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	configPath := filepath.Join(configDir, GlobalConfigFileName)
	if err := os.WriteFile(configPath, []byte("version: \"1.0\"\n"), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	// Make the config directory unsearchable
	if err := os.Chmod(configDir, 0o000); err != nil {
		t.Fatalf("Failed to chmod: %v", err)
	}
	defer func() {
		_ = os.Chmod(configDir, 0o755)
	}()

	t.Setenv("XDG_CONFIG_HOME", tempDir)

	_, err := LoadGlobalConfig()
	if err == nil {
		t.Error("Expected error for stat failure, got nil")
	}
}

func TestLoadGlobalConfig_InvalidConfig(t *testing.T) {
	tempDir := t.TempDir()
	configDir := filepath.Join(tempDir, "wtp")
	if err := os.MkdirAll(configDir, 0o755); err != nil {
		t.Fatalf("Failed to create config dir: %v", err)
	}

	// Write a config with invalid hook (missing required fields)
	configContent := `version: "1.0"
hooks:
  post_create:
    - type: copy
      from: ".env.example"
      # Missing 'to' field - should fail validation
`
	configPath := filepath.Join(configDir, GlobalConfigFileName)
	if err := os.WriteFile(configPath, []byte(configContent), 0o644); err != nil {
		t.Fatalf("Failed to write config: %v", err)
	}

	t.Setenv("XDG_CONFIG_HOME", tempDir)

	_, err := LoadGlobalConfig()
	if err == nil {
		t.Error("Expected validation error for invalid global config, got nil")
	}
}
