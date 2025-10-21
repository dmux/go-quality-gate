---
layout: page
title: Usage
permalink: /usage/
---

# Usage Guide

This guide covers the basic and advanced usage of Go Quality Gate in your development workflow.

## Basic Usage

### Initialize in Your Project

Navigate to your project directory and run:

```bash
quality-gate init
```

This command will:
- Create a `quality.yml` configuration file
- Set up Git hooks automatically
- Detect your project's language and configure appropriate tools

### Manual Execution

You can run quality checks manually at any time:

```bash
# Run all configured tools
quality-gate run

# Run with verbose output
quality-gate run --verbose

# Run with JSON output (for CI/CD)
quality-gate run --json
```

## Git Hooks Integration

Go Quality Gate automatically integrates with Git hooks to run quality checks:

### Pre-commit Hook
Runs automatically before each commit:
- Code formatting
- Linting
- Basic security checks
- Import organization

### Pre-push Hook
Runs before pushing to remote repository:
- Comprehensive linting
- Security scanning
- Test validation (if configured)

## Command Line Options

### Global Options

```bash
quality-gate [global options] command [command options]
```

**Global Options:**
- `--config, -c`: Specify config file path (default: `quality.yml`)
- `--verbose, -v`: Enable verbose output
- `--quiet, -q`: Suppress output except errors
- `--json`: Output results in JSON format
- `--help, -h`: Show help
- `--version`: Show version information

### Available Commands

#### `init`
Initialize Go Quality Gate in your project:

```bash
quality-gate init [options]

Options:
  --force, -f    Overwrite existing configuration
  --hooks        Install Git hooks only
  --config-only  Create configuration file only
```

#### `run`
Execute quality checks:

```bash
quality-gate run [options] [tools...]

Options:
  --all, -a      Run all configured tools (default)
  --staged       Run only on staged files (git)
  --fix          Automatically fix issues when possible
  --dry-run      Show what would be executed without running

Examples:
  quality-gate run                    # Run all tools
  quality-gate run golangci-lint      # Run specific tool
  quality-gate run --staged          # Run on staged files only
  quality-gate run --fix             # Auto-fix issues
```

#### `install`
Install quality tools:

```bash
quality-gate install [tools...]

Examples:
  quality-gate install               # Install all detected tools
  quality-gate install golangci-lint # Install specific tool
```

#### `status`
Show project and tools status:

```bash
quality-gate status

# Shows:
# - Detected languages
# - Configured tools
# - Tool installation status
# - Git hooks status
```

## Workflow Examples

### Daily Development Workflow

1. **Start working on a feature:**
   ```bash
   git checkout -b feature/new-feature
   ```

2. **Make changes and commit:**
   ```bash
   git add .
   git commit -m "Add new feature"
   # Quality Gate runs automatically via pre-commit hook
   ```

3. **Push changes:**
   ```bash
   git push origin feature/new-feature
   # Quality Gate runs comprehensive checks via pre-push hook
   ```

### CI/CD Integration

Use Go Quality Gate in your CI/CD pipeline:

```bash
# In your CI script
quality-gate run --json > quality-report.json

# Check exit code
if [ $? -ne 0 ]; then
  echo "Quality checks failed"
  exit 1
fi
```

**GitHub Actions Example:**

```yaml
name: Quality Gate
on: [push, pull_request]

jobs:
  quality:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v3
      
      - name: Download Quality Gate
        run: |
          wget https://github.com/dmux/go-quality-gate/releases/latest/download/quality-gate-linux-amd64
          chmod +x quality-gate-linux-amd64
          sudo mv quality-gate-linux-amd64 /usr/local/bin/quality-gate
          
      - name: Run Quality Checks
        run: quality-gate run --json
```

### Language-Specific Examples

#### Go Project
```bash
# Initialize for Go project
quality-gate init

# This automatically detects and configures:
# - gofmt, goimports (formatting)
# - golangci-lint (linting)
# - go vet (static analysis)
# - gosec (security)
```

#### Python Project
```bash
# Initialize for Python project
quality-gate init

# This automatically detects and configures:
# - black (formatting)
# - isort (import sorting)
# - flake8 (linting)
# - mypy (type checking)
# - bandit (security)
```

#### Multi-language Project
```bash
# Works seamlessly with projects containing multiple languages
quality-gate init

# Detects all languages and configures appropriate tools
# Example: Go + Python + Node.js project gets tools for all three
```

## Output Formats

### Standard Output
Human-readable output with colors and progress indicators:

```
🔍 Running quality checks...
✅ gofmt: All files formatted correctly
⚠️  golangci-lint: 2 issues found
❌ gosec: 1 security issue detected

Quality gate: FAILED (1 critical, 2 warnings)
```

### JSON Output
Machine-readable output for automation:

```json
{
  "status": "failed",
  "summary": {
    "total_tools": 3,
    "passed": 1,
    "failed": 2,
    "execution_time": "2.34s"
  },
  "results": [
    {
      "tool": "gofmt",
      "status": "passed",
      "execution_time": "0.12s"
    }
  ]
}
```

### Verbose Output
Detailed information about tool execution:

```
[INFO] Detected languages: go, python
[INFO] Loading configuration from quality.yml
[INFO] Running tool: gofmt
[DEBUG] Command: gofmt -l .
[DEBUG] Exit code: 0
[INFO] gofmt: ✅ PASSED
```

## Next Steps

- [Learn about configuration options](configuration.html)
- [Customize tool behavior](configuration.html#tool-configuration)
- [Set up advanced workflows](configuration.html#workflow-configuration)