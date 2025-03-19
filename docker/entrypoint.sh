#!/bin/sh

echo "Git Repository Monitoring Tool"
echo "============================="
echo ""

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