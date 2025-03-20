package prchecker

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/anupsv/git-monitoring/pkg/config"
	"github.com/anupsv/git-monitoring/pkg/tools/common"
	"github.com/google/go-github/v45/github"
)

// Result represents the result of checking a repository
type Result struct {
	Repository    string
	UnapprovedPRs []PR
	Error         error
}

// PR represents a pull request with essential information
type PR struct {
	Number int
	Title  string
	Author string
	URL    string
}

// MonitorService is the interface for the PR checker service
type MonitorService interface {
	CheckRepository(repository string, token string, timeWindow int) Result
}

// Service implements the MonitorService interface
type Service struct {
	NewClient func(ctx context.Context, token string) common.GitHubClientInterface
}

// NewService creates a new PR checker service
func NewService() *Service {
	return &Service{
		NewClient: func(ctx context.Context, token string) common.GitHubClientInterface {
			return common.NewGitHubClient(ctx, token)
		},
	}
}

// Monitor checks all repositories in the configuration for unapproved PRs
func Monitor(cfg *config.Config) []Result {
	if !cfg.Monitors.PRChecker.Enabled {
		return nil
	}

	return MonitorWithService(cfg, NewService())
}

// MonitorWithService is a testable version of Monitor that accepts a custom service
// This makes it easier to test with mock services
func MonitorWithService(cfg *config.Config, service *Service) []Result {
	if !cfg.Monitors.PRChecker.Enabled {
		return nil
	}

	ctx := context.Background()

	var repositories []string

	// Determine which repositories to check based on visibility setting
	switch cfg.Monitors.PRChecker.RepoVisibility {
	case "specific":
		// Use the specifically listed repositories in the config
		repositories = cfg.Monitors.PRChecker.SpecificRepositories
	case "all", "public-only", "private-only":
		// Fetch repositories based on visibility and organization
		client := service.NewClient(ctx, cfg.GitHub.Token)
		var repos []*github.Repository
		var err error

		if cfg.Monitors.PRChecker.Organization != "" {
			// Fetch repositories from the specified organization
			fmt.Printf("Fetching repositories for organization '%s' with visibility '%s'...\n",
				cfg.Monitors.PRChecker.Organization, cfg.Monitors.PRChecker.RepoVisibility)
			repos, err = client.ListOrganizationRepositories(ctx, cfg.Monitors.PRChecker.Organization, cfg.Monitors.PRChecker.RepoVisibility)
			if err != nil {
				return []Result{
					{
						Repository: "org:" + cfg.Monitors.PRChecker.Organization,
						Error:      fmt.Errorf("failed to fetch organization repositories: %v", err),
					},
				}
			}
			fmt.Printf("Found %d repositories for organization '%s' with visibility '%s'\n",
				len(repos), cfg.Monitors.PRChecker.Organization, cfg.Monitors.PRChecker.RepoVisibility)
		} else {
			// Fetch repositories for the authenticated user
			fmt.Printf("Fetching repositories for authenticated user with visibility '%s'...\n",
				cfg.Monitors.PRChecker.RepoVisibility)
			repos, err = client.ListUserRepositories(ctx, cfg.Monitors.PRChecker.RepoVisibility)
			if err != nil {
				return []Result{
					{
						Repository: "user-repositories",
						Error:      fmt.Errorf("failed to fetch user repositories: %v", err),
					},
				}
			}
			fmt.Printf("Found %d repositories for authenticated user with visibility '%s'\n",
				len(repos), cfg.Monitors.PRChecker.RepoVisibility)
		}

		// Create a map of excluded repositories for faster lookup
		excludedRepos := make(map[string]bool)
		for _, repo := range cfg.Monitors.PRChecker.ExcludedRepositories {
			excludedRepos[repo] = true
		}

		// Extract full name (owner/repo) from each repository, excluding any in the excluded list
		for _, repo := range repos {
			repoFullName := repo.GetFullName()
			if !excludedRepos[repoFullName] {
				repositories = append(repositories, repoFullName)
			} else {
				fmt.Printf("Excluding repository: %s (found in excluded_repositories list)\n", repoFullName)
			}
		}

		if len(cfg.Monitors.PRChecker.ExcludedRepositories) > 0 {
			fmt.Printf("After applying exclusions: Processing %d repositories\n", len(repositories))
		}
	default:
		// This shouldn't happen due to config validation, but handle it anyway
		return []Result{
			{
				Repository: "all-repositories",
				Error:      fmt.Errorf("invalid repository visibility setting: %s", cfg.Monitors.PRChecker.RepoVisibility),
			},
		}
	}

	results := make([]Result, 0, len(repositories))

	fmt.Printf("Processing %d repositories...\n", len(repositories))
	for i, repo := range repositories {
		fmt.Printf("[%d/%d] Checking repository: %s\n", i+1, len(repositories), repo)
		result := service.CheckRepository(repo, cfg.GitHub.Token, cfg.Monitors.PRChecker.TimeWindow, cfg.Monitors.PRChecker.DebugLogging)
		results = append(results, result)
	}
	fmt.Printf("Completed checking all %d repositories\n", len(repositories))

	return results
}

