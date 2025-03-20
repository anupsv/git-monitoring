package test

import (
	"context"
	"testing"
	"time"

	"github.com/google/go-github/v45/github"

	"github.com/anupsv/git-monitoring/pkg/config"
	"github.com/anupsv/git-monitoring/pkg/tools/common/test"
	"github.com/anupsv/git-monitoring/pkg/tools/repovisibility"
)

func TestRepoVisibilityChecker_CheckOrganization(t *testing.T) {
	// Create a mock client
	mockClient := &test.MockGitHubClient{}

	// Setup test config
	testConfig := &config.Config{
		Monitors: config.MonitorsConfig{
			RepoVisibility: config.RepoVisibilityConfig{
				Organizations:  []string{"testorg"},
				CheckWindow:    24,
				Enabled:        true,
				RepoVisibility: "specific",
			},
		},
	}

	// Create the checker
	checker := repovisibility.NewRepoVisibilityChecker(mockClient, testConfig)

	// Test 1: Repository created recently (should be included)
	t.Run("RecentlyCreatedRepository", func(t *testing.T) {
		// Setup mock repository handler function
		// nolint:revive
		mockClient.ListOrgRepositoriesFunc = func(ctx context.Context, org, visibility string) ([]*github.Repository, error) {
			return []*github.Repository{
				{
					Name: github.String("new-repo"),
					Owner: &github.User{
						Login: github.String("testorg"),
					},
				},
			}, nil
		}

		// Run the check
		publicRepos, err := checker.CheckOrganization(context.Background(), "testorg")

		// Verify results
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !contains(publicRepos, "testorg/new-repo") {
			t.Errorf("Expected repository %s to be included, got %v", "testorg/new-repo", publicRepos)
		}
	})

	// Test 2: Old repository that was recently made public
	t.Run("OldRepositoryMadePublic", func(t *testing.T) {
		// Setup mock repository handler function with "old" timestamp
		// nolint:revive
		mockClient.ListOrgRepositoriesFunc = func(ctx context.Context, org, visibility string) ([]*github.Repository, error) {
			// Create a timestamp that will be considered "old"
			oldTime := &github.Timestamp{
				Time: time.Now().Add(-48 * time.Hour), // 48 hours ago, outside the check window
			}

			return []*github.Repository{
				{
					Name: github.String("old-repo"),
					Owner: &github.User{
						Login: github.String("testorg"),
					},
					CreatedAt: oldTime,
				},
			}, nil
		}

		// Setup mock event handler function
		// nolint:revive
		mockClient.ListRepositoryEventsFunc = func(ctx context.Context, owner, repo string) ([]*github.Event, error) {
			return []*github.Event{
				{
					Type: github.String("PublicEvent"),
					Repo: &github.Repository{
						Name: github.String("testorg/old-repo"),
					},
				},
			}, nil
		}

		// Run the check
		publicRepos, err := checker.CheckOrganization(context.Background(), "testorg")

		// Verify results
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !contains(publicRepos, "testorg/old-repo") {
			t.Errorf("Expected repository %s to be included, got %v", "testorg/old-repo", publicRepos)
		}
	})

	// Test 3: Old repository that was not recently made public
	t.Run("OldRepositoryNotMadePublic", func(t *testing.T) {
		// Setup mock repository handler function with "old" timestamp
		// nolint:revive
		mockClient.ListOrgRepositoriesFunc = func(ctx context.Context, org, visibility string) ([]*github.Repository, error) {
			// Create a timestamp that will be considered "old"
			oldTime := &github.Timestamp{
				Time: time.Now().Add(-48 * time.Hour), // 48 hours ago, outside the check window
			}

			return []*github.Repository{
				{
					Name: github.String("old-repo-private"),
					Owner: &github.User{
						Login: github.String("testorg"),
					},
					CreatedAt: oldTime,
				},
			}, nil
		}

		// Setup mock event handler function that returns no public events
		// nolint:revive
		mockClient.ListRepositoryEventsFunc = func(ctx context.Context, owner, repo string) ([]*github.Event, error) {
			return []*github.Event{}, nil
		}

		// Run the check
		publicRepos, err := checker.CheckOrganization(context.Background(), "testorg")

		// Verify results
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if contains(publicRepos, "testorg/old-repo-private") {
			t.Errorf("Repository %s should not be included, got %v", "testorg/old-repo-private", publicRepos)
		}
	})

	// Test 4: Organization visibility check with 'all' option
	t.Run("OrganizationAllVisibility", func(t *testing.T) {
		// Setup test config with "all" visibility
		orgAllConfig := &config.Config{
			Monitors: config.MonitorsConfig{
				RepoVisibility: config.RepoVisibilityConfig{
					Organizations:  []string{"testorg"},
					CheckWindow:    24,
					Enabled:        true,
					RepoVisibility: "all",
				},
			},
		}

		orgChecker := repovisibility.NewRepoVisibilityChecker(mockClient, orgAllConfig)

		// Setup mock repository handler function
		// nolint:revive
		mockClient.ListOrgRepositoriesFunc = func(ctx context.Context, org, visibility string) ([]*github.Repository, error) {
			// Create a timestamp that will be considered "old"
			oldTime := &github.Timestamp{
				Time: time.Now().Add(-48 * time.Hour), // 48 hours ago, outside the check window
			}

			return []*github.Repository{
				{
					Name: github.String("public-repo"),
					Owner: &github.User{
						Login: github.String("testorg"),
					},
					CreatedAt: oldTime,
					Private:   github.Bool(false),
				},
				{
					Name: github.String("private-repo"),
					Owner: &github.User{
						Login: github.String("testorg"),
					},
					CreatedAt: oldTime,
					Private:   github.Bool(true),
				},
			}, nil
		}

		// Make the public repo show up as recently made public
		// nolint:revive
		mockClient.ListRepositoryEventsFunc = func(ctx context.Context, owner, repo string) ([]*github.Event, error) {
			if repo == "public-repo" {
				return []*github.Event{
					{
						Type: github.String("PublicEvent"),
						Repo: &github.Repository{
							Name: github.String("testorg/public-repo"),
						},
					},
				}, nil
			}
			return []*github.Event{}, nil
		}

		// Run the check
		publicRepos, err := orgChecker.Run(context.Background())

		// Verify results
		if err != nil {
			t.Errorf("Expected no error, got %v", err)
		}
		if !contains(publicRepos, "testorg/public-repo") {
			t.Errorf("Expected repository %s to be included, got %v", "testorg/public-repo", publicRepos)
		}
	})
}

