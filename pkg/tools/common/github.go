package common

import (
	"context"
	"fmt"
	"log"
	"strings"
	"time"

	"github.com/google/go-github/v45/github"
	"golang.org/x/oauth2"
	"golang.org/x/time/rate"
)

// GitHubClientInterface defines the interface for GitHub client operations
// This allows us to mock it for testing
type GitHubClientInterface interface {
	ExecuteWithRateLimit(ctx context.Context, f func() error) error
	GetPullRequests(ctx context.Context, owner, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	ListPullRequestReviews(ctx context.Context, owner, repo string, number int, opts *github.ListOptions) ([]*github.PullRequestReview, *github.Response, error)
	ListUserRepositories(ctx context.Context, visibility string) ([]*github.Repository, error)
	ListOrganizationRepositories(ctx context.Context, org string, visibility string) ([]*github.Repository, error)
	ListRepositoryEvents(ctx context.Context, owner, repo string) ([]*github.Event, error)
	ListUserEventsForOrganization(ctx context.Context, org, user string) ([]*github.Event, error)
	ListRepositoryPublicEvents(ctx context.Context) ([]*github.Event, error)
}

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

// GetPullRequests gets pull requests for a repository
func (c *GitHubClient) GetPullRequests(ctx context.Context, owner, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	var prs []*github.PullRequest
	var resp *github.Response
	err := c.ExecuteWithRateLimit(ctx, func() error {
		var apiErr error
		prs, resp, apiErr = c.Client.PullRequests.List(ctx, owner, repo, opts)
		return apiErr
	})

	return prs, resp, err
}

// ListPullRequestReviews lists reviews for a pull request
func (c *GitHubClient) ListPullRequestReviews(ctx context.Context, owner, repo string, number int, opts *github.ListOptions) ([]*github.PullRequestReview, *github.Response, error) {
	var reviews []*github.PullRequestReview
	var resp *github.Response
	err := c.ExecuteWithRateLimit(ctx, func() error {
		var apiErr error
		reviews, resp, apiErr = c.Client.PullRequests.ListReviews(ctx, owner, repo, number, opts)
		return apiErr
	})

	return reviews, resp, err
}

// ListUserRepositories lists repositories for the authenticated user based on visibility
func (c *GitHubClient) ListUserRepositories(ctx context.Context, visibility string) ([]*github.Repository, error) {
	opts := &github.RepositoryListOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	// Set visibility based on configuration
	switch visibility {
	case "public-only":
		opts.Visibility = "public"
	case "private-only":
		opts.Visibility = "private"
	case "all":
		opts.Visibility = "all"
	default:
		return nil, fmt.Errorf("invalid repository visibility: %s", visibility)
	}

	var allRepos []*github.Repository
	page := 1

	for {
		opts.Page = page
		var repos []*github.Repository
		var resp *github.Response

		err := c.ExecuteWithRateLimit(ctx, func() error {
			var apiErr error
			repos, resp, apiErr = c.Client.Repositories.List(ctx, "", opts)
			return apiErr
		})

		if err != nil {
			return nil, err
		}

		allRepos = append(allRepos, repos...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	return allRepos, nil
}

// ListOrganizationRepositories lists repositories for the specified organization based on visibility
func (c *GitHubClient) ListOrganizationRepositories(ctx context.Context, org string, visibility string) ([]*github.Repository, error) {
	if org == "" {
		return nil, fmt.Errorf("organization name cannot be empty")
	}

	opts := &github.RepositoryListByOrgOptions{
		ListOptions: github.ListOptions{PerPage: 100},
	}

	// Set type based on visibility
	switch visibility {
	case "public-only":
		opts.Type = "public"
	case "private-only":
		opts.Type = "private"
	case "all":
		opts.Type = "all"
	default:
		return nil, fmt.Errorf("invalid repository visibility: %s", visibility)
	}

	var allRepos []*github.Repository
	page := 1

	for {
		opts.Page = page
		var repos []*github.Repository
		var resp *github.Response

		err := c.ExecuteWithRateLimit(ctx, func() error {
			var apiErr error
			repos, resp, apiErr = c.Client.Repositories.ListByOrg(ctx, org, opts)
			return apiErr
		})

		if err != nil {
			return nil, fmt.Errorf("error listing repositories for organization %s: %v", org, err)
		}

		allRepos = append(allRepos, repos...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	return allRepos, nil
}

// ListRepositoryEvents lists events for a specific repository
func (c *GitHubClient) ListRepositoryEvents(ctx context.Context, owner, repo string) ([]*github.Event, error) {
	opts := &github.ListOptions{
		PerPage: 100,
	}

	var allEvents []*github.Event
	page := 1

	for {
		opts.Page = page
		var events []*github.Event
		var resp *github.Response

		err := c.ExecuteWithRateLimit(ctx, func() error {
			var apiErr error
			events, resp, apiErr = c.Client.Activity.ListRepositoryEvents(ctx, owner, repo, opts)
			return apiErr
		})

		if err != nil {
			return nil, fmt.Errorf("error listing repository events for %s/%s: %v", owner, repo, err)
		}

		allEvents = append(allEvents, events...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	return allEvents, nil
}

// ListUserEventsForOrganization lists events performed by a user in an organization
func (c *GitHubClient) ListUserEventsForOrganization(ctx context.Context, org, user string) ([]*github.Event, error) {
	opts := &github.ListOptions{
		PerPage: 100,
	}

	var allEvents []*github.Event
	page := 1

	for {
		opts.Page = page
		var events []*github.Event
		var resp *github.Response

		err := c.ExecuteWithRateLimit(ctx, func() error {
			var apiErr error
			events, resp, apiErr = c.Client.Activity.ListUserEventsForOrganization(ctx, org, user, opts)
			return apiErr
		})

		if err != nil {
			return nil, fmt.Errorf("error listing user events for organization %s and user %s: %v", org, user, err)
		}

		allEvents = append(allEvents, events...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	return allEvents, nil
}

// ListRepositoryPublicEvents lists public events across GitHub
func (c *GitHubClient) ListRepositoryPublicEvents(ctx context.Context) ([]*github.Event, error) {
	opts := &github.ListOptions{
		PerPage: 100,
	}

	var allEvents []*github.Event
	page := 1

	for {
		opts.Page = page
		var events []*github.Event
		var resp *github.Response

		err := c.ExecuteWithRateLimit(ctx, func() error {
			var apiErr error
			events, resp, apiErr = c.Client.Activity.ListEvents(ctx, opts)
			return apiErr
		})

		if err != nil {
			return nil, fmt.Errorf("error listing public events: %v", err)
		}

		allEvents = append(allEvents, events...)

		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	return allEvents, nil
}

// ParseRepository parses an "owner/repo" string into separate owner and repo components
func ParseRepository(repository string) (string, string, bool) {
	parts := strings.Split(repository, "/")
	if len(parts) != 2 {
		return "", "", false
	}
	return parts[0], parts[1], true
}
