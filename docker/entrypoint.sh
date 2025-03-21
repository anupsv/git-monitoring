#!/bin/sh

echo "Git Repository Monitoring Tool"
echo "============================="
echo ""

# Check if we need to download config from S3
if [ ! -z "$S3_CONFIG_BUCKET" ] && [ ! -z "$S3_CONFIG_KEY" ]; then
  echo "[CONFIG] Downloading config from S3 bucket: $S3_CONFIG_BUCKET, key: $S3_CONFIG_KEY"
  mkdir -p $(dirname $CONFIG_PATH)
  if aws s3 cp s3://$S3_CONFIG_BUCKET/$S3_CONFIG_KEY $CONFIG_PATH; then
    echo "[CONFIG] Successfully downloaded config from S3"
  else
    echo "[CONFIG] Failed to download config from S3. Will use local config if available."
  fi
fi

# Use local config as fallback
if [ ! -f "$CONFIG_PATH" ]; then
  echo "[CONFIG] No config file found at $CONFIG_PATH"
  echo "[CONFIG] Using default example config from /app/config.toml.example"
  cp /app/config.toml.example $CONFIG_PATH
  echo "[CONFIG] Created default config at $CONFIG_PATH"
else
  echo "[CONFIG] Using config file: $CONFIG_PATH"
fi

echo ""

if [ -z "$GITHUB_TOKEN" ]; then
  echo "[WARNING] GITHUB_TOKEN environment variable is not set"
  echo "[WARNING] The application may not function properly without a valid GitHub token"
  echo ""
else
  echo "[AUTH] GitHub token is set"
fi

echo "[EXEC] Running: git-monitor --config \"$CONFIG_PATH\" $@"
echo "============================="
echo ""

git-monitor --config "$CONFIG_PATH" "$@" 