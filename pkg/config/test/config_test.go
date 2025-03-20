package test

import (
	"os"
	"strings"
	"testing"

	"github.com/asv/git-monitoring/pkg/config"
)

func TestValidate(t *testing.T) {
	tests := []struct {
		name          string
		config        *config.Config
		expectError   bool
		errorContains string
	}{
		{
			name: "Valid configuration",
			config: &config.Config{
				GitHub: config.GitHubConfig{
					Token: "valid-token",
				},
				Monitors: config.MonitorsConfig{
					PRChecker: config.PRCheckerConfig{
						Enabled:              true,
						RepoVisibility:       "specific",
						Organization:         "",
						SpecificRepositories: []string{"owner/repo"},
						TimeWindow:           24,
					},
					RepoVisibility: config.RepoVisibilityConfig{
						Enabled: false,
					},
				},
			},
			expectError: false,
		},
		{
			name: "Missing GitHub token",
			config: &config.Config{
				GitHub: config.GitHubConfig{
					Token: "",
				},
				Monitors: config.MonitorsConfig{
					PRChecker: config.PRCheckerConfig{
						Enabled:              true,
						RepoVisibility:       "specific",
						Organization:         "",
						SpecificRepositories: []string{"owner/repo"},
						TimeWindow:           24,
					},
				},
			},
			expectError:   true,
			errorContains: "GitHub token is required",
		},
		{
			name: "PR Checker enabled but no repositories",
			config: &config.Config{
				GitHub: config.GitHubConfig{
					Token: "valid-token",
				},
				Monitors: config.MonitorsConfig{
					PRChecker: config.PRCheckerConfig{
						Enabled:              true,
						RepoVisibility:       "specific",
						Organization:         "",
						SpecificRepositories: []string{},
						TimeWindow:           24,
					},
				},
			},
			expectError:   true,
			errorContains: "at least one repository",
		},
		{
			name: "Invalid time window",
			config: &config.Config{
				GitHub: config.GitHubConfig{
					Token: "valid-token",
				},
				Monitors: config.MonitorsConfig{
					PRChecker: config.PRCheckerConfig{
						Enabled:              true,
						RepoVisibility:       "specific",
						Organization:         "",
						SpecificRepositories: []string{"owner/repo"},
						TimeWindow:           0,
					},
				},
			},
			expectError:   true,
			errorContains: "time window must be greater than 0",
		},
		{
			name: "Invalid repository visibility",
			config: &config.Config{
				GitHub: config.GitHubConfig{
					Token: "valid-token",
				},
				Monitors: config.MonitorsConfig{
					PRChecker: config.PRCheckerConfig{
						Enabled:              true,
						RepoVisibility:       "invalid",
						Organization:         "",
						SpecificRepositories: []string{"owner/repo"},
						TimeWindow:           24,
					},
				},
			},
			expectError:   true,
			errorContains: "invalid repository visibility",
		},
		{
			name: "Valid all repositories configuration with organization",
			config: &config.Config{
				GitHub: config.GitHubConfig{
					Token: "valid-token",
				},
				Monitors: config.MonitorsConfig{
					PRChecker: config.PRCheckerConfig{
						Enabled:              true,
						RepoVisibility:       "all",
						Organization:         "testorg",
						SpecificRepositories: []string{}, // Can be empty with non-specific visibility
						TimeWindow:           24,
					},
				},
			},
			expectError: false,
		},
		{
			name: "Organization specified with specific visibility",
			config: &config.Config{
				GitHub: config.GitHubConfig{
					Token: "valid-token",
				},
				Monitors: config.MonitorsConfig{
					PRChecker: config.PRCheckerConfig{
						Enabled:              true,
						RepoVisibility:       "specific",
						Organization:         "testorg", // Will generate a warning but not an error
						SpecificRepositories: []string{"owner/repo"},
						TimeWindow:           24,
					},
				},
			},
			expectError: false, // Not an error, just a warning
		},
		{
			name: "Repo Visibility enabled with invalid check window",
			config: &config.Config{
				GitHub: config.GitHubConfig{
					Token: "valid-token",
				},
				Monitors: config.MonitorsConfig{
					PRChecker: config.PRCheckerConfig{
						Enabled:    false,
						TimeWindow: 24, // Add default time window to pass validation
					},
					RepoVisibility: config.RepoVisibilityConfig{
						Enabled:        true,
						Organizations:  []string{"test-org"},
						CheckWindow:    0,     // Invalid check window
						RepoVisibility: "all", // Valid repo visibility
					},
				},
			},
			expectError:   true,
			errorContains: "check window for repo visibility must be greater than 0",
		},
		{
			name: "Valid Repo Visibility configuration",
			config: &config.Config{
				GitHub: config.GitHubConfig{
					Token: "valid-token",
				},
				Monitors: config.MonitorsConfig{
					PRChecker: config.PRCheckerConfig{
						Enabled:    false,
						TimeWindow: 24, // Add default time window to pass validation
					},
					RepoVisibility: config.RepoVisibilityConfig{
						Enabled:        true,
						Organizations:  []string{"test-org"},
						CheckWindow:    24, // Valid check window
						RepoVisibility: "specific",
					},
				},
			},
			expectError: false,
		},
		{
			name: "Repo Visibility with specific but no organizations",
			config: &config.Config{
				GitHub: config.GitHubConfig{
					Token: "valid-token",
				},
				Monitors: config.MonitorsConfig{
					PRChecker: config.PRCheckerConfig{
						Enabled:    false,
						TimeWindow: 24,
					},
					RepoVisibility: config.RepoVisibilityConfig{
						Enabled:        true,
						RepoVisibility: "specific",
						Organizations:  []string{}, // Empty organizations list with specific visibility
						CheckWindow:    24,
					},
				},
			},
			expectError:   true,
			errorContains: "at least one organization must be specified",
		},
		{
			name: "Repo Visibility with invalid visibility value",
			config: &config.Config{
				GitHub: config.GitHubConfig{
					Token: "valid-token",
				},
				Monitors: config.MonitorsConfig{
					PRChecker: config.PRCheckerConfig{
						Enabled:    false,
						TimeWindow: 24,
					},
					RepoVisibility: config.RepoVisibilityConfig{
						Enabled:        true,
						RepoVisibility: "invalid-value", // Invalid visibility
						CheckWindow:    24,
					},
				},
			},
			expectError:   true,
			errorContains: "invalid repository visibility",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			err := tc.config.Validate()

			if tc.expectError && err == nil {
				t.Error("Expected an error but got nil")
			}

			if !tc.expectError && err != nil {
				t.Errorf("Did not expect an error but got: %v", err)
			}

			if tc.expectError && tc.errorContains != "" && err != nil {
				if !strings.Contains(err.Error(), tc.errorContains) {
					t.Errorf("Expected error to contain %q, got %q", tc.errorContains, err.Error())
				}
			}
		})
	}
}

