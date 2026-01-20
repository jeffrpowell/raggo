# Devcontainer Development Guide

## Overview

Raggo includes a complete VS Code devcontainer configuration for a consistent development environment. The devcontainer provides:

- **Go 1.21** with all standard tools
- **Qdrant** vector database (automatically started)
- **Pre-configured LSP** and linting
- **All dependencies** installed and binaries built

## Quick Start

### Prerequisites

- VS Code with **Dev Containers** extension
- Docker Desktop or Docker Engine

### Opening in Devcontainer

1. **Open project in VS Code:**
   ```bash
   code /home/jeffpowell/dev/jeffrpowell/raggo
   ```

2. **Reopen in container:**
   - Click notification: "Reopen in Container"
   - Or: Command Palette → "Dev Containers: Reopen in Container"

3. **Wait for build:**
   - First time takes 2-5 minutes
   - Subsequent starts are near-instant

4. **Verify setup:**
   ```bash
   # Inside container terminal
   go version
   make --version
   ls bin/
   curl http://localhost:6333/collections
   ```

## What's Included

### Container Image

- **Base**: Microsoft's official Go 1.21 devcontainer
- **Tools**: jq, curl, wget, git, make, build-essential
- **Go Tools**: golangci-lint, gopls, delve

### Services

- **Qdrant**: Runs alongside the devcontainer
  - HTTP API: `localhost:6333`
  - gRPC API: `localhost:6334`
  - Data persists in Docker volume

### VS Code Extensions

Pre-installed:
- `golang.go` - Official Go extension
- `ms-azuretools.vscode-docker` - Docker management
- `eamodio.gitlens` - Git visualization
- `tamasfe.even-better-toml` - TOML support

### VS Code Settings

- Auto-formatting on save
- Organize imports on save
- golangci-lint integration
- Go LSP enabled

## Container Architecture

```
┌─────────────────────────────────┐
│   VS Code Dev Container         │
│   ┌─────────────────────────┐   │
│   │  Go Environment         │   │
│   │  - gopls (LSP)          │   │
│   │  - golangci-lint        │   │
│   │  - dlv (debugger)       │   │
│   └─────────────────────────┘   │
│                                 │
│   Mounted: /workspaces/raggo    │
│   Data: /workspaces/raggo/data  │
└──────────┬──────────────────────┘
           │ Network: shared with Qdrant
           │
┌──────────▼───────────────────────┐
│   Qdrant Container               │
│   - HTTP: 6333                   │
│   - gRPC: 6334                   │
│   - Volume: qdrant-storage       │
└──────────────────────────────────┘
```

## Post-Create Commands

The devcontainer automatically runs on first start:

```bash
make install-deps  # Downloads Go modules
make build         # Compiles all binaries
```

## Port Forwarding

Automatically forwarded to your local machine:
- **6333**: Qdrant HTTP API
- **6334**: Qdrant gRPC API

Access from your local browser:
```
http://localhost:6333/dashboard
```

## Data Persistence

### Code and Artifacts

The entire project is mounted at `/workspaces/raggo`:
- Changes in container → reflected on host
- Changes on host → reflected in container

### Qdrant Data

Stored in Docker volume `qdrant-storage`:
- Persists across container rebuilds
- Survives container deletion
- To reset: `docker volume rm raggo_qdrant-storage`

## Common Workflows

### Running the Pipeline

```bash
# Inside devcontainer terminal
make podcast FEED_URL="https://example.com/podcast.rss"
```

### Debugging

Use VS Code's built-in debugger:

1. Set breakpoints in Go files
2. F5 or "Run and Debug" panel
3. Debug configuration auto-configured by Go extension

### Installing Additional Tools

```bash
# Inside container
go install github.com/some/tool@latest
```

### Rebuilding Container

If you modify `.devcontainer/`:

1. Command Palette
2. "Dev Containers: Rebuild Container"
3. Wait for rebuild

## Troubleshooting

### Container Won't Start

```bash
# Check Docker is running
docker ps

# View container logs
docker logs <container-id>

# Reset everything
docker-compose -f .devcontainer/docker-compose.yml down -v
```

### Qdrant Not Accessible

```bash
# Inside container
curl http://localhost:6333/collections

# If fails, check Qdrant container
docker ps | grep qdrant
docker logs <qdrant-container-id>
```

### Go LSP Not Working

```bash
# Reinstall gopls
go install golang.org/x/tools/gopls@latest

# Restart Go language server
# Command Palette → "Go: Restart Language Server"
```

### Slow Performance

- Ensure Docker has sufficient resources (4GB+ RAM)
- Check Docker Desktop settings
- Use volume mounts for data directories (already configured)

## Customization

### Adding Extensions

Edit `.devcontainer/devcontainer.json`:

```json
"customizations": {
  "vscode": {
    "extensions": [
      "golang.go",
      "your.extension.id"
    ]
  }
}
```

### Changing Go Version

Edit `.devcontainer/Dockerfile`:

```dockerfile
FROM mcr.microsoft.com/devcontainers/go:1-1.22-bullseye
```

### Adding System Packages

Edit `.devcontainer/Dockerfile`:

```dockerfile
RUN apt-get update && export DEBIAN_FRONTEND=noninteractive \
    && apt-get -y install --no-install-recommends \
        your-package-here
```

## Alternative: Docker Compose Only

If not using VS Code:

```bash
# Start services
docker-compose -f .devcontainer/docker-compose.yml up -d

# Exec into container
docker exec -it <container-name> bash

# Work normally
make build
make podcast FEED_URL="..."
```

## Benefits Over Local Development

| Feature | Local | Devcontainer |
|---------|-------|--------------|
| **Setup time** | 15-30 min | 5 min (first time) |
| **Go version** | Manual install | Automatic |
| **Qdrant** | Separate setup | Auto-started |
| **Consistency** | Varies by machine | Identical everywhere |
| **Isolation** | Affects host | Contained |
| **Onboarding** | Multi-step docs | One click |

## Production Considerations

The devcontainer is **development-only**:

- ❌ Do NOT deploy the devcontainer to production
- ✅ DO use the compiled binaries (`bin/`)
- ✅ DO use the Makefiles in CI/CD
- ✅ DO run Qdrant separately in production

## Further Reading

- [VS Code Dev Containers](https://code.visualstudio.com/docs/devcontainers/containers)
- [Go Devcontainer Features](https://github.com/devcontainers/features/tree/main/src/go)
- [Docker Compose](https://docs.docker.com/compose/)
