# Contributing to SafeUpgrade Agent

Thank you for your interest in contributing to SafeUpgrade! 🎉

## Code of Conduct

Be respectful, inclusive, and collaborative. We're all here to make dependency upgrades safer.

## How to Contribute

### Reporting Bugs

1. Check if the bug is already reported in [Issues](https://github.com/aivar-tech/safeupgrade-agent/issues)
2. If not, create a new issue with:
   - Clear title and description
   - Steps to reproduce
   - Expected vs actual behavior
   - Your environment (OS, Go version, ecosystem)
   - Relevant logs

### Suggesting Features

1. Open an issue with the `enhancement` label
2. Describe the use case and proposed solution
3. Be open to discussion and feedback

### Contributing Code

1. **Fork the repository**
   ```bash
   git clone https://github.com/aivar-tech/safeupgrade-agent
   cd safeupgrade-agent
   git remote add upstream https://github.com/aivar-tech/safeupgrade-agent
   ```

2. **Create a branch**
   ```bash
   git checkout -b feature/your-feature-name
   ```

3. **Make your changes**
   - Write tests for new functionality
   - Update documentation
   - Follow Go best practices
   - Run linters: `make lint`

4. **Test your changes**
   ```bash
   make test
   make test-integration
   ```

5. **Commit your changes**
   ```bash
   git add .
   git commit -m "feat: add support for Rust ecosystem"
   ```

   Follow [Conventional Commits](https://www.conventionalcommits.org/):
   - `feat:` - New feature
   - `fix:` - Bug fix
   - `docs:` - Documentation only
   - `refactor:` - Code refactoring
   - `test:` - Adding tests
   - `chore:` - Maintenance tasks

6. **Push and create PR**
   ```bash
   git push origin feature/your-feature-name
   ```

   Then open a Pull Request on GitHub with:
   - Clear description of changes
   - Link to related issues
   - Screenshots (if UI changes)

## Development Setup

### Prerequisites
- Go 1.22+
- Docker
- Make
- golangci-lint

### Setup
```bash
# Install dependencies
make install

# Run tests
make test

# Build
make build

# Run on sample project
make run-sample
```

### Project Structure
```
safeupgrade-agent/
├── cmd/              # CLI commands
├── internal/
│   ├── analyzer/     # AI analysis logic
│   ├── scanner/      # Dependency scanning
│   ├── policy/       # Policy evaluation
│   ├── executor/     # Upgrade execution
│   ├── reporter/     # Report generation
│   └── tools/        # CVE, changelog, provenance
├── configs/          # Default configs
├── docs/             # Documentation
└── testdata/         # Test fixtures
```

## Testing Guidelines

### Unit Tests
```bash
# Run unit tests
go test ./...

# With coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out
```

### Integration Tests
```bash
# Run integration tests
go test -tags=integration ./...
```

### Testing AI Features

When testing AI-powered features, use mock responses:

```go
// internal/analyzer/analyzer_test.go
func TestAnalyzer_Analyze(t *testing.T) {
    // Use mock AI backend
    analyzer := NewWithMock(mockResponses)
    
    result, err := analyzer.Analyze(testDeps)
    assert.NoError(t, err)
    assert.Equal(t, "SAFE", result[0].Recommendation)
}
```

## Documentation

- Update README.md for user-facing changes
- Add inline comments for complex logic
- Update docs/ for architectural changes
- Include examples in documentation

## Code Style

- Follow standard Go conventions
- Use `gofmt` and `goimports`
- Run `golangci-lint` before committing
- Keep functions small and focused
- Write clear error messages

## Adding New Ecosystems

To add support for a new package ecosystem:

1. **Create scanner** in `internal/scanner/`
   ```go
   func (s *Scanner) scanRust() (*Report, error) {
       // Implement Cargo.toml parsing
   }
   ```

2. **Add tests** in `internal/scanner/scanner_test.go`

3. **Update detection** in `cmd/scan.go`
   ```go
   if _, err := os.Stat(repo + "/Cargo.toml"); err == nil {
       return "cargo"
   }
   ```

4. **Update documentation**

## Adding New AI Providers

To add support for a new AI provider:

1. **Update config** in `internal/config/config.go`

2. **Add provider** in `internal/analyzer/analyzer.go`
   ```go
   func (a *Analyzer) callNewProvider(system, prompt string) (string, error) {
       // Implement API call
   }
   ```

3. **Add tests** with mock responses

4. **Update documentation**

## Release Process

Maintainers will:
1. Update CHANGELOG.md
2. Create a git tag: `git tag v1.0.0`
3. Push tag: `git push origin v1.0.0`
4. GitHub Actions will build and publish Docker images
5. Create GitHub release with binaries

## Questions?

- Open a [Discussion](https://github.com/aivar-tech/safeupgrade-agent/discussions)
- Join our Discord (link in README)
- Email: opensource@aivar.app

## License

By contributing, you agree that your contributions will be licensed under the MIT License.

---

Thank you for making SafeUpgrade better! 🚀
