package config

import (
	"fmt"
	"log"
	"os"

	"github.com/BurntSushi/toml"
)

// Config represents the application configuration
type Config struct {
	GitHub      GitHubConfig   `toml:"github"`
	Monitors    MonitorsConfig `toml:"monitors"`
	RepoFilters Filters        `toml:"repo_filters"`
}

// GitHubConfig contains GitHub API configuration
type GitHubConfig struct {
	Token string `toml:"token"`
}

// MonitorsConfig contains configuration for all monitors
type MonitorsConfig struct {
	PRChecker      PRCheckerConfig      `toml:"pr_checker"`
	RepoVisibility RepoVisibilityConfig `toml:"repo_visibility"`
}

// PRCheckerConfig contains configuration for the PR checker
type PRCheckerConfig struct {
	Enabled              bool     `toml:"enabled"`
	RepoVisibility       string   `toml:"repo_visibility"`       // Options: "all", "public-only", "private-only", "specific"
	Organization         string   `toml:"organization"`          // GitHub organization name (optional)
	SpecificRepositories []string `toml:"specific_repositories"` // Only used when RepoVisibility is "specific"
	ExcludedRepositories []string `toml:"excluded_repositories"` // Used with "all", "public-only", "private-only" to exclude specific repos
	TimeWindow           int      `toml:"time_window_hours"`     // Time window in hours
	DebugLogging         bool     `toml:"debug_logging"`         // Enable verbose logging for debugging
}

// RepoVisibilityConfig contains configuration for the repository visibility checker
type RepoVisibilityConfig struct {
	Enabled bool `toml:"enabled"` // Whether the repository visibility checker is enabled

	// Repository visibility filter. Options: "all", "public-only", "private-only", "specific"
	RepoVisibility string `toml:"repo_visibility"`

	// Organizations to monitor for repository visibility changes
	Organizations []string `toml:"organizations"`

	// Time window (in hours) to look for visibility changes
	CheckWindow int `toml:"check_window_hours"`
}

// Filters contains repository filtering configuration
type Filters struct {
	Topic      string   `toml:"topic"`
	Exclusions []string `toml:"exclusions"`
}

// LoadConfig loads the configuration from the specified file
func LoadConfig(filePath string) (*Config, error) {
	config := &Config{
		Monitors: MonitorsConfig{
			PRChecker: PRCheckerConfig{
				TimeWindow:           24,         // Default to 24 hours
				RepoVisibility:       "specific", // Default to specific repos
				SpecificRepositories: []string{}, // Empty list as default
				ExcludedRepositories: []string{}, // Empty list as default
			},
			RepoVisibility: RepoVisibilityConfig{
				Enabled:        false, // Default to disabled
				CheckWindow:    24,    // Default to 24 hours
				Organizations:  []string{},
				RepoVisibility: "specific", // Default to specific repos
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

	if c.Monitors.PRChecker.Enabled {
		// Validate repo visibility setting
		validVisibilities := map[string]bool{
			"all":          true,
			"public-only":  true,
			"private-only": true,
			"specific":     true,
		}

		if !validVisibilities[c.Monitors.PRChecker.RepoVisibility] {
			return fmt.Errorf("invalid repository visibility: %s. Must be one of: all, public-only, private-only, specific",
				c.Monitors.PRChecker.RepoVisibility)
		}

		// Only check repositories list if visibility is set to "specific"
		if c.Monitors.PRChecker.RepoVisibility == "specific" && len(c.Monitors.PRChecker.SpecificRepositories) == 0 {
			return fmt.Errorf("at least one repository must be specified for PR checker when repo_visibility is 'specific'")
		}

		// If organization is specified with "specific" visibility, warn but continue
		if c.Monitors.PRChecker.RepoVisibility == "specific" && c.Monitors.PRChecker.Organization != "" {
			log.Printf("WARNING: Organization '%s' is specified but repo_visibility is 'specific'. The organization setting will be ignored.",
				c.Monitors.PRChecker.Organization)
		}
	}

	if c.Monitors.PRChecker.TimeWindow <= 0 {
		return fmt.Errorf("time window must be greater than 0")
	}

	if c.Monitors.RepoVisibility.Enabled {
		// Validate repo visibility setting
		validVisibilities := map[string]bool{
			"all":          true,
			"public-only":  true,
			"private-only": true,
			"specific":     true,
		}

		if !validVisibilities[c.Monitors.RepoVisibility.RepoVisibility] {
			return fmt.Errorf("invalid repository visibility for repo_visibility monitor: %s. Must be one of: all, public-only, private-only, specific",
				c.Monitors.RepoVisibility.RepoVisibility)
		}

		// If using "specific" visibility, require at least one organization
		if c.Monitors.RepoVisibility.RepoVisibility == "specific" && len(c.Monitors.RepoVisibility.Organizations) == 0 {
			return fmt.Errorf("at least one organization must be specified for repo_visibility monitor when repo_visibility is 'specific'")
		}

		// All visibility options require at least one organization
		if len(c.Monitors.RepoVisibility.Organizations) == 0 {
			return fmt.Errorf("at least one organization must be specified for repo_visibility monitor")
		}

		if c.Monitors.RepoVisibility.CheckWindow <= 0 {
			return fmt.Errorf("check window for repo visibility must be greater than 0")
		}
	}

	return nil
}
