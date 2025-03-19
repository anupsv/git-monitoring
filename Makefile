.PHONY: build run clean check lint lint-fix

# Build the application
build:
	go build -o bin/git-monitor ./cmd/git-monitor

# Clean build artifacts
clean:
	rm -rf bin/

# Install linting tools
lint-setup:
	@command -v golangci-lint >/dev/null 2>&1 || { \
		echo "Installing golangci-lint..."; \
		go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest; \
	}

# Run linters
lint: lint-setup
	golangci-lint run -v ./...

# Fix issues automatically where possible
lint-fix: lint-setup
	golangci-lint run -v --fix ./...

# Copy example config if no config exists
config:
	@if [ ! -f config.toml ]; then \
		cp config.toml.example config.toml; \
		echo "Created config.toml from example. Please edit it with your configuration."; \
		echo "Don't forget to set the GITHUB_TOKEN environment variable!"; \
		exit 1; \
	fi

# Check if GitHub token is set
check-env:
	@if [ -z "$$GITHUB_TOKEN" ]; then \
		echo "GITHUB_TOKEN environment variable is not set"; \
		echo "Please set it: export GITHUB_TOKEN=your-github-token"; \
		exit 1; \
	fi

# Run the monitoring checks
check: build config check-env
	./bin/git-monitor --config config.toml

# Run with a specific config file
run: build check-env
	./bin/git-monitor --config $(CONFIG) 