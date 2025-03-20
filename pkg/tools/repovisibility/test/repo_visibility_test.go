package test

import (
	"context"
	"testing"

	"github.com/google/go-github/v45/github"

	"github.com/anupsv/git-monitoring/pkg/config"
	mockgithub "github.com/anupsv/git-monitoring/pkg/tools/common/test"
	"github.com/anupsv/git-monitoring/pkg/tools/repovisibility"
)

func TestNewRepoVisibilityChecker(t *testing.T) {
	mockClient := &mockgithub.MockGitHubClient{}
	cfg := &config.Config{
		Monitors: config.MonitorsConfig{
			RepoVisibility: config.RepoVisibilityConfig{
				Enabled:        true,
				CheckWindow:    48,
				RepoVisibility: "all",
				Organizations:  []string{"testorg"},
			},
		},
	}

	checker := repovisibility.NewRepoVisibilityChecker(mockClient, cfg)
	if checker == nil {
		t.Fatal("Expected a non-nil checker")
	}
}

func TestRunWithInvalidVisibility(t *testing.T) {
	// Setup mock client
	mockClient := &mockgithub.MockGitHubClient{}

	// Create config with invalid visibility
	cfg := &config.Config{
		Monitors: config.MonitorsConfig{
			RepoVisibility: config.RepoVisibilityConfig{
				Enabled:        true,
				CheckWindow:    24,
				RepoVisibility: "invalid-value", // Invalid value to trigger error
				Organizations:  []string{"testorg"},
			},
		},
	}

	// Create checker
	checker := repovisibility.NewRepoVisibilityChecker(mockClient, cfg)

	// Run the checker
	_, err := checker.Run(context.Background())

	// Expect an error for invalid visibility
	if err == nil {
		t.Error("Expected an error for invalid visibility but got nil")
	}
}

func TestRunWithNoEvents(t *testing.T) {
	// Setup mock client with empty results
	mockClient := &mockgithub.MockGitHubClient{
		MockOrgRepositories: []*github.Repository{},
		MockRepoEvents:      []*github.Event{},
	}

	// Create config
	cfg := &config.Config{
		Monitors: config.MonitorsConfig{
			RepoVisibility: config.RepoVisibilityConfig{
				Enabled:        true,
				CheckWindow:    24,
				RepoVisibility: "all",
				Organizations:  []string{"testorg"},
			},
		},
	}

	// Create checker
	checker := repovisibility.NewRepoVisibilityChecker(mockClient, cfg)

	// Run the checker
	results, err := checker.Run(context.Background())

	// Verify results
	if err != nil {
		t.Errorf("Did not expect an error but got: %v", err)
	}

	if len(results) != 0 {
		t.Errorf("Expected 0 results, got %d", len(results))
	}
}
