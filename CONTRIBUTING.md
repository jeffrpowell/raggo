# Contributing to Raggo

## Development Setup

```bash
# Install Go 1.21+
# Install Make

# Clone and build
git clone https://github.com/jeffrpowell/raggo.git
cd raggo
make install-deps
make build
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

## Adding a New Stage

1. Create new binary in `cmd/raggo-<action>-<type>/`
2. Define schemas in `pkg/schema/`
3. Add Makefile targets
4. Update README and ARCHITECTURE.md
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
