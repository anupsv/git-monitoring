package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/asv/git-monitoring/pkg/config"
	"github.com/asv/git-monitoring/pkg/tools/prchecker"
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

	// Run monitors based on configuration
	if cfg.Monitors.PRChecker.Enabled {
		fmt.Println("Running PR Checker monitor...")
		results := prchecker.Monitor(cfg)

		allApproved := prchecker.PrintResults(results)
		if !allApproved {
			os.Exit(1)
		}
	} else {
		fmt.Println("PR Checker monitor is disabled in configuration")
	}

	fmt.Println("All monitors completed successfully")
}
