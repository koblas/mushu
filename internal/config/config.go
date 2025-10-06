package config

import (
	"context"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/providers/env"
	"github.com/knadh/koanf/providers/file"
	"github.com/knadh/koanf/v2"
)

// Config represents the main configuration structure
type Config struct {
	GitHub  GitHubConfig  `yaml:"github"`
	Teams   TeamsConfig   `yaml:"teams"`
	Policy  PolicyConfig  `yaml:"policy"`
	Logging LoggingConfig `yaml:"logging"`
}

// GitHubConfig contains GitHub API configuration
type GitHubConfig struct {
	Token   string `yaml:"token"`
	Owner   string `yaml:"owner"`
	Repo    string `yaml:"repo"`
	BaseURL string `yaml:"base_url"`
}

// TeamsConfig contains team management configuration
type TeamsConfig struct {
	UseGitHubAPI bool   `yaml:"use_github_api"`
	TeamsFile    string `yaml:"teams_file"`
	TeamsDir     string `yaml:"teams_dir"`
}

// PolicyConfig contains policy configuration
type PolicyConfig struct {
	PolicyDir   string   `yaml:"policy_dir"`
	PolicyFiles []string `yaml:"policy_files"`
	RulesFile   string   `yaml:"rules_file"`
}

// LoggingConfig contains logging configuration
type LoggingConfig struct {
	Level  string `yaml:"level"`
	Format string `yaml:"format"`
}

// Load loads configuration with proper priority: file -> environment -> command line
func Load(ctx context.Context, configPath string, flagSet *flag.FlagSet) (*Config, error) {
	// Use command line config file if provided, otherwise use default
	if flagSet != nil {
		if configFlag := flagSet.Lookup("config"); configFlag != nil && configFlag.Value.String() != "" {
			configPath = configFlag.Value.String()
		}
	}
	if configPath == "" {
		configPath = "config.yaml"
	}

	// Create koanf instance
	k := koanf.New(".")

	// 1. Load configuration file (lowest priority)
	if _, err := os.Stat(configPath); err == nil {
		if err := k.Load(file.Provider(configPath), yaml.Parser()); err != nil {
			return nil, fmt.Errorf("failed to load config file %s: %w", configPath, err)
		}
	}

	// 2. Load environment variables with MUSHU_ prefix (medium priority)
	if err := k.Load(env.Provider("MUSHU_", ".", func(s string) string {
		return strings.ReplaceAll(strings.ToLower(strings.TrimPrefix(s, "MUSHU_")), "_", ".")
	}), nil); err != nil {
		return nil, fmt.Errorf("failed to load environment variables: %w", err)
	}

	// 3. Load command line flags (highest priority)
	if flagSet != nil {
		// Extract flag values and create confmap
		flagMap := make(map[string]interface{})

		// Define flag mappings to config paths
		flagMappings := map[string]string{
			"github-token": "github.token",
			"github-owner": "github.owner",
			"github-repo":  "github.repo",
			"log-level":    "logging.level",
			"log-format":   "logging.format",
		}

		// Extract flag values
		flagSet.Visit(func(f *flag.Flag) {
			if configPath, exists := flagMappings[f.Name]; exists {
				flagMap[configPath] = f.Value.String()
			}
		})

		// Load flag values using confmap provider
		if len(flagMap) > 0 {
			if err := k.Load(confmap.Provider(flagMap, "."), nil); err != nil {
				return nil, fmt.Errorf("failed to load command line flags: %w", err)
			}
		}
	}

	// Unmarshal into Config struct
	cfg := Config{
		Logging: LoggingConfig{
			Level:  "info",
			Format: "text",
		},
	}
	if err := k.Unmarshal("", &cfg); err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	// Set defaults
	setDefaults(&cfg)

	return &cfg, nil
}

// setDefaults sets default values for configuration fields
func setDefaults(cfg *Config) {
	if cfg.GitHub.BaseURL == "" {
		cfg.GitHub.BaseURL = "https://api.github.com"
	}
	if cfg.Teams.TeamsFile == "" {
		cfg.Teams.TeamsFile = "teams.yaml"
	}
	if cfg.Teams.TeamsDir == "" {
		cfg.Teams.TeamsDir = "teams/"
	}
	if cfg.Policy.PolicyDir == "" {
		cfg.Policy.PolicyDir = "policies/"
	}
	if cfg.Policy.RulesFile == "" {
		cfg.Policy.RulesFile = "mushu.yaml"
	}
	if cfg.Logging.Level == "" {
		cfg.Logging.Level = "info"
	}
	if cfg.Logging.Format == "" {
		cfg.Logging.Format = "json"
	}
}

// GetRulesPath returns the path to the rules file for a given directory
func (c *Config) GetRulesPath(dir string) string {
	return filepath.Join(dir, c.Policy.RulesFile)
}