func TestLoadConfig(t *testing.T) {
	// Test loading the config file
	// Create a temporary config file for testing
	validConfig := `
# GitHub API configuration
[github]
token = "test-token"

# Monitor configurations
[monitors]
  # PR Checker configuration
  [monitors.pr_checker]
  enabled = true
  repo_visibility = "specific"
  organization = "test-org"
  specific_repositories = [
    "owner/repo"
  ]
  time_window_hours = 24
  
  # Repository visibility checker
  [monitors.repo_visibility]
  enabled = true
  repo_visibility = "all"
  organizations = ["test-org"]
  check_window_hours = 48
`

	tempFile, err := os.CreateTemp("", "config-*.toml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write([]byte(validConfig)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config file
	cfg, err := config.LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify the loaded configuration
	if cfg.GitHub.Token != "test-token" {
		t.Errorf("Expected GitHub token to be %q, got %q", "test-token", cfg.GitHub.Token)
	}

	if !cfg.Monitors.PRChecker.Enabled {
		t.Error("Expected PR Checker to be enabled")
	}

	if cfg.Monitors.PRChecker.RepoVisibility != "specific" {
		t.Errorf("Expected repository visibility to be %q, got %q", "specific", cfg.Monitors.PRChecker.RepoVisibility)
	}

	if cfg.Monitors.PRChecker.Organization != "test-org" {
		t.Errorf("Expected organization to be %q, got %q", "test-org", cfg.Monitors.PRChecker.Organization)
	}

	if len(cfg.Monitors.PRChecker.SpecificRepositories) != 1 || cfg.Monitors.PRChecker.SpecificRepositories[0] != "owner/repo" {
		t.Errorf("Expected specific_repositories to be [\"owner/repo\"], got %v", cfg.Monitors.PRChecker.SpecificRepositories)
	}

	if cfg.Monitors.PRChecker.TimeWindow != 24 {
		t.Errorf("Expected time window to be 24, got %d", cfg.Monitors.PRChecker.TimeWindow)
	}

	// Verify repository visibility configuration
	if !cfg.Monitors.RepoVisibility.Enabled {
		t.Error("Expected Repository Visibility monitor to be enabled")
	}

	if cfg.Monitors.RepoVisibility.RepoVisibility != "all" {
		t.Errorf("Expected RepoVisibility to be %q, got %q", "all", cfg.Monitors.RepoVisibility.RepoVisibility)
	}

	if len(cfg.Monitors.RepoVisibility.Organizations) != 1 || cfg.Monitors.RepoVisibility.Organizations[0] != "test-org" {
		t.Errorf("Expected organizations to be [\"test-org\"], got %v", cfg.Monitors.RepoVisibility.Organizations)
	}

	if cfg.Monitors.RepoVisibility.CheckWindow != 48 {
		t.Errorf("Expected check window to be 48, got %d", cfg.Monitors.RepoVisibility.CheckWindow)
	}
}

func TestLoadConfigWithEnvVariable(t *testing.T) {
	// Set the environment variable
	os.Setenv("GITHUB_TOKEN", "env-token")
	defer os.Unsetenv("GITHUB_TOKEN")

	// Create a temporary config file for testing
	configWithEmptyToken := `
# GitHub API configuration
[github]
token = ""

# Monitor configurations
[monitors]
  # PR Checker configuration
  [monitors.pr_checker]
  enabled = true
  repo_visibility = "specific"
  organization = "test-org"
  repositories = [
    "owner/repo"
  ]
  time_window_hours = 24
`

	tempFile, err := os.CreateTemp("", "config-*.toml")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tempFile.Name())

	if _, err := tempFile.Write([]byte(configWithEmptyToken)); err != nil {
		t.Fatalf("Failed to write to temp file: %v", err)
	}

	if err := tempFile.Close(); err != nil {
		t.Fatalf("Failed to close temp file: %v", err)
	}

	// Test loading the config file
	cfg, err := config.LoadConfig(tempFile.Name())
	if err != nil {
		t.Fatalf("Failed to load config: %v", err)
	}

	// Verify that the token was set from the environment variable
	if cfg.GitHub.Token != "env-token" {
		t.Errorf("Expected GitHub token to be %q (from env), got %q", "env-token", cfg.GitHub.Token)
	}
}

func TestLoadConfigFileNotFound(t *testing.T) {
	_, err := config.LoadConfig("non-existent-file.toml")
	if err == nil {
		t.Error("Expected an error for non-existent file, but got nil")
	}
}
