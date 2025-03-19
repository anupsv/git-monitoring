package config

import (
	"fmt"
	"os"

	"github.com/BurntSushi/toml"
)

// Config represents the application configuration
type Config struct {
	GitHub   GitHubConfig   `toml:"github"`
	Monitors MonitorsConfig `toml:"monitors"`
}

// GitHubConfig contains GitHub API configuration
type GitHubConfig struct {
	Token string `toml:"token"`
}

// MonitorsConfig contains configuration for all monitors
type MonitorsConfig struct {
	PRChecker PRCheckerConfig `toml:"pr_checker"`
}

// PRCheckerConfig contains configuration for the PR checker
type PRCheckerConfig struct {
	Enabled      bool     `toml:"enabled"`
	Repositories []string `toml:"repositories"`
	TimeWindow   int      `toml:"time_window_hours"` // Time window in hours
}

// LoadConfig loads the configuration from the specified file
func LoadConfig(filePath string) (*Config, error) {
	config := &Config{
		Monitors: MonitorsConfig{
			PRChecker: PRCheckerConfig{
				TimeWindow: 24, // Default to 24 hours
			},
		},
	}

	_, err := os.Stat(filePath)
	if err != nil {
		return nil, fmt.Errorf("config file not found: %v", err)
	}

	_, err = toml.DecodeFile(filePath, config)
	if err != nil {
		return nil, fmt.Errorf("error decoding config file: %v", err)
	}

	// Check if token is in environment variable
	if envToken := os.Getenv("GITHUB_TOKEN"); envToken != "" {
		config.GitHub.Token = envToken
	}

	return config, nil
}

// Validate ensures the configuration is valid
func (c *Config) Validate() error {
	if c.GitHub.Token == "" {
		return fmt.Errorf("GitHub token is required. Set it in the config file or GITHUB_TOKEN environment variable")
	}

	if c.Monitors.PRChecker.Enabled && len(c.Monitors.PRChecker.Repositories) == 0 {
		return fmt.Errorf("at least one repository must be specified for PR checker")
	}

	if c.Monitors.PRChecker.TimeWindow <= 0 {
		return fmt.Errorf("time window must be greater than 0")
	}

	return nil
}