func TestRepoVisibilityChecker_Run(t *testing.T) {
	// Create a mock client
	mockClient := &test.MockGitHubClient{}

	// Setup test config with multiple orgs
	testConfig := &config.Config{
		Monitors: config.MonitorsConfig{
			RepoVisibility: config.RepoVisibilityConfig{
				Organizations:  []string{"org1", "org2"},
				CheckWindow:    24,
				Enabled:        true,
				RepoVisibility: "all",
			},
		},
	}

	// Create the checker
	checker := repovisibility.NewRepoVisibilityChecker(mockClient, testConfig)

	// Setup mock functions for organizations
	// nolint:revive
	mockClient.ListOrgRepositoriesFunc = func(ctx context.Context, org, visibility string) ([]*github.Repository, error) {
		if org == "org1" {
			return []*github.Repository{
				{
					Name: github.String("repo1"),
					Owner: &github.User{
						Login: github.String("org1"),
					},
				},
			}, nil
		}
		return []*github.Repository{
			{
				Name: github.String("repo2"),
				Owner: &github.User{
					Login: github.String("org2"),
				},
			},
		}, nil
	}

	// Run the check
	publicRepos, err := checker.Run(context.Background())

	// Verify results
	if err != nil {
		t.Errorf("Expected no error, got %v", err)
	}
	if !contains(publicRepos, "org1/repo1") {
		t.Errorf("Expected repository %s to be included, got %v", "org1/repo1", publicRepos)
	}
	if !contains(publicRepos, "org2/repo2") {
		t.Errorf("Expected repository %s to be included, got %v", "org2/repo2", publicRepos)
	}
}

// Helper function to check if a slice contains a string
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}
