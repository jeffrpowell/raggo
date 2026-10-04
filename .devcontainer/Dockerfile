FROM mcr.microsoft.com/devcontainers/go:1-1.27-trixie

RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
        jq \
        curl \
        wget \
        git \
        make \
        build-essential \
        pkg-config \
        libmupdf-dev \
        mupdf-tools \
    && apt-get clean \
    && rm -rf /var/lib/apt/lists/*

RUN go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest \
    && go install golang.org/x/tools/gopls@latest \
    && go install github.com/go-delve/delve/cmd/dlv@latest

USER vscode
