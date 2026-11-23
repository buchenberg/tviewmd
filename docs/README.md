# tviewmd Documentation

This directory contains documentation for the tviewmd project.

## Structure

```
docs/
├── README.md          # This file
└── adr/              # Architecture Decision Records
    ├── README.md      # ADR index and guidelines
    └── 0001-*.md      # Individual ADRs (sequentially numbered)
```

## Documentation Index

### Architecture Decisions

Architecture Decision Records (ADRs) capture significant architectural choices made during the project's development.

| ADR | Title | Status |
|-----|-------|--------|
| [ADR-0001](adr/0001-streaming-support.md) | Streaming Markdown Rendering for LLM Responses | Accepted |

See [adr/README.md](adr/README.md) for more information about the ADR process.

### API Documentation

The primary API documentation is in the Go package documentation:

- [Package overview](https://pkg.go.dev/github.com/buchenberg/tviewmd) (when published)
- Inline code comments in `doc.go`, `parse.go`, `render.go`, etc.

### Examples

See the main [README.md](../README.md) for usage examples.

### Contribution Guidelines

- Follow existing code style and patterns
- Add tests for new functionality
- Document breaking changes in [CHANGELOG.md](../CHANGELOG.md)
- Use ADRs for significant architectural decisions

## Building Documentation

```bash
# Generate Go documentation
godoc -url "http://localhost:6060" .

# Or use pkg.go.dev for published modules
```

## Version History

The repository keeps a [Keep a Changelog](https://keepachangelog.com/en/1.1.0/)
style [CHANGELOG.md](../CHANGELOG.md) at its root, covering released and
unreleased changes. Git tags carry the release history as well.