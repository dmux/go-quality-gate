---
layout: page
title: Contributing
permalink: /contributing/
---

# Contributing to Go Quality Gate

We welcome contributions to Go Quality Gate! This guide will help you get started with contributing to the project.

## Table of Contents

- [Getting Started](#getting-started)
- [Development Setup](#development-setup)
- [Making Changes](#making-changes)
- [Testing](#testing)
- [Submitting Changes](#submitting-changes)
- [Code Style](#code-style)
- [Documentation](#documentation)

## Getting Started

### Prerequisites

- Go 1.24 or later
- Git
- Make (for running build tasks)

### Fork and Clone

1. Fork the repository on GitHub
2. Clone your fork locally:

```bash
git clone https://github.com/YOUR-USERNAME/go-quality-gate.git
cd go-quality-gate
```

3. Add the upstream repository:

```bash
git remote add upstream https://github.com/dmux/go-quality-gate.git
```

## Development Setup

### Install Dependencies

```bash
# Download Go module dependencies
go mod download

# Install development tools
make install-tools
```

### Build the Project

```bash
# Build the binary
make build

# Build for all platforms
make build-all

# Run tests
make test

# Run linting
make lint
```

### Available Make Targets

```bash
make help  # Show all available targets

# Common targets:
make build      # Build the binary
make test       # Run tests
make lint       # Run linting
make clean      # Clean build artifacts
make install    # Install binary locally
```

## Making Changes

### Creating a Branch

Create a descriptive branch name for your changes:

```bash
git checkout -b feature/add-new-tool
git checkout -b fix/issue-123
git checkout -b docs/update-readme
```

### Commit Guidelines

We follow conventional commit format:

```
type(scope): description

[optional body]

[optional footer]
```

**Types:**
- `feat`: New feature
- `fix`: Bug fix
- `docs`: Documentation changes
- `style`: Code style changes (formatting, etc.)
- `refactor`: Code refactoring
- `test`: Adding or updating tests
- `chore`: Maintenance tasks

**Examples:**
```bash
git commit -m "feat(tools): add support for rust analyzer"
git commit -m "fix(hooks): resolve pre-commit hook execution issue"
git commit -m "docs(readme): update installation instructions"
```

## Testing

### Running Tests

```bash
# Run all tests
make test

# Run tests with coverage
make test-coverage

# Run specific test
go test ./internal/service -v

# Run tests in watch mode
make test-watch
```

### Writing Tests

We use table-driven tests where appropriate:

```go
func TestSomeFunction(t *testing.T) {
    tests := []struct {
        name     string
        input    string
        expected string
        wantErr  bool
    }{
        {
            name:     "valid input",
            input:    "test",
            expected: "expected",
            wantErr:  false,
        },
    }
    
    for _, tt := range tests {
        t.Run(tt.name, func(t *testing.T) {
            result, err := SomeFunction(tt.input)
            if (err != nil) != tt.wantErr {
                t.Errorf("SomeFunction() error = %v, wantErr %v", err, tt.wantErr)
                return
            }
            if result != tt.expected {
                t.Errorf("SomeFunction() = %v, want %v", result, tt.expected)
            }
        })
    }
}
```

### Test Coverage

We aim for high test coverage. Check coverage with:

```bash
make test-coverage
# Opens coverage report in browser
```

## Submitting Changes

### Before Submitting

1. Ensure tests pass:
   ```bash
   make test
   ```

2. Run linting:
   ```bash
   make lint
   ```

3. Update documentation if needed

4. Add yourself to CONTRIBUTORS.md if it's your first contribution

### Creating a Pull Request

1. Push your branch:
   ```bash
   git push origin feature/your-feature-name
   ```

2. Create a pull request on GitHub with:
   - Clear title and description
   - Reference any related issues
   - Screenshots if UI changes are involved
   - Test results if applicable

### Pull Request Template

```markdown
## Description
Brief description of changes

## Type of Change
- [ ] Bug fix
- [ ] New feature
- [ ] Breaking change
- [ ] Documentation update

## Testing
- [ ] Tests pass locally
- [ ] Added tests for new functionality
- [ ] Manual testing completed

## Checklist
- [ ] Code follows project style guidelines
- [ ] Self-review completed
- [ ] Documentation updated
- [ ] No breaking changes (or documented)
```

## Code Style

### Go Code Style

We follow standard Go conventions:

```go
// Good: Clear, descriptive names
func DetectLanguages(projectPath string) ([]Language, error) {
    // Implementation
}

// Good: Proper error handling
if err != nil {
    return nil, fmt.Errorf("failed to detect languages: %w", err)
}

// Good: Table-driven tests
func TestDetectLanguages(t *testing.T) {
    tests := []struct {
        name string
        path string
        want []Language
    }{
        // Test cases
    }
}
```

### Linting

We use `golangci-lint` with strict settings:

```bash
# Run linting
make lint

# Fix auto-fixable issues
make lint-fix
```

## Documentation

### Code Documentation

- Add godoc comments to all exported functions and types
- Include examples in documentation when helpful
- Keep comments concise but informative

```go
// DetectLanguages scans the project directory and returns a slice of detected
// programming languages based on file extensions and project markers.
//
// Example:
//   languages, err := DetectLanguages("/path/to/project")
//   if err != nil {
//       log.Fatal(err)
//   }
func DetectLanguages(projectPath string) ([]Language, error) {
    // Implementation
}
```

### README and Docs

- Update README.md for user-facing changes
- Add examples for new features
- Keep documentation in sync with code changes
- Use clear, concise language

## Adding New Tools

### Tool Implementation

To add support for a new quality tool:

1. **Create tool definition:**
   ```go
   // internal/domain/tool.go
   type NewTool struct {
       Name        string
       Language    Language
       Installable bool
   }
   ```

2. **Implement tool manager:**
   ```go
   // internal/service/tool_manager.go
   func (tm *ToolManager) InstallNewTool(ctx context.Context) error {
       // Installation logic
   }
   ```

3. **Add configuration support:**
   ```go
   // internal/config/config.go
   type NewToolConfig struct {
       Enabled bool   `yaml:"enabled"`
       Args    []string `yaml:"args"`
   }
   ```

4. **Write tests:**
   ```go
   func TestNewToolInstallation(t *testing.T) {
       // Test implementation
   }
   ```

5. **Update documentation:**
   - Add tool to README.md
   - Update configuration.md
   - Add usage examples

### Tool Requirements

New tools should:
- Be cross-platform or have platform-specific implementations
- Support automated installation
- Provide structured output when possible
- Have reasonable defaults
- Be widely used in the target language ecosystem

## Release Process

### Version Bumping

We use semantic versioning (SemVer):
- `MAJOR.MINOR.PATCH`
- Breaking changes: bump MAJOR
- New features: bump MINOR  
- Bug fixes: bump PATCH

### Release Checklist

1. Update version in `cmd/quality-gate/version.go`
2. Update CHANGELOG.md
3. Create release commit
4. Tag the release
5. GitHub Actions handles the rest

## Getting Help

### Communication

- **GitHub Issues**: Bug reports and feature requests
- **GitHub Discussions**: General questions and ideas
- **Pull Requests**: Code review and discussion

### Resources

- [Go Documentation](https://golang.org/doc/)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go.html)

## Recognition

Contributors are recognized in:
- CONTRIBUTORS.md file
- GitHub contributor statistics
- Release notes for significant contributions

Thank you for contributing to Go Quality Gate! 🎉