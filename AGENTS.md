# AGENTS.md - Vibe Configuration

This file contains configuration and instructions for Mistral Vibe when working with this repository.

## Repository Context

- **Project**: tviewmd - Native markdown rendering for tview TextViews
- **Language**: Go (Golang)
- **Primary Use Case**: Rendering markdown in terminal applications
- **Key Dependencies**: goldmark, chroma, tview

## Tooling

### Linter
- **Primary**: golangci-lint (v1.61.0)
- **Config**: `.golangci.yml`
- **Command**: `make lint` or `golangci-lint run --timeout 5m`

### Testing
- **Test Framework**: Go's built-in `testing` package
- **Fuzzing**: Built-in Go fuzzing for `FuzzParse`
- **Race Detector**: Always run with `-race` flag
- **Coverage**: Use `go test -coverprofile=coverage.out -covermode=atomic`

### Build
- **Go Version**: 1.24+ (1.25 recommended)
- **Module Path**: `github.com/buchenberg/tviewmd`
- **Build Command**: `go build -v ./...`

## Commands

### Development Workflow

```bash
# Install dependencies
go mod download

# Run all checks
make all

# Run tests with race detector
make test

# Run linting
golangci-lint run --timeout 5m

# Build the project
make build

# Generate coverage report
make coverage

# Run fuzz tests
make fuzz
```

### Git Workflow

```bash
# Create feature branch
git checkout -b feature/name

# Add changes
git add .

# Commit with conventional commit message
git commit -m "feat: add new feature"

# Push to origin
git push origin feature/name

# Create PR
# (Use GitHub UI or gh CLI)
```

## Code Style

### Go Conventions
- Follow [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- Use `gofmt` for formatting
- Prefer `strings.Builder` for string concatenation in loops
- Use `context.Context` for cancellable operations
- Follow [Effective Go](https://go.dev/doc/effective_go)

### Project-Specific
- Match existing code style and patterns
- Add tests for all new functionality
- Document breaking changes in CHANGELOG
- Use ADRs for significant architectural decisions (see `docs/adr/`)

## Testing Requirements

### Unit Tests
- All public functions should have unit tests
- Tests should cover happy path and error cases
- Use table-driven tests where appropriate
- Aim for >80% code coverage

### Fuzz Testing
- All parsing functions should have fuzz tests
- Fuzz tests must run without panicking on arbitrary input
- Minimum fuzz time: 30 seconds per target

### Integration Tests
- Test interactions between components
- Verify correct markdown rendering output
- Test with various markdown features (tables, code blocks, etc.)

## CI/CD

### GitHub Actions
- Workflow file: `.github/workflows/ci.yml`
- Triggers: Push/PR to main, v1.0 branches
- Jobs: test, lint, build, coverage, release

### Release Process
1. Create PR with changes
2. Merge to main
3. Create annotated tag: `git tag -a v1.0.0 -m "Release v1.0.0"`
4. Push tag: `git push origin v1.0.0`
5. GitHub Actions automatically creates release

## Security

### Dependency Scanning
- Regularly run `go mod tidy`
- Check for vulnerabilities with `govulncheck`
- Keep dependencies updated

### Secret Scanning
- Never commit API keys, tokens, or credentials
- Use GitHub secrets for CI/CD tokens
- Add sensitive files to `.gitignore`

## Documentation

### Required Documentation
- Package-level godoc comments
- Function-level godoc comments
- README.md with usage examples
- ADRs for architectural decisions (in `docs/adr/`)

### Godoc Examples
```go
// Example usage in godoc
//
//   tv := tview.NewTextView().SetWrap(true)
//   fmt.Fprint(tv, tviewmd.Render("# Hello", tviewmd.Options{}))
func Render(src string, opts Options) string {
    // ...
}
```

## Vibe-Specific Instructions

### Skills to Load
- `golang-how-to` - For general Go development
- `golang-testing` - For test patterns
- `golang-lint` - For linting configuration
- `github-ops` - For repository management

### When to Ask for Confirmation
- Before pushing to main or v1.0 branches
- Before force-pushing or resetting branches
- Before deleting files or branches
- Before modifying CI/CD configuration

### Automatic Approvals
- Routine test updates
- Dependency version bumps (within same major version)
- Documentation improvements
- Linting fixes

## File Structure

```
tviewmd/
├── .github/
│   └── workflows/
│       └── ci.yml              # GitHub Actions CI
├── .gitignore                 # Git ignore patterns
├── .golangci.yml              # Linter configuration
├── AGENTS.md                  # This file
├── Makefile                   # Development targets
├── go.mod                     # Go module definition
├── go.sum                     # Dependency checksums
├── README.md                  # Project documentation
├── docs/
│   ├── README.md              # Documentation index
│   └── adr/                   # Architecture Decision Records
│       ├── README.md          # ADR process
│       └── 0001-*.md           # Individual ADRs
└── *.go                      # Source files
```

## Common Tasks

### Adding a New Feature
1. Create branch: `git checkout -b feat/feature-name`
2. Add code with tests
3. Run `make all` to verify
4. Document in README if user-facing
5. Add ADR if architectural decision
6. Push and create PR

### Fixing a Bug
1. Create branch: `git checkout -b fix/bug-name`
2. Add regression test
3. Fix the bug
4. Run `make all` to verify
5. Push and create PR

### Updating Dependencies
1. Run `go get package@version`
2. Run `go mod tidy`
3. Verify tests pass
4. Commit dependency updates

## Versioning

This project follows semantic versioning:
- **MAJOR**: Breaking changes
- **MINOR**: New features (backward compatible)
- **PATCH**: Bug fixes (backward compatible)

### Pre-v1.0
- Follow same principles but version is `v0.x.y`
- Breaking changes increment MINOR version
- v1.0.0 will be first stable release

## Contacts

- **Repository**: https://github.com/buchenberg/tviewmd
- **Issue Tracker**: GitHub Issues
- **Discussions**: GitHub Discussions
- **Maintainer**: @buchenberg

## Last Updated

2026-08-07