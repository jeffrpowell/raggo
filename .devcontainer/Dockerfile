FROM mcr.microsoft.com/devcontainers/go:1.27-trixie

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends jq curl wget git make build-essential pkg-config \
    && apt-get clean && rm -rf /var/lib/apt/lists/*

USER vscode
# A fresh claude-code-history volume copies this dir's ownership.
RUN mkdir -p /home/vscode/.claude
RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest \
    && go install golang.org/x/tools/gopls@latest \
    && go install github.com/go-delve/delve/cmd/dlv@latest
