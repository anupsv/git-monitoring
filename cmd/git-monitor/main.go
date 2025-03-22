package main

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"log"
	"net/http"
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
		// We don't print to console here anymore, just return the results
		// The caller will handle capturing the output
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
		if !useMarkdown {
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
		log.Printf("Creating directory: %s", dir)
		// Create directory with permissive permissions (0755)
		if err := os.MkdirAll(dir, 0755); err != nil {
			log.Printf("Error creating directory %s: %v", dir, err)
			return false
		}

		// Explicitly set permissions on the directory to ensure it's accessible
		if err := os.Chmod(dir, 0755); err != nil {
			log.Printf("Warning: Failed to set permissions on directory %s: %v", dir, err)
			// Continue anyway - we'll try to create the file
		}

		// Log directory info
		if info, err := os.Stat(dir); err == nil {
			log.Printf("Directory %s created with mode: %v", dir, info.Mode())
		} else {
			log.Printf("Warning: Could not stat directory %s: %v", dir, err)
		}
	}

	// Use 0600 permissions (read/write for owner only) for better security
	log.Printf("Writing markdown results to %s", outputPath)
	if err := os.WriteFile(outputPath, []byte(content), 0600); err != nil {
		log.Printf("Error writing markdown results to file %s: %v", outputPath, err)

		// Fallback: Try to write to a file in the current directory
		fallbackPath := filepath.Base(outputPath)
		log.Printf("Attempting to write to fallback location: %s", fallbackPath)
		if err := os.WriteFile(fallbackPath, []byte(content), 0600); err != nil {
			log.Printf("Error writing to fallback location %s: %v", fallbackPath, err)

			// Print content with special markers for extraction
			fmt.Println("\n--- MARKDOWN_OUTPUT_START ---")
			fmt.Println(content)
			fmt.Println("--- MARKDOWN_OUTPUT_END ---")
			fmt.Println("\nCouldn't write to file. Use the marked output above.")
			return false
		}

		fmt.Printf("\nMarkdown results written to fallback location: %s\n", fallbackPath)
		return true
	}

	// Log file info
	if info, err := os.Stat(outputPath); err == nil {
		log.Printf("File %s created with mode: %v, size: %d bytes", outputPath, info.Mode(), info.Size())
	} else {
		log.Printf("Warning: Could not stat file %s: %v", outputPath, err)
	}

	fmt.Printf("\nMarkdown results written to %s\n", outputPath)
	return true
}

// sendToSlack sends the markdown content directly to a Slack webhook
func sendToSlack(webhookURL string, content string) bool {
	log.Printf("Preparing to send results to Slack webhook")

	// Format content for Slack - wrap in a code block
	summary := "Git Monitoring Results"

	// Extract first header as summary if available
	contentLines := strings.Split(content, "\n")
	for _, line := range contentLines {
		if strings.HasPrefix(line, "## ") {
			summary = strings.TrimPrefix(line, "## ")
			break
		}
	}

	// Create the Slack payload
	type SlackText struct {
		Type string `json:"type"`
		Text string `json:"text"`
	}

	type SlackBlock struct {
		Type string    `json:"type"`
		Text SlackText `json:"text,omitempty"`
	}

	type SlackPayload struct {
		Text   string       `json:"text"`
		Blocks []SlackBlock `json:"blocks"`
	}

	// Create a message with code block formatting
	formattedText := fmt.Sprintf("*%s*\n\n```\n%s\n```", summary, content)

	// Slack has a 3000 character limit for block text
	if len(formattedText) > 3000 {
		formattedText = formattedText[:2950] + "...\n```\n(Content truncated due to size limits)"
	}

	payload := SlackPayload{
		Text: summary,
		Blocks: []SlackBlock{
			{
				Type: "section",
				Text: SlackText{
					Type: "mrkdwn",
					Text: formattedText,
				},
			},
		},
	}

	// Convert payload to JSON
	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		log.Printf("Error creating JSON payload: %v", err)
		return false
	}

	// Print masked webhook URL for debugging
	if len(webhookURL) > 10 {
		maskedURL := webhookURL[:8] + "..." + webhookURL[len(webhookURL)-10:]
		log.Printf("Sending to webhook URL (masked): %s", maskedURL)
	} else {
		log.Printf("Webhook URL is too short, might be invalid")
	}

	// Basic validation to ensure the URL is HTTPS (more permissive)
	if !strings.HasPrefix(webhookURL, "https://") {
		log.Printf("Invalid Slack webhook URL: URL must begin with https://")
		log.Printf("Please check your webhook URL and ensure it starts with https://")
		return false
	}

	// Log request details
	log.Printf("Sending payload to Slack (size: %d bytes)", len(jsonPayload))

	// Send request to Slack
	// #nosec G107 -- URL is validated above to use HTTPS
	resp, err := http.Post(webhookURL, "application/json", bytes.NewBuffer(jsonPayload))
	if err != nil {
		log.Printf("Error sending to Slack: %v", err)
		log.Printf("Network details: %T", err)
		return false
	}
	defer resp.Body.Close()

	// Check response
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		log.Printf("Slack API error: Status: %d, Response: %s", resp.StatusCode, string(body))
		log.Printf("Headers: %v", resp.Header)
		return false
	}

	log.Printf("Successfully sent results to Slack webhook (HTTP %d)", resp.StatusCode)
	return true
}

