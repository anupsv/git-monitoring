package test

import (
	"context"
	"errors"
	"testing"

	"github.com/asv/git-monitoring/pkg/tools/common"
	"github.com/google/go-github/v45/github"
	"golang.org/x/time/rate"
)

func TestParseRepository(t *testing.T) {
	tests := []struct {
		name          string
		repository    string
		expectedOwner string
		expectedRepo  string
		expectedOk    bool
	}{
		{
			name:          "Valid repository format",
			repository:    "owner/repo",
			expectedOwner: "owner",
			expectedRepo:  "repo",
			expectedOk:    true,
		},
		{
			name:          "Invalid format - missing slash",
			repository:    "ownerrepo",
			expectedOwner: "",
			expectedRepo:  "",
			expectedOk:    false,
		},
		{
			name:          "Invalid format - too many parts",
			repository:    "owner/repo/extra",
			expectedOwner: "",
			expectedRepo:  "",
			expectedOk:    false,
		},
		{
			name:          "Invalid format - empty string",
			repository:    "",
			expectedOwner: "",
			expectedRepo:  "",
			expectedOk:    false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			owner, repo, ok := common.ParseRepository(tc.repository)
			if ok != tc.expectedOk {
				t.Errorf("Expected ok to be %v, got %v", tc.expectedOk, ok)
			}
			if owner != tc.expectedOwner {
				t.Errorf("Expected owner to be %q, got %q", tc.expectedOwner, owner)
			}
			if repo != tc.expectedRepo {
				t.Errorf("Expected repo to be %q, got %q", tc.expectedRepo, repo)
			}
		})
	}
}

func TestExecuteWithRateLimit(t *testing.T) {
	tests := []struct {
		name        string
		funcError   error
		expectError bool
	}{
		{
			name:        "Function returns no error",
			funcError:   nil,
			expectError: false,
		},
		{
			name:        "Function returns error",
			funcError:   errors.New("test error"),
			expectError: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// Create a GitHub client without using NewGitHubClient to avoid rate limit check
			limiter := rate.NewLimiter(rate.Limit(1.0), 1)
			client := &common.GitHubClient{
				Client:      github.NewClient(nil),
				RateLimiter: limiter,
			}

			// Test the ExecuteWithRateLimit function
			err := client.ExecuteWithRateLimit(context.Background(), func() error {
				return tc.funcError
			})

			// Check if the error is as expected
			if tc.expectError && err == nil {
				t.Errorf("Expected an error but got nil")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Did not expect an error but got: %v", err)
			}
			if tc.expectError && err != tc.funcError {
				t.Errorf("Expected error %v, got %v", tc.funcError, err)
			}
		})
	}
}

func TestNewGitHubClient(t *testing.T) {
	// Test that the client is created with the token
	client := common.NewGitHubClient(context.Background(), "test-token")

	if client == nil {
		t.Fatal("Expected non-nil client, got nil")
	}

	if client.Client == nil {
		t.Error("Expected client.Client to be non-nil, got nil")
	}

	if client.RateLimiter == nil {
		t.Error("Expected client.RateLimiter to be non-nil, got nil")
	}
}
