package common

import (
	"context"
	"log"
	"strings"
	"time"

	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

// GitHubClient wraps the GitHub client with rate limiting
type GitHubClient struct {
	Client      *github.Client
	RateLimiter *rate.Limiter
}

// NewGitHubClient creates a new authenticated GitHub client with rate limiting
func NewGitHubClient(ctx context.Context, token string) *GitHubClient {
	ts := oauth2.StaticTokenSource(
		&oauth2.Token{AccessToken: token},
	)
	tc := oauth2.NewClient(ctx, ts)
	client := github.NewClient(tc)

	// GitHub's API allows 5000 requests per hour for authenticated requests
	// We'll set a conservative limit of 4500 per hour (1.25 per second)
	limiter := rate.NewLimiter(rate.Limit(1.25), 1)

	return &GitHubClient{
		Client:      client,
		RateLimiter: limiter,
	}
}

// ExecuteWithRateLimit executes a GitHub API call with rate limiting
func (c *GitHubClient) ExecuteWithRateLimit(ctx context.Context, f func() error) error {
	if err := c.RateLimiter.Wait(ctx); err != nil {
		return err
	}

	err := f()

	// Check if we're approaching rate limits and log
	rateLimits, _, rateLimitErr := c.Client.RateLimits(ctx)
	if rateLimitErr == nil && rateLimits.Core != nil && rateLimits.Core.Remaining < 100 {
		log.Printf("WARNING: GitHub API rate limit is getting low. %d/%d requests remaining, resets at %s",
			rateLimits.Core.Remaining, rateLimits.Core.Limit, rateLimits.Core.Reset.Time.Format(time.RFC3339))
	}

	return err
}

// ParseRepository parses an "owner/repo" string into separate owner and repo components
func ParseRepository(repository string) (string, string, bool) {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}
