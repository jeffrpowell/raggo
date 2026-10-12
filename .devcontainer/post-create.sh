#!/usr/bin/env bash
set -euo pipefail

# Trust the bind-mounted corporate root CA (Go and curl use the system store;
# Claude Code reads NODE_EXTRA_CA_CERTS). Must run before any downloads.
sudo update-ca-certificates 2>/dev/null || true

# The claude-code-history volume may have been created root-owned by another
# devcontainer; Claude Code runs as vscode here.
sudo chown -R vscode:vscode /home/vscode/.claude

go mod download
./scripts/build.sh bin
