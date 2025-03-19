package prchecker

import (
	"context"
	"fmt"
	"time"

	"github.com/asv/git-monitoring/pkg/config"
	"github.com/asv/git-monitoring/pkg/tools/common"
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

// Monitor checks all repositories in the configuration for unapproved PRs
func Monitor(cfg *config.Config) []Result {
	if !cfg.Monitors.PRChecker.Enabled {
		return nil
	}

	results := make([]Result, 0, len(cfg.Monitors.PRChecker.Repositories))
	for _, repo := range cfg.Monitors.PRChecker.Repositories {
		result := checkRepository(repo, cfg.GitHub.Token, cfg.Monitors.PRChecker.TimeWindow)
		results = append(results, result)
	}

	return results
}

// PrintResults prints the results of the monitoring
func PrintResults(results []Result) bool {
	allApproved := true

	for _, result := range results {
		if result.Error != nil {
			fmt.Printf("Error checking %s: %v\n", result.Repository, result.Error)
			allApproved = false
			continue
		}

		if len(result.UnapprovedPRs) > 0 {
			fmt.Printf("Found %d unapproved PRs in %s:\n", len(result.UnapprovedPRs), result.Repository)
			for _, pr := range result.UnapprovedPRs {
				fmt.Printf("- #%d: %s (created by %s) %s\n", pr.Number, pr.Title, pr.Author, pr.URL)
			}
			allApproved = false
		} else {
			fmt.Printf("All PRs in %s from the configured time window are approved\n", result.Repository)
		}
	}

	return allApproved
}

// checkRepository checks a single repository for unapproved PRs
func checkRepository(repository, token string, timeWindow int) Result {
	result := Result{
		Repository: repository,
	}

	// Create an authenticated GitHub client
	ctx := context.Background()
	client := common.NewGitHubClient(ctx, token)

	// Parse owner and repo
	owner, repo, ok := common.ParseRepository(repository)
	if !ok {
		result.Error = fmt.Errorf("invalid repository format, expected 'owner/repo'")
		return result
	}

	// Get pull requests
	opts := &github.PullRequestListOptions{
		State: "all",
		ListOptions: github.ListOptions{
			PerPage: 100,
		},
	}

	// Calculate the time window
	now := time.Now()
	cutoffTime := now.Add(-time.Duration(timeWindow) * time.Hour)

	unapprovedPRs := []PR{}
	page := 1

	for {
		opts.Page = page

		var prs []*github.PullRequest
		var resp *github.Response

		// Use rate limiting for the API call
		err := client.ExecuteWithRateLimit(ctx, func() error {
			var apiErr error
			prs, resp, apiErr = client.Client.PullRequests.List(ctx, owner, repo, opts)
			return apiErr
		})

		if err != nil {
			result.Error = fmt.Errorf("error getting pull requests: %v", err)
			return result
		}

		// Check each PR
		for _, pr := range prs {
			// Skip PRs created before our timeframe
			if pr.GetCreatedAt().Before(cutoffTime) {
				continue
			}

			// Check if this PR is approved
			isApproved, err := isPRApproved(ctx, client, owner, repo, pr.GetNumber())
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

		// Check if there are more pages
		if resp.NextPage == 0 {
			break
		}
		page = resp.NextPage
	}

	result.UnapprovedPRs = unapprovedPRs
	return result
}

// isPRApproved checks if a specific PR has been approved
func isPRApproved(ctx context.Context, client *common.GitHubClient, owner, repo string, prNumber int) (bool, error) {
	var reviews []*github.PullRequestReview

	err := client.ExecuteWithRateLimit(ctx, func() error {
		var apiErr error
		reviews, _, apiErr = client.Client.PullRequests.ListReviews(ctx, owner, repo, prNumber, nil)
		return apiErr
	})

	if err != nil {
		return false, err
	}

	// Check for at least one approval and no pending requested changes
	hasApproval := false
	for _, review := range reviews {
		switch review.GetState() {
		case "APPROVED":
			hasApproval = true
		case "CHANGES_REQUESTED":
			// If the most recent review from this reviewer requests changes, this PR is not approved
			return false, nil
		}
	}

	return hasApproval, nil
}
