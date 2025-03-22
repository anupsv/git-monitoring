package repovisibility

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/anupsv/git-monitoring/pkg/config"
	"github.com/anupsv/git-monitoring/pkg/tools/common"
	"github.com/google/go-github/v45/github"
)

const (
	// DefaultCheckWindow is the default time window to check for visibility changes
	DefaultCheckWindow = 24 * time.Hour
)

// Checker is a service that checks for repositories that were made public
type Checker struct {
	client      common.GitHubClientInterface
	checkWindow time.Duration
	config      *config.Config
}

// NewRepoVisibilityChecker creates a new Checker
func NewRepoVisibilityChecker(client common.GitHubClientInterface, config *config.Config) *Checker {
	checkWindow := DefaultCheckWindow
	if config.Monitors.RepoVisibility.CheckWindow > 0 {
		checkWindow = time.Duration(config.Monitors.RepoVisibility.CheckWindow) * time.Hour
	}

	return &Checker{
		client:      client,
		checkWindow: checkWindow,
		config:      config,
	}
}

// CheckOrganization checks an organization for repositories that were made public
func (r *Checker) CheckOrganization(ctx context.Context, orgName string) ([]string, error) {
	log.Printf("Checking for public repositories in %s organization within the last %v", orgName, r.checkWindow)

	// Get all public repositories for the organization
	repos, err := r.client.ListOrganizationRepositories(ctx, orgName, "public-only")
	if err != nil {
		return nil, fmt.Errorf("failed to list organization repositories: %w", err)
	}

	// Filter repositories by creation date and check events
	recentlyPublic := make([]string, 0)
	cutoffTime := time.Now().Add(-r.checkWindow)

	for _, repo := range repos {
		// If CreatedAt is nil, we'll consider it was created recently (for testing purposes)
		isRecent := true
		if repo.CreatedAt != nil {
			isRecent = !repo.GetCreatedAt().Before(cutoffTime)
		}

		if isRecent {
			// New repositories created within our window that are public
			recentlyPublic = append(recentlyPublic, fmt.Sprintf("%s/%s", orgName, repo.GetName()))
		} else {
			// For older repos, we need to check if they were recently made public
			madePublic, err := r.wasRecentlyMadePublic(ctx, orgName, repo.GetName())
			if err != nil {
				log.Printf("Error checking events for %s/%s: %v", orgName, repo.GetName(), err)
				continue
			}

			if madePublic {
				recentlyPublic = append(recentlyPublic, fmt.Sprintf("%s/%s", orgName, repo.GetName()))
			}
		}
	}

	return recentlyPublic, nil
}

// wasRecentlyMadePublic checks if a repository was made public within the check window
func (r *Checker) wasRecentlyMadePublic(ctx context.Context, owner, repo string) (bool, error) {
	// Get repository events
	events, err := r.client.ListRepositoryEvents(ctx, owner, repo)
	if err != nil {
		return false, fmt.Errorf("failed to list repository events: %w", err)
	}

	cutoffTime := time.Now().Add(-r.checkWindow)

	// Look for public event
	for _, event := range events {
		// If CreateAt is nil (in tests), consider it recent
		isInWindow := true
		if event.CreatedAt != nil {
			isInWindow = !event.GetCreatedAt().Before(cutoffTime)
		}

		// Stop checking if we're past the cutoff time
		if !isInWindow {
			return false, nil
		}

		// Check if this is a visibility change event
		if event.GetType() == "PublicEvent" {
			return true, nil
		}
	}

	return false, nil
}

// Run checks repositories based on configuration settings
func (r *Checker) Run(ctx context.Context) ([]string, error) {
	allPublicRepos := make([]string, 0)

	// Determine which repositories to check based on visibility setting
	switch r.config.Monitors.RepoVisibility.RepoVisibility {
	case "specific":
		// When using "specific" visibility, check only the specified organizations
		for _, org := range r.config.Monitors.RepoVisibility.Organizations {
			repos, err := r.CheckOrganization(ctx, org)
			if err != nil {
				log.Printf("Error checking organization %s: %v", org, err)
				continue
			}
			allPublicRepos = append(allPublicRepos, repos...)
		}

	case "all", "public-only", "private-only":
		// Check all organizations listed in the config with the selected visibility
		for _, org := range r.config.Monitors.RepoVisibility.Organizations {
			repos, err := r.CheckOrganizationWithVisibility(ctx, org, r.config.Monitors.RepoVisibility.RepoVisibility)
			if err != nil {
				log.Printf("Error checking organization %s: %v", org, err)
				continue
			}
			allPublicRepos = append(allPublicRepos, repos...)
		}

	default:
		return nil, fmt.Errorf("invalid repository visibility setting: %s", r.config.Monitors.RepoVisibility.RepoVisibility)
	}

	return allPublicRepos, nil
}

