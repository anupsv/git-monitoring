package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"strings"

	"github.com/anupsv/git-monitoring/pkg/config"
	"github.com/anupsv/git-monitoring/pkg/tools/common"
	"github.com/anupsv/git-monitoring/pkg/tools/prchecker"
	"github.com/anupsv/git-monitoring/pkg/tools/repovisibility"
)

func main() {
	// Define command line flags
	configPath := flag.String("config", "config.toml", "Path to configuration file")
	markdownOutput := flag.Bool("markdown", true, "Output results in Markdown format for Slack (default)")
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
	// Collect problematic results for markdown output
	var markdownOutput1 []prchecker.Result
	var markdownOutput2 []string
	// String builder to collect markdown output
	var markdownBuilder strings.Builder

	// Capture output for markdown file
	captureOutput := func(f func()) string {
		old := os.Stdout
		r, w, _ := os.Pipe()
		os.Stdout = w

		f()

		w.Close()
		os.Stdout = old

		var buf strings.Builder
		_, _ = io.Copy(&buf, r)
		return buf.String()
	}

	// Run monitors based on configuration
	if cfg.Monitors.PRChecker.Enabled {
		if !*markdownOutput {
			fmt.Println("Running PR Checker monitor...")
		}
		results := prchecker.Monitor(cfg)

		// Check if any results contain errors
		for _, result := range results {
			if result.Error != nil {
				monitorFailed = true
				break
			}
			// Save problematic results for markdown output
			if len(result.UnapprovedPRs) > 0 {
				markdownOutput1 = append(markdownOutput1, result)
			}
		}

		// Print results based on output format
		if *markdownOutput {
			// Capture markdown output
			output := captureOutput(func() {
				prchecker.PrintResultsMarkdown(markdownOutput1)
			})
			// Add to markdown builder for file
			markdownBuilder.WriteString(output)
			// Print to console
			fmt.Print(output)
		} else {
			prchecker.PrintResults(results)
		}
	} else if !*markdownOutput {
		fmt.Println("PR Checker monitor is disabled in configuration")
	}

	// Run the repository visibility checker if enabled
	if cfg.Monitors.RepoVisibility.Enabled {
		if !*markdownOutput {
			fmt.Println("Running Repository Visibility monitor...")
		}

		// Create GitHub client
		client := common.NewGitHubClient(context.Background(), cfg.GitHub.Token)

		// Create and run the visibility checker
		checker := repovisibility.NewRepoVisibilityChecker(client, cfg)
		recentlyPublic, err := checker.Run(context.Background())

		if err != nil {
			log.Printf("Error checking repository visibility: %v", err)
			monitorFailed = true
		} else if len(recentlyPublic) > 0 {
			markdownOutput2 = recentlyPublic
			if *markdownOutput {
				// Capture markdown output
				output := captureOutput(func() {
					repovisibility.PrintResultsMarkdown(recentlyPublic)
				})
				// Add to markdown builder for file
				markdownBuilder.WriteString(output)
				// Print to console
				fmt.Print(output)
			} else {
				fmt.Println("WARNING: The following repositories were recently made public:")
				for _, repo := range recentlyPublic {
					fmt.Printf("  - %s\n", repo)
				}
			}
			// Finding recently public repos is not an error, just a condition to report
		} else if !*markdownOutput {
			fmt.Println("No organization repositories were recently made public")
		}
	} else if !*markdownOutput {
		fmt.Println("Repository Visibility monitor is disabled in configuration")
	}

	// Write markdown results to file if we have any content and --markdown is enabled
	if *markdownOutput && markdownBuilder.Len() > 0 {
		err := os.WriteFile("markdown-result.md", []byte(markdownBuilder.String()), 0644)
		if err != nil {
			log.Printf("Error writing markdown results to file: %v", err)
		} else {
			fmt.Println("\nMarkdown results written to markdown-result.md")
		}
	} else if *markdownOutput && markdownBuilder.Len() == 0 {
		// Write a simple message when no issues were found
		noIssuesMessage := "## :white_check_mark: No Issues Found\n\nAll repositories are compliant with policies.\n"
		err := os.WriteFile("markdown-result.md", []byte(noIssuesMessage), 0644)
		if err != nil {
			log.Printf("Error writing markdown results to file: %v", err)
		} else {
			fmt.Println("\nNo issues found. Markdown results written to markdown-result.md")
		}
	}

	if monitorFailed {
		if !*markdownOutput {
			fmt.Println("One or more monitors encountered processing errors")
		}
		os.Exit(1)
	}

	// Only show "completed successfully" if there are no problematic results
	if !*markdownOutput && len(markdownOutput1) == 0 && len(markdownOutput2) == 0 {
		fmt.Println("All monitors completed successfully")
	}
}