// PrintResults prints the results of the monitoring
func PrintResults(results []Result) bool {
	allApproved := true
	var reposWithErrors []string
	var reposWithUnapprovedPRs []string
	var approvedRepos []string
	var unapprovedPRsList []string
	var errorMessages []string

	// First pass: categorize repositories
	for _, result := range results {
		if result.Error != nil {
			reposWithErrors = append(reposWithErrors, result.Repository)
			errorMessages = append(errorMessages, fmt.Sprintf("%s: %v", result.Repository, result.Error))
			allApproved = false
			continue
		}

		if len(result.UnapprovedPRs) > 0 {
			reposWithUnapprovedPRs = append(reposWithUnapprovedPRs, result.Repository)
			for _, pr := range result.UnapprovedPRs {
				unapprovedPRsList = append(unapprovedPRsList,
					fmt.Sprintf("- %s #%d: %s (created by %s) %s",
						result.Repository, pr.Number, pr.Title, pr.Author, pr.URL))
			}
			allApproved = false
		} else {
			approvedRepos = append(approvedRepos, result.Repository)
		}
	}

	// Output errors first
	if len(reposWithErrors) > 0 {
		fmt.Println("\n🔴 ERRORS ENCOUNTERED:")
		for _, errMsg := range errorMessages {
			fmt.Printf("  %s\n", errMsg)
		}
	}

	// Output unapproved PRs next
	if len(reposWithUnapprovedPRs) > 0 {
		fmt.Println("\n🔔 UNAPPROVED PULL REQUESTS:")
		for _, prInfo := range unapprovedPRsList {
			fmt.Println(prInfo)
		}
	}

	// Print summary
	fmt.Println("\n📊 SUMMARY:")
	if len(reposWithErrors) > 0 {
		fmt.Printf("  Repositories with errors: %d\n", len(reposWithErrors))
	}
	if len(reposWithUnapprovedPRs) > 0 {
		fmt.Printf("  Repositories with unapproved PRs: %d\n", len(reposWithUnapprovedPRs))
	}
	fmt.Printf("  Repositories with all PRs approved: %d\n", len(approvedRepos))
	fmt.Printf("  Total repositories checked: %d\n", len(results))

	// Print approved repos in a comma-separated list
	if len(approvedRepos) > 0 {
		fmt.Println("\n✅ REPOSITORIES WITH ALL PRS APPROVED:")
		fmt.Printf("  %s\n", strings.Join(approvedRepos, ", "))
	}

	return allApproved
}

