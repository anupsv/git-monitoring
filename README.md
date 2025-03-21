# Git Repository Monitoring

A Go application to monitor Git repositories and ensure they follow best practices and policies.

## Features

- **PR Approval Checker**: Verifies that all pull requests merged within the configured time window have been approved before merge
- **Repository Visibility Checker**: Monitors for repositories that were recently made public
- **GitHub API Rate Limiting**: Respects GitHub API rate limits to prevent throttling issues

## Installation

```bash
# Clone the repository
git clone https://github.com/anupsv/git-monitoring.git
cd git-monitoring

# Build the application
make build

# Create your configuration file
cp config.toml.example config.toml
```

Edit the `config.toml` file with your repositories to monitor.

## Configuration

The application is configured using a TOML file and environment variables.

### Environment Variables

- `GITHUB_TOKEN` - GitHub API token for authentication (required)

### Config File

```toml
# GitHub API configuration
[github]
# Token will be read from GITHUB_TOKEN environment variable
token = ""

# Monitor configurations
[monitors]
  # PR Checker configuration
  [monitors.pr_checker]
  enabled = true
  # Repository visibility filter options:
  # - "specific": Only check repositories listed below (default)
  # - "all": Check all repos (public and private) the token has access to
  # - "public-only": Only check public repositories
  # - "private-only": Only check private repositories
  repo_visibility = "specific"
  # GitHub organization to check (optional)
  # Only used when repo_visibility is not "specific"
  # If not specified, repositories of the authenticated user will be checked
  organization = ""
  # List of repositories to check (only used when repo_visibility = "specific")
  specific_repositories = [
    "owner1/repo1",
    "owner2/repo2"
  ]
  # List of repositories to exclude (only used with "all", "public-only", or "private-only")
  excluded_repositories = [
    "owner1/exclude-repo1",
    "owner2/exclude-repo2"
  ]
  time_window_hours = 24  # Default is 24 hours
  # Enable verbose logging for PR approval debugging
  debug_logging = false
  
  # Repository Visibility Monitor Configuration
  [monitors.repo_visibility]
  enabled = false # Set to true to enable the repository visibility monitor
  # Repository visibility filter options:
  # - "specific": Only check repositories of organizations listed below (default)
  # - "all": Check all repos (public and private) in the organizations
  # - "public-only": Only check public repositories in the organizations
  # - "private-only": Only check private repositories in the organizations
  repo_visibility = "specific"
  # List of GitHub organizations to monitor for repository visibility changes
  # Used with all visibility options
  organizations = [
    "example-org1",
    "example-org2"
  ]
  # How many hours back to look for visibility changes
  check_window_hours = 24
```

## Usage

```bash
# Set GitHub token
export GITHUB_TOKEN="your-github-token"

# Run with default config file (config.toml)
make check

# Run with a specific config file
make run CONFIG=path/to/config.toml

# Or run directly
./bin/git-monitor --config path/to/config.toml
```

## Development

### Testing

The project includes comprehensive tests for all key components. To run the tests:

```bash
# Run all tests
make test

# Run tests with verbose output
make test-verbose

# Run tests with coverage report
make test-coverage

# Generate HTML coverage report
make test-coverage-html
```

### Linting

The project uses golangci-lint for code quality. To run the linter:

```bash
# Run the linter
make lint

# Auto-fix issues where possible
make lint-fix
```

### Docker

You can use the official Docker image from GitHub Container Registry:

```bash
# Pull the latest image
docker pull ghcr.io/anupsv/git-monitoring:latest

# Run the container
docker run -e GITHUB_TOKEN="your-github-token" -v $(pwd)/config.toml:/app/config.toml ghcr.io/anupsv/git-monitoring:latest
```

Or build it locally:

```bash
# Build the Docker image
docker build -t git-monitor -f docker/Dockerfile .

# Run the container
docker run --rm \
  -e GITHUB_TOKEN=your_github_token \
  git-monitor
```

### CI/CD

This project includes a GitHub Actions workflow that:
- Runs linting checks
- Runs tests
- Builds and publishes Docker images to GitHub Container Registry

The workflow is automatically triggered on pushes to main and pull requests.

## Adding New Tools

1. Create a new package under `pkg/tools/`
2. Add configuration in `pkg/config/config.go`
3. Implement your tool's functionality
4. Add the tool to the main program in `cmd/git-monitor/main.go`

## Docker Usage

### Running with Docker

The application is available as a Docker image and can be run in various configurations:

```bash
# Pull the official image
docker pull ghcr.io/anupsv/git-monitoring:latest

# Basic usage with required GitHub token
docker run --rm -e GITHUB_TOKEN=your_github_token ghcr.io/anupsv/git-monitoring
```

### Configuration Options

#### Environment Variables

| Variable | Description | Required | Default |
|----------|-------------|----------|---------|
| `GITHUB_TOKEN` | GitHub API token for authentication | Yes | - |
| `CONFIG_PATH` | Path to the configuration file inside the container | No | `/config/config.toml` |

#### Volumes

The Docker image is configured with a volume at `/config` for persistent configuration storage.

### Using Custom Configuration

There are several ways to provide your configuration to the Docker container:

#### Option 1: Mount a Config File to the Default Location

This is the simplest approach - mount your local config file to the default location:

```bash
# Create a config file on your host machine
cp config.toml.example my-config.toml
# Edit my-config.toml with your preferred settings

# Run with your custom config
docker run --rm \
  -v $(pwd)/my-config.toml:/config/config.toml \
  -e GITHUB_TOKEN=your_github_token \
  ghcr.io/anupsv/git-monitoring
```

#### Option 2: Mount a Config File to a Custom Location

If you prefer to use a different location for your config file:

```bash
# Mount your config file to a custom location and specify the path
docker run --rm \
  -v $(pwd)/my-config.toml:/app/custom-location/config.toml \
  -e CONFIG_PATH=/app/custom-location/config.toml \
  -e GITHUB_TOKEN=your_github_token \
  ghcr.io/anupsv/git-monitoring
```

#### Option 3: Mount a Config Directory

If you manage multiple configurations:

```bash
# Create a directory with different config files
mkdir -p configs
cp config.toml.example configs/prod.toml
cp config.toml.example configs/dev.toml
# Edit the config files as needed

# Mount the directory and specify which config to use
docker run --rm \
  -v $(pwd)/configs:/config \
  -e CONFIG_PATH=/config/prod.toml \
  -e GITHUB_TOKEN=your_github_token \
  ghcr.io/anupsv/git-monitoring
```

### Running in CI/CD Pipelines

Example GitHub Actions workflow step:

```yaml
- name: Check repository compliance
  uses: docker://ghcr.io/anupsv/git-monitoring:latest
  env:
    GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}
  with:
    args: ""  # Additional arguments if needed
```

### Building the Docker Image Locally

```bash
# Clone the repository
git clone https://github.com/anupsv/git-monitoring.git
cd git-monitoring

# Build the Docker image
docker build -t git-monitor -f docker/Dockerfile .

# Run the container
docker run --rm \
  -e GITHUB_TOKEN=your_github_token \
  git-monitor
```

## License

MIT 

## GitHub API Rate Limiting

This application implements rate limiting for all GitHub API calls to respect GitHub's API usage limits:

- Maintains a conservative rate limit of 1.25 requests per second (4500/hour)
- Automatically waits when approaching rate limits
- Logs warnings when rate limits are getting low
- Properly spaces API requests to avoid hitting rate limits

This ensures the application can be run safely without hitting GitHub's API rate limits, even when monitoring many repositories. 