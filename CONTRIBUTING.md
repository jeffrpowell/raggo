# Contributing to Raggo

## Development Setup

### Option 1: Devcontainer (Recommended)

Raggo includes a complete VS Code devcontainer configuration for a consistent development environment.

**Prerequisites:**
- VS Code with Dev Containers extension
- Docker Desktop or Docker Engine

**Quick Start:**

1. Open project in VS Code:
   ```bash
   code /path/to/raggo
   ```

2. Reopen in container:
   - Click notification: "Reopen in Container"
   - Or: Command Palette → "Dev Containers: Reopen in Container"

3. Wait for initial build (2-5 minutes first time)

4. Verify setup:
   ```bash
   go version
   make --version
   ls bin/
   curl http://localhost:6333/collections
   ```

**What's Included:**
- Go 1.21 with all standard tools (gopls, golangci-lint, delve)
- Qdrant vector database (automatically started on ports 6333/6334)
- Pre-configured LSP and linting
- All dependencies installed and binaries built via post-create commands

**Data Persistence:**
- Code and artifacts: mounted at `/workspaces/raggo` (bidirectional sync)
- Qdrant data: Docker volume `qdrant-storage` (persists across rebuilds)

**Troubleshooting:**

```bash
# Container won't start
docker ps
docker logs <container-id>

# Qdrant not accessible
curl http://localhost:6333/collections
docker ps | grep qdrant

# Go LSP not working
go install golang.org/x/tools/gopls@latest
# Command Palette → "Go: Restart Language Server"

# Rebuild container after modifying .devcontainer/
# Command Palette → "Dev Containers: Rebuild Container"
```

**Customization:**

Edit `.devcontainer/devcontainer.json` to add VS Code extensions:
```json
"customizations": {
  "vscode": {
    "extensions": ["golang.go", "your.extension.id"]
  }
}
```

Edit `.devcontainer/Dockerfile` to change Go version or add system packages:
```dockerfile
FROM mcr.microsoft.com/devcontainers/go:1-1.22-bullseye
RUN apt-get update && apt-get -y install --no-install-recommends your-package
```

### Option 2: Local Development

**Prerequisites:**
- Go 1.21+
- GNU Make
- jq (for JSON processing)
- Qdrant instance
- Optional: STT service, Embedding service

**Setup:**

```bash
git clone https://github.com/jeffrpowell/raggo.git
cd raggo
make install-deps
make build

# Start Qdrant
docker-compose up -d
```

## Code Style

- Follow standard Go conventions (`gofmt`, `golint`)
- Write tests for shared libraries
- Document exported functions
- Use structured logging, not `fmt.Println`

## Testing

### Unit Tests

```bash
go test ./pkg/...
```

### Integration Tests

```bash
# Test individual binaries
bin/raggo-rss-podcast -feed-url="file://test/fixtures/feed.xml" -manifest-dir="test/output"
```

### End-to-End Tests

```bash
# Use test feed with fixtures
make podcast FEED_URL="file://test/fixtures/small-feed.xml"
test -f data/podcast/index/*.done
```

## Adding a New Stage

1. Create new binary in `cmd/raggo-<action>-<type>/`
2. Define schemas in `pkg/schema/`
3. Add Makefile targets in appropriate pipeline
4. Update documentation (README.md, ARCHITECTURE.md, EXAMPLES.md)
5. Submit PR with test coverage

## Design Principles

- **Files over abstractions**: Persist everything to disk
- **Explicit over implicit**: No hidden magic
- **Deterministic**: Same input → same output
- **Fail-fast**: Loud errors, no silent failures

## Pull Request Process

1. Fork the repository
2. Create a feature branch
3. Make changes following code style
4. Add tests
5. Update documentation
6. Submit PR with clear description