// CheckRepository checks a single repository for unapproved PRs
func (s *Service) CheckRepository(repository, token string, timeWindow int, debugLogging bool) Result {
	result := Result{
		Repository: repository,
	}

	// Create an authenticated GitHub client
	ctx := context.Background()
	client := s.NewClient(ctx, token)

	// Parse owner and repo
	owner, repo, ok := common.ParseRepository(repository)
	if !ok {
		result.Error = fmt.Errorf("invalid repository format, expected 'owner/repo'")
		return result
	}

	// Calculate the time window
	now := time.Now()
	cutoffTime := now.Add(-time.Duration(timeWindow) * time.Hour)

	// Get pull requests that were updated within our time window
	// This is more efficient than fetching all PRs and filtering locally
	opts := &github.PullRequestListOptions{
		State:     "closed",  // We're interested in merged PRs, which are in "closed" state
		Sort:      "updated", // Sort by last updated
		Direction: "desc",    // Most recently updated first
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	if debugLogging {
		fmt.Printf("  Using time window: PRs merged since %s\n", cutoffTime.Format(time.RFC3339))
	}

	unapprovedPRs := []PR{}
	page := 1
	totalPRs := 0
	totalMergedPRsInWindow := 0
	stopFetching := false

	// Counter for consecutive PRs outside our time window
	consecutivePRsOutsideWindow := 0
	// Threshold for how many consecutive PRs outside window before stopping
	const outOfWindowThreshold = 20
	// Counter for skipped PRs (either not merged or merged before cutoff)
	skippedPRs := 0

	for {
		if stopFetching {
			break
		}

		opts.Page = page
		fmt.Printf("  Fetching PRs from %s/%s (page %d)...\n", owner, repo, page)

		prs, resp, err := client.GetPullRequests(ctx, owner, repo, opts)
		if err != nil {
			result.Error = fmt.Errorf("error getting pull requests: %v", err)
			return result
		}

		if len(prs) == 0 {
			// No more PRs to check
			break
		}

		pageSkippedPRs := 0
		mergedPRsInWindow := 0

		// Check each PR
		for _, pr := range prs {
			totalPRs++

			// If this PR was updated before our cutoff time, we can stop checking
			// since GitHub returns PRs sorted by updated_at in descending order
			updatedAt := pr.GetUpdatedAt()
			if updatedAt.Before(cutoffTime) {
				if debugLogging {
					fmt.Printf("  Found PR #%d updated at %s (before cutoff), stopping further requests\n",
						pr.GetNumber(), updatedAt.Format(time.RFC3339))
				}
				stopFetching = true
				break
			}

			// Skip PRs that haven't been merged
			if pr.GetMergedAt().IsZero() {
				pageSkippedPRs++
				skippedPRs++
				consecutivePRsOutsideWindow++
				continue
			}

			// Skip PRs merged before our timeframe
			mergedAt := pr.GetMergedAt()
			if mergedAt.Before(cutoffTime) {
				pageSkippedPRs++
				skippedPRs++
				consecutivePRsOutsideWindow++

				// If we've seen too many consecutive PRs outside our window, assume we're unlikely
				// to find more relevant PRs and stop processing
				if consecutivePRsOutsideWindow >= outOfWindowThreshold {
					if debugLogging {
						fmt.Printf("  Found %d consecutive PRs outside time window, stopping further requests\n",
							consecutivePRsOutsideWindow)
					}
					stopFetching = true
					break
				}
				continue
			}

			// This PR is in our time window, reset the counter
			consecutivePRsOutsideWindow = 0
			mergedPRsInWindow++
			totalMergedPRsInWindow++

			// Debug logging
			if debugLogging {
				fmt.Printf("  Checking PR #%d in %s/%s: %s (merged at %s)\n",
					pr.GetNumber(), owner, repo, pr.GetTitle(), mergedAt.Format(time.RFC3339))
			}

			// Check if this PR is approved
			isApproved, err := isPRApproved(ctx, client, owner, repo, pr.GetNumber(), debugLogging)
			if err != nil {
				result.Error = fmt.Errorf("error checking PR approval: %v", err)
				return result
			}

			if !isApproved {
				unapprovedPRs = append(unapprovedPRs, PR{
					Number: pr.GetNumber(),
					Title:  pr.GetTitle(),
					Author: pr.GetUser().GetLogin(),
					URL:    pr.GetHTMLURL(),
				})
			}
		}

		fmt.Printf("  Found %d PRs on page %d, %d merged within time window, %d skipped\n",
			len(prs), page, mergedPRsInWindow, pageSkippedPRs)

		// If we've reached the stop fetching flag or there are no more pages, break
		if stopFetching || resp.NextPage == 0 {
			break
		}

		// If this entire page yielded no PRs in our window, increment our threshold counter
		// This helps us stop early if multiple pages in a row have no relevant PRs
		if mergedPRsInWindow == 0 {
			consecutivePRsOutsideWindow += outOfWindowThreshold / 2
			if consecutivePRsOutsideWindow >= outOfWindowThreshold {
				if debugLogging {
					fmt.Printf("  No PRs in time window on this page, stopping further requests\n")
				}
				stopFetching = true
			}
		}

		page = resp.NextPage
	}

	fmt.Printf("  Completed checking %s: %d total PRs examined, %d merged within time window, %d skipped, %d unapproved\n",
		repository, totalPRs, totalMergedPRsInWindow, skippedPRs, len(unapprovedPRs))

	result.UnapprovedPRs = unapprovedPRs
	return result
}

// isPRApproved checks if a specific PR has been approved
func isPRApproved(ctx context.Context, client common.GitHubClientInterface, owner, repo string, prNumber int, debugLogging bool) (bool, error) {
	reviews, _, err := client.ListPullRequestReviews(ctx, owner, repo, prNumber, nil)
	if err != nil {
		return false, err
	}

	if debugLogging {
		fmt.Printf("PR #%d: Found %d reviews\n", prNumber, len(reviews))
	}

	// Track the latest review from each reviewer
	latestReviewByReviewer := make(map[string]string)

	// Process all reviews in order (GitHub returns them chronologically)
	for _, review := range reviews {
		reviewer := review.GetUser().GetLogin()
		state := review.GetState()

		if debugLogging {
			fmt.Printf("PR #%d: Review by %s with state %s (submitted at %s)\n",
				prNumber, reviewer, state, review.GetSubmittedAt().Format(time.RFC3339))
		}

		// Skip reviews with empty state or from ghost users
		if state == "" || reviewer == "" || reviewer == "ghost" {
			continue
		}

		// Only track reviews that represent a clear state (APPROVED or CHANGES_REQUESTED)
		// Ignore COMMENTED reviews as they don't change approval status
		if state == "APPROVED" || state == "CHANGES_REQUESTED" {
			latestReviewByReviewer[reviewer] = state
		}
	}

	// Check if there's at least one approval and no pending requested changes
	hasApproval := false
	for reviewer, state := range latestReviewByReviewer {
		if state == "APPROVED" {
			hasApproval = true
			if debugLogging {
				fmt.Printf("PR #%d: Has approval from %s\n", prNumber, reviewer)
			}
		} else if state == "CHANGES_REQUESTED" {
			// If any reviewer's latest review is CHANGES_REQUESTED, PR is not approved
			if debugLogging {
				fmt.Printf("PR #%d: Changes requested by %s, PR not approved\n", prNumber, reviewer)
			}
			return false, nil
		}
	}

	if debugLogging {
		if hasApproval {
			fmt.Printf("PR #%d: Is approved with no pending change requests\n", prNumber)
		} else {
			fmt.Printf("PR #%d: No approvals found\n", prNumber)
		}
	}

	return hasApproval, nil
}
