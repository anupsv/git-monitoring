package test

import (
	"context"

	"github.com/google/go-github/v45/github"
)

// MockGitHubClient is a mock implementation of GitHubClientInterface for testing
type MockGitHubClient struct {
	// Mock return values
	MockPullRequests        []*github.PullRequest
	MockPullRequestResp     *github.Response
	MockPullRequestErr      error
	MockReviews             []*github.PullRequestReview
	MockReviewResp          *github.Response
	MockReviewErr           error
	MockExecuteRateLimitErr error
	MockRepositories        []*github.Repository
	MockRepositoriesErr     error
	MockOrgRepositories     []*github.Repository
	MockOrgRepositoriesErr  error
	MockRepoEvents          []*github.Event
	MockRepoEventsErr       error
	MockUserOrgEvents       []*github.Event
	MockUserOrgEventsErr    error
	MockPublicEvents        []*github.Event
	MockPublicEventsErr     error

	// Custom mock functions
	GetPullRequestsFunc        func(ctx context.Context, owner, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error)
	ListPullRequestReviewsFunc func(ctx context.Context, owner, repo string, number int, opts *github.ListOptions) ([]*github.PullRequestReview, *github.Response, error)
	ListUserRepositoriesFunc   func(ctx context.Context, visibility string) ([]*github.Repository, error)
	ListOrgRepositoriesFunc    func(ctx context.Context, org string, visibility string) ([]*github.Repository, error)
	ListRepositoryEventsFunc   func(ctx context.Context, owner, repo string) ([]*github.Event, error)
	ListUserOrgEventsFunc      func(ctx context.Context, org, user string) ([]*github.Event, error)
	ListPublicEventsFunc       func(ctx context.Context) ([]*github.Event, error)

	// Tracking calls
	GetPullRequestsCalls              int
	ListPullRequestReviewsCalls       int
	ExecuteWithRateLimitCalls         int
	ListUserRepositoriesCalls         int
	ListOrganizationRepositoriesCalls int
	ListRepositoryEventsCalls         int
	ListUserOrgEventsCalls            int
	ListPublicEventsCalls             int
}

// ExecuteWithRateLimit is a mock implementation
func (m *MockGitHubClient) ExecuteWithRateLimit(_ context.Context, f func() error) error {
	m.ExecuteWithRateLimitCalls++
	if m.MockExecuteRateLimitErr != nil {
		return m.MockExecuteRateLimitErr
	}
	return f()
}

// GetPullRequests is a mock implementation
func (m *MockGitHubClient) GetPullRequests(ctx context.Context, owner, repo string, opts *github.PullRequestListOptions) ([]*github.PullRequest, *github.Response, error) {
	m.GetPullRequestsCalls++

	// Use custom function if provided
	if m.GetPullRequestsFunc != nil {
		return m.GetPullRequestsFunc(ctx, owner, repo, opts)
	}

	return m.MockPullRequests, m.MockPullRequestResp, m.MockPullRequestErr
}

// ListPullRequestReviews is a mock implementation
func (m *MockGitHubClient) ListPullRequestReviews(ctx context.Context, owner, repo string, number int, opts *github.ListOptions) ([]*github.PullRequestReview, *github.Response, error) {
	m.ListPullRequestReviewsCalls++

	// Use custom function if provided
	if m.ListPullRequestReviewsFunc != nil {
		return m.ListPullRequestReviewsFunc(ctx, owner, repo, number, opts)
	}

	return m.MockReviews, m.MockReviewResp, m.MockReviewErr
}

// ListUserRepositories is a mock implementation
func (m *MockGitHubClient) ListUserRepositories(ctx context.Context, visibility string) ([]*github.Repository, error) {
	m.ListUserRepositoriesCalls++

	// Use custom function if provided
	if m.ListUserRepositoriesFunc != nil {
		return m.ListUserRepositoriesFunc(ctx, visibility)
	}

	return m.MockRepositories, m.MockRepositoriesErr
}

// ListOrganizationRepositories is a mock implementation
func (m *MockGitHubClient) ListOrganizationRepositories(ctx context.Context, org string, visibility string) ([]*github.Repository, error) {
	m.ListOrganizationRepositoriesCalls++

	// Use custom function if provided
	if m.ListOrgRepositoriesFunc != nil {
		return m.ListOrgRepositoriesFunc(ctx, org, visibility)
	}

	return m.MockOrgRepositories, m.MockOrgRepositoriesErr
}

// ListRepositoryEvents is a mock implementation
func (m *MockGitHubClient) ListRepositoryEvents(ctx context.Context, owner, repo string) ([]*github.Event, error) {
	m.ListRepositoryEventsCalls++

	// Use custom function if provided
	if m.ListRepositoryEventsFunc != nil {
		return m.ListRepositoryEventsFunc(ctx, owner, repo)
	}

	return m.MockRepoEvents, m.MockRepoEventsErr
}

// ListUserEventsForOrganization is a mock implementation
func (m *MockGitHubClient) ListUserEventsForOrganization(ctx context.Context, org, user string) ([]*github.Event, error) {
	m.ListUserOrgEventsCalls++

	// Use custom function if provided
	if m.ListUserOrgEventsFunc != nil {
		return m.ListUserOrgEventsFunc(ctx, org, user)
	}

	return m.MockUserOrgEvents, m.MockUserOrgEventsErr
}

// ListRepositoryPublicEvents is a mock implementation
func (m *MockGitHubClient) ListRepositoryPublicEvents(ctx context.Context) ([]*github.Event, error) {
	m.ListPublicEventsCalls++

	// Use custom function if provided
	if m.ListPublicEventsFunc != nil {
		return m.ListPublicEventsFunc(ctx)
	}

	return m.MockPublicEvents, m.MockPublicEventsErr
}
