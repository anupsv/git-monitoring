package test

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"github.com/anupsv/git-monitoring/pkg/tools/common"
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

func TestListRepositoryMethods(t *testing.T) {
	// Create a mock GitHub client for testing
	mockClient := github.NewClient(nil)
	limiter := rate.NewLimiter(rate.Limit(1.0), 1)
	client := &common.GitHubClient{
		Client:      mockClient,
		RateLimiter: limiter,
	}

	// Create a context with a timeout
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	// Test cases for repository visibility values
	testCases := []struct {
		name           string
		visibility     string
		expectError    bool
		errorSubstring string
	}{
		{
			name:        "Valid visibility all",
			visibility:  "all",
			expectError: false,
		},
		{
			name:        "Valid visibility public-only",
			visibility:  "public-only",
			expectError: false,
		},
		{
			name:        "Valid visibility private-only",
			visibility:  "private-only",
			expectError: false,
		},
		{
			name:           "Invalid visibility",
			visibility:     "invalid",
			expectError:    true,
			errorSubstring: "invalid repository visibility",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name+" ListUserRepositories", func(t *testing.T) {
			_, err := client.ListUserRepositories(ctx, tc.visibility)

			// In our test context, this will always error with a timeout or transport error
			// But we can still check for the validation error for invalid visibility
			if tc.expectError {
				if err == nil {
					t.Error("Expected an error but got nil")
				} else if !strings.Contains(err.Error(), tc.errorSubstring) {
					t.Errorf("Expected error to contain %q, got %q", tc.errorSubstring, err.Error())
				}
			}
		})

		t.Run(tc.name+" ListOrganizationRepositories", func(t *testing.T) {
			_, err := client.ListOrganizationRepositories(ctx, "testorg", tc.visibility)

			// In our test context, this will always error with a timeout or transport error
			// But we can still check for the validation error for invalid visibility
			if tc.expectError {
				if err == nil {
					t.Error("Expected an error but got nil")
				} else if !strings.Contains(err.Error(), tc.errorSubstring) {
					t.Errorf("Expected error to contain %q, got %q", tc.errorSubstring, err.Error())
				}
			}
		})
	}

	// Test empty organization name
	t.Run("Empty organization name", func(t *testing.T) {
		_, err := client.ListOrganizationRepositories(ctx, "", "all")
		if err == nil {
			t.Error("Expected an error for empty organization name but got nil")
		}
		if !strings.Contains(err.Error(), "organization name cannot be empty") {
			t.Errorf("Expected error to contain 'organization name cannot be empty', got %v", err)
		}
	})
}

func TestParseRepositoryEdgeCases(t *testing.T) {
	testCases := []struct {
		name       string
		input      string
		wantOwner  string
		wantRepo   string
		wantResult bool
	}{
		{
			name:       "Missing repo part",
			input:      "owner/",
			wantOwner:  "owner",
			wantRepo:   "",
			wantResult: true, // The function considers this valid as it has 2 parts
		},
		{
			name:       "Missing owner part",
			input:      "/repo",
			wantOwner:  "",
			wantRepo:   "repo",
			wantResult: true, // The function considers this valid as it has 2 parts
		},
		{
			name:       "Too many parts",
			input:      "owner/repo/extra",
			wantOwner:  "",
			wantRepo:   "",
			wantResult: false,
		},
		{
			name:       "With whitespace",
			input:      "owner / repo",
			wantOwner:  "owner ",
			wantRepo:   " repo",
			wantResult: true, // The function just splits on / without trimming
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			owner, repo, ok := common.ParseRepository(tc.input)
			if ok != tc.wantResult {
				t.Errorf("ParseRepository(%q) result = %v, want %v", tc.input, ok, tc.wantResult)
			}
			if owner != tc.wantOwner {
				t.Errorf("ParseRepository(%q) owner = %q, want %q", tc.input, owner, tc.wantOwner)
			}
			if repo != tc.wantRepo {
				t.Errorf("ParseRepository(%q) repo = %q, want %q", tc.input, repo, tc.wantRepo)
			}
		})
	}
}
