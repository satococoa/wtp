// Package config defines the configuration schema and helpers for wtp.
package config

import (
	"fmt"
	"os"
	"path/filepath"

	"go.yaml.in/yaml/v3"
)

// Config represents the wtp configuration
type Config struct {
	Version  string   `yaml:"version"`
	Defaults Defaults `yaml:"defaults,omitempty"`
	Hooks    Hooks    `yaml:"hooks,omitempty"`
}

// Defaults represents default configuration values
type Defaults struct {
	BaseDir string `yaml:"base_dir,omitempty"`
}

// Hooks represents the post-create hooks configuration
type Hooks struct {
	PostCreate []Hook `yaml:"post_create,omitempty"`
}

// Hook represents a single hook configuration
type Hook struct {
	Type    string            `yaml:"type"` // "copy" or "command"
	From    string            `yaml:"from,omitempty"`
	To      string            `yaml:"to,omitempty"`
	Command string            `yaml:"command,omitempty"`
	Env     map[string]string `yaml:"env,omitempty"`
	WorkDir string            `yaml:"work_dir,omitempty"`
}

const (
	// ConfigFileName is the default filename for the wtp configuration.
	ConfigFileName = ".wtp.yml"
	// CurrentVersion represents the current configuration version written to disk.
	CurrentVersion = "1.0"
	// DefaultBaseDir is the default directory for new worktrees relative to a repository.
	DefaultBaseDir = "../worktrees"
	// HookTypeCopy identifies a hook that copies files.
	HookTypeCopy = "copy"
	// HookTypeCommand identifies a hook that executes a command.
	HookTypeCommand       = "command"
	configFilePermissions = 0o600
)

// LoadConfig loads configuration from .wtp.yml in the repository root
func LoadConfig(repoRoot string) (*Config, error) {
	cleanedRoot := filepath.Clean(repoRoot)
	if !filepath.IsAbs(cleanedRoot) {
		absRoot, err := filepath.Abs(cleanedRoot)
		if err != nil {
			return nil, fmt.Errorf("failed to resolve repository root: %w", err)
		}
		cleanedRoot = absRoot
	}

	configPath := filepath.Join(cleanedRoot, ConfigFileName)

	// If config file doesn't exist, return default config
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return &Config{
			Version: CurrentVersion,
			Defaults: Defaults{
				BaseDir: "../worktrees",
			},
			Hooks: Hooks{},
		}, nil
	}

	// #nosec G304 -- configPath is derived from the validated repository root and fixed file name
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	if err := yaml.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse config file: %w", err)
	}

	// Validate configuration
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &config, nil
}

// SaveConfig saves configuration to .git-worktree-plus.yml in the repository root
func SaveConfig(repoRoot string, config *Config) error {
	if err := config.Validate(); err != nil {
		return fmt.Errorf("invalid configuration: %w", err)
	}

	configPath := filepath.Join(repoRoot, ConfigFileName)

	data, err := yaml.Marshal(config)
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	if err := os.WriteFile(configPath, data, configFilePermissions); err != nil {
		return fmt.Errorf("failed to write config file: %w", err)
	}

	return nil
}

// Validate validates the configuration
func (c *Config) Validate() error {
	if c.Version == "" {
		c.Version = CurrentVersion
	}

	// Set default base_dir if not specified
	if c.Defaults.BaseDir == "" {
		c.Defaults.BaseDir = DefaultBaseDir
	}

	// Validate hooks
	for i, hook := range c.Hooks.PostCreate {
		if err := hook.Validate(); err != nil {
			return fmt.Errorf("invalid hook %d: %w", i+1, err)
		}
	}

	return nil
}

// Validate validates a single hook configuration
func (h *Hook) Validate() error {
	switch h.Type {
	case HookTypeCopy:
		if h.From == "" || h.To == "" {
			return fmt.Errorf("copy hook requires both 'from' and 'to' fields")
		}
		if h.Command != "" {
			return fmt.Errorf("copy hook should not have 'command' field")
		}
	case HookTypeCommand:
		if h.Command == "" {
			return fmt.Errorf("command hook requires 'command' field")
		}
		if h.From != "" || h.To != "" {
			return fmt.Errorf("command hook should not have 'from' or 'to' fields")
		}
	default:
		return fmt.Errorf("invalid hook type '%s', must be 'copy' or 'command'", h.Type)
	}

	return nil
}

// HasHooks returns true if the configuration has any post-create hooks
func (c *Config) HasHooks() bool {
	return len(c.Hooks.PostCreate) > 0
}

// ResolveWorktreePath resolves the full path for a worktree given a name
func (c *Config) ResolveWorktreePath(repoRoot, worktreeName string) string {
	baseDir := c.Defaults.BaseDir
	if !filepath.IsAbs(baseDir) {
		baseDir = filepath.Join(repoRoot, baseDir)
	}
	return filepath.Join(baseDir, worktreeName)
}
