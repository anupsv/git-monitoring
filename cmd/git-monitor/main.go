package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/anupsv/git-monitoring/pkg/config"
	"github.com/anupsv/git-monitoring/pkg/tools/common"
	"github.com/anupsv/git-monitoring/pkg/tools/prchecker"
	"github.com/anupsv/git-monitoring/pkg/tools/repovisibility"
)

func main() {
	// Define command line flags
	configPath := flag.String("config", "config.toml", "Path to configuration file")
	flag.Parse()

	// Load configuration
	cfg, err := config.LoadConfig(*configPath)
	if err != nil {
		log.Fatalf("Error loading configuration: %v", err)
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		log.Fatalf("Invalid configuration: %v", err)
	}

	// Flag to track if any monitor has experienced an actual error
	monitorFailed := false

	// Run monitors based on configuration
	if cfg.Monitors.PRChecker.Enabled {
		fmt.Println("Running PR Checker monitor...")
		results := prchecker.Monitor(cfg)

		// Check if any results contain errors
		for _, result := range results {
			if result.Error != nil {
				monitorFailed = true
				break
			}
		}

		// Print results, but don't use the return value to determine exit code
		prchecker.PrintResults(results)
	} else {
		fmt.Println("PR Checker monitor is disabled in configuration")
	}

	// Run the repository visibility checker if enabled
	if cfg.Monitors.RepoVisibility.Enabled {
		fmt.Println("Running Repository Visibility monitor...")

		// Create GitHub client
		client := common.NewGitHubClient(context.Background(), cfg.GitHub.Token)

		// Create and run the visibility checker
		checker := repovisibility.NewRepoVisibilityChecker(client, cfg)
		recentlyPublic, err := checker.Run(context.Background())

		if err != nil {
			log.Printf("Error checking repository visibility: %v", err)
			monitorFailed = true
		} else if len(recentlyPublic) > 0 {
			fmt.Println("WARNING: The following repositories were recently made public:")
			for _, repo := range recentlyPublic {
				fmt.Printf("  - %s\n", repo)
			}
			// Finding recently public repos is not an error, just a condition to report
		} else {
			fmt.Println("No organization repositories were recently made public")
		}
	} else {
		fmt.Println("Repository Visibility monitor is disabled in configuration")
	}

	if monitorFailed {
		fmt.Println("One or more monitors encountered processing errors")
		os.Exit(1)
	}
	fmt.Println("All monitors completed successfully")
}