// CheckRepository checks a specific repository for visibility changes
func (r *Checker) CheckRepository(ctx context.Context, owner, repo string) (bool, error) {
	log.Printf("Checking repository %s/%s for visibility changes within the last %v", owner, repo, r.checkWindow)

	// Try to get the repository
	repos, err := r.client.ListOrganizationRepositories(ctx, owner, "public-only")
	if err != nil {
		return false, fmt.Errorf("failed to list repositories: %w", err)
	}

	// Check if our repo is in the list of public repos
	var found bool
	var foundRepo *github.Repository
	for _, r := range repos {
		if r.GetName() == repo {
			found = true
			foundRepo = r
			break
		}
	}

	if !found {
		// Repository is not public or doesn't exist
		return false, nil
	}

	cutoffTime := time.Now().Add(-r.checkWindow)

	// If recently created and public, consider it recently made public
	if foundRepo.CreatedAt != nil && !foundRepo.GetCreatedAt().Before(cutoffTime) {
		return true, nil
	}

	// Check if repository was recently made public
	madePublic, err := r.wasRecentlyMadePublic(ctx, owner, repo)
	if err != nil {
		log.Printf("Error checking events for %s/%s: %v", owner, repo, err)
		return false, err
	}

	return madePublic, nil
}

// CheckOrganizationWithVisibility checks an organization's repositories with the specified visibility
func (r *Checker) CheckOrganizationWithVisibility(ctx context.Context, orgName, visibility string) ([]string, error) {
	log.Printf("Checking for public repositories in %s organization with visibility %s within the last %v",
		orgName, visibility, r.checkWindow)

	// When checking for public repos, we only need to list public repositories
	// For all or private, we need to check which public repos were previously private
	var visibilityFilter string
	if visibility == "public-only" {
		visibilityFilter = "public-only"
	} else {
		// For "all" or "private-only", we need to check all repos and filter later
		visibilityFilter = visibility
	}

	// Get repositories for the organization based on visibility
	repos, err := r.client.ListOrganizationRepositories(ctx, orgName, visibilityFilter)
	if err != nil {
		return nil, fmt.Errorf("failed to list organization repositories: %w", err)
	}

	// Filter repositories
	recentlyPublic := make([]string, 0)
	cutoffTime := time.Now().Add(-r.checkWindow)

	for _, repo := range repos {
		// Skip private repos if we're only interested in public ones
		if visibility == "public-only" && repo.GetPrivate() {
			continue
		}

		// Skip public repos if we're only interested in private ones
		if visibility == "private-only" && !repo.GetPrivate() {
			continue
		}

		// For non-private repos, check if they're recently public
		if !repo.GetPrivate() {
			// If created recently and public, consider it recently made public
			isRecent := true
			if repo.CreatedAt != nil {
				isRecent = !repo.GetCreatedAt().Before(cutoffTime)
			}

			if isRecent {
				// New repositories created within our window that are public
				recentlyPublic = append(recentlyPublic, fmt.Sprintf("%s/%s", orgName, repo.GetName()))
			} else {
				// For older repos, we need to check if they were recently made public
				madePublic, err := r.wasRecentlyMadePublic(ctx, orgName, repo.GetName())
				if err != nil {
					log.Printf("Error checking events for %s/%s: %v", orgName, repo.GetName(), err)
					continue
				}

				if madePublic {
					recentlyPublic = append(recentlyPublic, fmt.Sprintf("%s/%s", orgName, repo.GetName()))
				}
			}
		}
	}

	return recentlyPublic, nil
}

// PrintResultsMarkdown outputs recently public repositories in a Markdown table format
// suitable for Slack notifications
func PrintResultsMarkdown(recentlyPublic []string) {
	if len(recentlyPublic) == 0 {
		return // No results to display
	}

	// Print header for public repository issues
	fmt.Println("## :warning: Recently Public Repositories")
	fmt.Println("")
	fmt.Println("| Repository | Action Needed |")
	fmt.Println("|------------|---------------|")

	// Print each public repository in a table row
	for _, repo := range recentlyPublic {
		fmt.Printf("| %s | Review visibility settings |\n", repo)
	}

	fmt.Println("")
}