// getMarkdownOutputPath returns the path to write markdown results to
// It checks command-line flag, environment variables, and falls back to a default
func getMarkdownOutputPath(outputFlag string) string {
	// If flag is set, use it
	if outputFlag != "" {
		log.Printf("Using output path from command line flag: %s", outputFlag)
		return outputFlag
	}

	// Check environment variables
	if path := os.Getenv("MARKDOWN_OUTPUT_PATH"); path != "" {
		log.Printf("Using output path from MARKDOWN_OUTPUT_PATH env var: %s", path)
		return path
	}

	// Check if we're in a GitHub Actions environment
	if os.Getenv("GITHUB_ACTIONS") == "true" {
		// GitHub Actions - use workspace directory if available
		if workspace := os.Getenv("GITHUB_WORKSPACE"); workspace != "" {
			path := filepath.Join(workspace, "markdown-result.md")
			log.Printf("In GitHub Actions, using workspace path: %s", path)
			return path
		}
		// Alternative: use temp directory which should be writable
		path := filepath.Join(os.TempDir(), "markdown-result.md")
		log.Printf("In GitHub Actions but no workspace, using temp dir: %s", path)
		return path
	}

	// Default fallback
	log.Printf("Using default output path: markdown-result.md")
	return "markdown-result.md"
}

func main() {
	// Define command line flags
	configPath := flag.String("config", "config.toml", "Path to configuration file")
	markdownOutput := flag.Bool("markdown", true, "Output results in Markdown format for Slack (default)")
	outputPath := flag.String("output", "", "Path to write markdown results (default: markdown-result.md)")
	slackWebhook := flag.String("slack", "", "Slack webhook URL to post results directly (overrides file output)")
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

		// Capture output for markdown file or Slack
		if *markdownOutput && len(prResults) > 0 {
			output := captureOutput(func() {
				prchecker.PrintResultsMarkdown(prResults)
			})
			markdownBuilder.WriteString(output)

			// Only print to console if not sending to Slack
			if *slackWebhook == "" {
				fmt.Print(output)
			}
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

		// Capture output for markdown file or Slack
		if *markdownOutput && len(repoResults) > 0 {
			output := captureOutput(func() {
				repovisibility.PrintResultsMarkdown(repoResults)
			})
			markdownBuilder.WriteString(output)

			// Only print to console if not sending to Slack
			if *slackWebhook == "" {
				fmt.Print(output)
			}
		}
	} else if !*markdownOutput {
		fmt.Println("Repository Visibility monitor is disabled in configuration")
	}

	// Determine content to write or send
	var content string
	if markdownBuilder.Len() > 0 {
		content = markdownBuilder.String()
	} else {
		// Write a simple message when no issues were found
		content = "## :white_check_mark: No Issues Found\n\nAll repositories are compliant with policies.\n"
	}

	// If Slack webhook is provided, send results directly to Slack
	if *slackWebhook != "" {
		log.Printf("Slack webhook provided, sending results directly")
		if sendToSlack(*slackWebhook, content) {
			fmt.Println("Results sent to Slack successfully")
			// Optionally print the content to console as well for visibility
			if *markdownOutput {
				fmt.Println("\nContent sent to Slack:")
				fmt.Println("-----------------------------------")
				fmt.Println(content)
				fmt.Println("-----------------------------------")
			}
		} else {
			fmt.Println("Failed to send results to Slack")
			// Print to console as fallback
			fmt.Println("\n--- MARKDOWN_OUTPUT_START ---")
			fmt.Println(content)
			fmt.Println("--- MARKDOWN_OUTPUT_END ---")
		}
	} else if *markdownOutput {
		// Otherwise, try to write to file if markdown output is enabled
		mdOutputPath := getMarkdownOutputPath(*outputPath)
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
