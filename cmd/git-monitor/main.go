package main

import (
	"context"
	"flag"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/anupsv/git-monitoring/pkg/config"
	"github.com/anupsv/git-monitoring/pkg/tools/common"
	"github.com/anupsv/git-monitoring/pkg/tools/prchecker"
	"github.com/anupsv/git-monitoring/pkg/tools/repovisibility"
)

// captureOutput captures stdout output from a function
func captureOutput(f func()) string {
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

// runPRChecker runs the PR checker monitor
func runPRChecker(cfg *config.Config, useMarkdown bool) ([]prchecker.Result, bool) {
	var problematicResults []prchecker.Result
	monitorFailed := false

	if !useMarkdown {
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
			problematicResults = append(problematicResults, result)
		}
	}

	// Print results based on output format
	if useMarkdown {
		// Capture markdown output
		output := captureOutput(func() {
			prchecker.PrintResultsMarkdown(problematicResults)
		})
		// Print to console
		fmt.Print(output)
		return problematicResults, monitorFailed
	}

	prchecker.PrintResults(results)
	return problematicResults, monitorFailed
}

// runRepoVisibilityChecker runs the repository visibility checker
func runRepoVisibilityChecker(cfg *config.Config, useMarkdown bool) ([]string, bool) {
	monitorFailed := false

	if !useMarkdown {
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
		return nil, monitorFailed
	}

	if len(recentlyPublic) > 0 {
		if useMarkdown {
			// Capture markdown output
			output := captureOutput(func() {
				repovisibility.PrintResultsMarkdown(recentlyPublic)
			})
			// Print to console
			fmt.Print(output)
		} else {
			fmt.Println("WARNING: The following repositories were recently made public:")
			for _, repo := range recentlyPublic {
				fmt.Printf("  - %s\n", repo)
			}
		}
		return recentlyPublic, monitorFailed
	}

	if !useMarkdown {
		fmt.Println("No organization repositories were recently made public")
	}

	return nil, monitorFailed
}

// writeMarkdownToFile writes the markdown results to a file
// Returns true if writing was successful, false otherwise
func writeMarkdownToFile(outputPath string, content string) bool {
	// Ensure directory exists if a path is specified
	dir := filepath.Dir(outputPath)
	if dir != "." && dir != "/" {
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Error creating directory %s: %v", dir, err)
			return false
		}
	}

	// Use 0600 permissions (read/write for owner only) for better security
	if err := os.WriteFile(outputPath, []byte(content), 0600); err != nil {
		log.Printf("Error writing markdown results to file %s: %v", outputPath, err)
		return false
	}

	fmt.Printf("\nMarkdown results written to %s\n", outputPath)
	return true
}

// getMarkdownOutputPath returns the path to write markdown results to
// It checks command-line flag, environment variables, and falls back to a default
func getMarkdownOutputPath(outputFlag string) string {
	// If flag is set, use it
	if outputFlag != "" {
		return outputFlag
	}

	// Check environment variables
	if path := os.Getenv("MARKDOWN_OUTPUT_PATH"); path != "" {
		return path
	}

	// Check if we're in a GitHub Actions environment
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		// GitHub Actions - use workspace directory if available
		if workspace := os.Getenv("GITHUB_WORKSPACE"); workspace != "" {
			return filepath.Join(workspace, "markdown-result.md")
		}
		// Alternative: use temp directory which should be writable
		return filepath.Join(os.TempDir(), "markdown-result.md")
	}

	// Default fallback
	return "markdown-result.md"
}

func main() {
	// Define command line flags
	configPath := flag.String("config", "config.toml", "Path to configuration file")
	markdownOutput := flag.Bool("markdown", true, "Output results in Markdown format for Slack (default)")
	outputPath := flag.String("output", "", "Path to write markdown results (default: markdown-result.md)")
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
	// String builder to collect markdown output
	var markdownBuilder strings.Builder

	// Run PR checker if enabled
	var prResults []prchecker.Result
	if cfg.Monitors.PRChecker.Enabled {
		var prFailed bool
		prResults, prFailed = runPRChecker(cfg, *markdownOutput)
		if prFailed {
			monitorFailed = true
		}

		// Capture output for markdown file
		if *markdownOutput && len(prResults) > 0 {
			output := captureOutput(func() {
				prchecker.PrintResultsMarkdown(prResults)
			})
			markdownBuilder.WriteString(output)
		}
	} else if !*markdownOutput {
		fmt.Println("PR Checker monitor is disabled in configuration")
	}

	// Run repository visibility checker if enabled
	var repoResults []string
	if cfg.Monitors.RepoVisibility.Enabled {
		var repoFailed bool
		repoResults, repoFailed = runRepoVisibilityChecker(cfg, *markdownOutput)
		if repoFailed {
			monitorFailed = true
		}

		// Capture output for markdown file
		if *markdownOutput && len(repoResults) > 0 {
			output := captureOutput(func() {
				repovisibility.PrintResultsMarkdown(repoResults)
			})
			markdownBuilder.WriteString(output)
		}
	} else if !*markdownOutput {
		fmt.Println("Repository Visibility monitor is disabled in configuration")
	}

	// Determine content to write
	var content string
	if markdownBuilder.Len() > 0 {
		content = markdownBuilder.String()
	} else {
		// Write a simple message when no issues were found
		content = "## :white_check_mark: No Issues Found\n\nAll repositories are compliant with policies.\n"
	}

	// Get the output path
	mdOutputPath := getMarkdownOutputPath(*outputPath)

	// Try to write to the file
	if *markdownOutput {
		fileWritten := writeMarkdownToFile(mdOutputPath, content)

		if !fileWritten {
			// If we couldn't write to the file, print the content with special markers
			// for easy extraction in GitHub Actions
			fmt.Println("\n--- MARKDOWN_OUTPUT_START ---")
			fmt.Println(content)
			fmt.Println("--- MARKDOWN_OUTPUT_END ---")
			fmt.Println("\nCouldn't write to file. Use the marked output above for webhook integration.")
		}
	}

	if monitorFailed {
		if !*markdownOutput {
			fmt.Println("One or more monitors encountered processing errors")
		}
		os.Exit(1)
	}

	// Only show "completed successfully" if there are no problematic results
	if !*markdownOutput && len(prResults) == 0 && len(repoResults) == 0 {
		fmt.Println("All monitors completed successfully")
	}
}
