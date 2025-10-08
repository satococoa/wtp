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

	// Internal field: tracks if namespace_by_repo was auto-detected (not in YAML)
	namespaceAutoDetected bool `yaml:"-"`
}

// Defaults represents default configuration values
type Defaults struct {
	BaseDir         string `yaml:"base_dir,omitempty"`
	NamespaceByRepo *bool  `yaml:"namespace_by_repo,omitempty"` // nil = auto-detect, true = namespaced, false = legacy
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
	ConfigFileName        = ".wtp.yml"
	CurrentVersion        = "1.0"
	DefaultBaseDir        = "../worktrees"
	HookTypeCopy          = "copy"
	HookTypeCommand       = "command"
	configFilePermissions = 0o600
)

// LoadConfig loads configuration from .wtp.yml in the repository root
func LoadConfig(repoRoot string) (*Config, error) {
	configPath := filepath.Join(repoRoot, ConfigFileName)

	// If config file doesn't exist, create default config with auto-detection
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		config := &Config{
			Version: CurrentVersion,
			Defaults: Defaults{
				BaseDir: DefaultBaseDir,
			},
			Hooks: Hooks{},
		}

		// Auto-detect: if legacy worktrees exist, use legacy layout
		if hasLegacyWorktrees(repoRoot, config.Defaults.BaseDir) {
			legacyMode := false
			config.Defaults.NamespaceByRepo = &legacyMode
			config.namespaceAutoDetected = true
		}
		// Otherwise use namespaced layout (NamespaceByRepo stays nil, defaults to true)

		return config, nil
	}

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

	// Auto-detect for existing configs without explicit namespace_by_repo setting
	if config.Defaults.NamespaceByRepo == nil {
		if hasLegacyWorktrees(repoRoot, config.Defaults.BaseDir) {
			legacyMode := false
			config.Defaults.NamespaceByRepo = &legacyMode
			config.namespaceAutoDetected = true
		}
	}

	return &config, nil
}

// hasLegacyWorktrees checks if there are worktrees in the legacy (non-namespaced) layout
func hasLegacyWorktrees(repoRoot, baseDir string) bool {
	if baseDir == "" {
		baseDir = DefaultBaseDir
	}

	if !filepath.IsAbs(baseDir) {
		baseDir = filepath.Join(repoRoot, baseDir)
	}

	// Check if base directory exists
	info, err := os.Stat(baseDir)
	if err != nil || !info.IsDir() {
		return false
	}

	// Check for subdirectories that look like worktrees (not repo names)
	// A legacy worktree would be directly under baseDir
	// A namespaced worktree would be under baseDir/<repo-name>/
	entries, err := os.ReadDir(baseDir)
	if err != nil {
		return false
	}

	repoName := filepath.Base(repoRoot)
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		// If we find a directory that's not the repo name, it's likely a legacy worktree
		if entry.Name() != repoName {
			// Check if it looks like a worktree (has .git file)
			worktreePath := filepath.Join(baseDir, entry.Name())
			if _, err := os.Stat(filepath.Join(worktreePath, ".git")); err == nil {
				return true
			}
		}
	}

	return false
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

	// Check if we should use namespacing
	if c.ShouldNamespaceByRepo() {
		repoName := filepath.Base(repoRoot)
		return filepath.Join(baseDir, repoName, worktreeName)
	}

	return filepath.Join(baseDir, worktreeName)
}

// ShouldNamespaceByRepo returns whether to use repository namespacing
// Returns true if:
// - namespace_by_repo is explicitly set to true
// - namespace_by_repo is nil (not set) and no config file exists (new installations)
func (c *Config) ShouldNamespaceByRepo() bool {
	if c.Defaults.NamespaceByRepo != nil {
		return *c.Defaults.NamespaceByRepo
	}
	// Default to true for new installations (will be auto-detected in LoadConfig)
	return true
}

// UsesLegacyLayout returns whether the config is using the legacy non-namespaced layout
func (c *Config) UsesLegacyLayout() bool {
	return c.Defaults.NamespaceByRepo != nil && !*c.Defaults.NamespaceByRepo
}

// ShouldShowMigrationWarning returns whether to show the migration warning
// Shows warning if using legacy layout but NamespaceByRepo was auto-detected (not explicitly set)
func (c *Config) ShouldShowMigrationWarning() bool {
	return c.namespaceAutoDetected
}

// GetMigrationWarning returns the migration warning message
func GetMigrationWarning() string {
	return `
⚠️  Legacy worktree layout detected. Consider migrating to namespaced layout:
   wtp migrate-worktrees --dry-run  # Preview changes
   wtp migrate-worktrees            # Migrate worktrees

   Or to keep current layout, add to .wtp.yml:
   defaults:
     namespace_by_repo: false
`
}
