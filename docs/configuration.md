---
layout: page
title: Configuration
permalink: /configuration/
---

# Configuration Reference

This guide covers all available configuration options for Go Quality Gate.

## Configuration File

Go Quality Gate uses a `quality.yml` file for configuration. The file is automatically generated when you run `quality-gate init`, but you can customize it to fit your project's needs.

### Basic Structure

```yaml
# quality.yml
version: "1.0"

# Global settings
settings:
  parallel: true
  fail_fast: false
  timeout: 300

# Language detection
languages:
  auto_detect: true
  include:
    - go
    - python
    - javascript
  exclude:
    - java

# Tool configuration
tools:
  gofmt:
    enabled: true
    args: ["-l", "."]
  
  golangci-lint:
    enabled: true
    config: ".golangci.yml"
    
  black:
    enabled: true
    line_length: 88
```

## Global Settings

### Core Settings

| Setting | Type | Default | Description |
|---------|------|---------|-------------|
| `parallel` | boolean | `true` | Run tools in parallel when possible |
| `fail_fast` | boolean | `false` | Stop execution on first failure |
| `timeout` | integer | `300` | Global timeout in seconds |
| `verbose` | boolean | `false` | Enable verbose logging |

### File Filtering

```yaml
settings:
  # Include patterns (glob format)
  include_patterns:
    - "**/*.go"
    - "**/*.py"
    - "**/*.js"
  
  # Exclude patterns (glob format)  
  exclude_patterns:
    - "**/vendor/**"
    - "**/node_modules/**"
    - "**/.git/**"
    - "**/.*"
```

## Language Configuration

### Auto Detection

```yaml
languages:
  auto_detect: true  # Automatically detect languages
  
  # Override detection for specific languages
  include:
    - go
    - python
    
  # Exclude specific languages even if detected
  exclude:
    - java
    - php
```

### Manual Configuration

```yaml
languages:
  auto_detect: false
  
  # Explicitly define languages
  languages:
    - name: go
      extensions: [".go"]
      directories: ["cmd", "internal", "pkg"]
    
    - name: python
      extensions: [".py"]
      files: ["requirements.txt", "setup.py"]
```

## Tool Configuration

### Go Tools

#### gofmt

```yaml
tools:
  gofmt:
    enabled: true
    args: ["-l", "."]
    fix_mode: true  # Run with -w to fix files
```

#### goimports

```yaml
tools:
  goimports:
    enabled: true
    args: ["-l", "."]
    fix_mode: true
```

#### golangci-lint

```yaml
tools:
  golangci-lint:
    enabled: true
    config: ".golangci.yml"  # Custom config file
    args: ["run", "./..."]
    timeout: 120
```

#### go vet

```yaml
tools:
  go-vet:
    enabled: true
    args: ["./..."]
```

#### gosec

```yaml
tools:
  gosec:
    enabled: true
    config: "gosec.json"
    severity: "medium"  # low, medium, high
    confidence: "medium"
```

### Python Tools

#### black

```yaml
tools:
  black:
    enabled: true
    line_length: 88
    target_version: ["py39"]
    fix_mode: true
```

#### isort

```yaml
tools:
  isort:
    enabled: true
    profile: "black"
    line_length: 88
    fix_mode: true
```

#### flake8

```yaml
tools:
  flake8:
    enabled: true
    config: ".flake8"
    max_line_length: 88
    ignore: ["E203", "W503"]
```

#### mypy

```yaml
tools:
  mypy:
    enabled: true
    config: "mypy.ini"
    strict: true
```

#### bandit

```yaml
tools:
  bandit:
    enabled: true
    config: ".bandit"
    severity: "medium"
    confidence: "medium"
```

### JavaScript/Node.js Tools

#### prettier

```yaml
tools:
  prettier:
    enabled: true
    config: ".prettierrc"
    fix_mode: true
```

#### eslint

```yaml
tools:
  eslint:
    enabled: true
    config: ".eslintrc.js"
    fix_mode: true
```

#### npm audit

```yaml
tools:
  npm-audit:
    enabled: true
    audit_level: "moderate"  # low, moderate, high, critical
```

### Universal Tools

#### gitleaks

```yaml
tools:
  gitleaks:
    enabled: true
    config: ".gitleaks.toml"
    verbose: false
```

#### shellcheck

```yaml
tools:
  shellcheck:
    enabled: true
    shell: "bash"  # bash, sh, zsh
    severity: "warning"  # error, warning, info, style
```

## Hook Configuration

### Git Hooks

```yaml
hooks:
  pre_commit:
    enabled: true
    tools:
      - gofmt
      - black
      - prettier
    
  pre_push:
    enabled: true
    tools:
      - golangci-lint
      - flake8
      - eslint
      - gosec
      - bandit
      - gitleaks
```

### Custom Hooks

```yaml
hooks:
  pre_commit:
    enabled: true
    
    # Custom commands
    custom_commands:
      - name: "custom-check"
        command: "./scripts/custom-check.sh"
        timeout: 60
```

## Workflow Configuration

### Staged Files Only

```yaml
workflows:
  pre_commit:
    staged_only: true  # Only check staged files
    
  manual:
    staged_only: false  # Check all files
```

### Parallel Execution

```yaml
settings:
  parallel: true
  max_parallel: 4  # Limit concurrent tools
```

### Error Handling

```yaml
settings:
  fail_fast: false  # Continue on errors
  
  # Define critical tools that must pass
  critical_tools:
    - gosec
    - bandit
    - gitleaks
```

## Advanced Configuration

### Custom Tools

```yaml
tools:
  custom_linter:
    enabled: true
    command: "./scripts/custom-linter.sh"
    args: ["--strict"]
    timeout: 120
    languages: ["go"]  # Only run for Go files
```

### Environment Variables

```yaml
settings:
  environment:
    CGO_ENABLED: "0"
    GOOS: "linux"
    
tools:
  golangci-lint:
    environment:
      GOLANGCI_LINT_CACHE: "/tmp/golangci-cache"
```

### Conditional Configuration

```yaml
# Different configurations for different branches
branches:
  main:
    tools:
      golangci-lint:
        args: ["run", "--timeout", "5m"]
  
  feature/*:
    tools:
      golangci-lint:
        args: ["run", "--fast"]
```

## Configuration Examples

### Minimal Configuration

```yaml
version: "1.0"
languages:
  auto_detect: true
```

### Comprehensive Go Project

```yaml
version: "1.0"

settings:
  parallel: true
  fail_fast: false
  timeout: 300

languages:
  auto_detect: true
  include: ["go"]

tools:
  gofmt:
    enabled: true
    fix_mode: true
    
  goimports:
    enabled: true
    fix_mode: true
    
  golangci-lint:
    enabled: true
    config: ".golangci.yml"
    timeout: 180
    
  gosec:
    enabled: true
    severity: "medium"
    
  gitleaks:
    enabled: true

hooks:
  pre_commit:
    enabled: true
    tools: ["gofmt", "goimports"]
    
  pre_push:
    enabled: true
    tools: ["golangci-lint", "gosec", "gitleaks"]
```

### Multi-language Project

```yaml
version: "1.0"

settings:
  parallel: true
  max_parallel: 6

languages:
  auto_detect: true

tools:
  # Go tools
  gofmt:
    enabled: true
    fix_mode: true
  golangci-lint:
    enabled: true
    
  # Python tools  
  black:
    enabled: true
    fix_mode: true
  flake8:
    enabled: true
    
  # JavaScript tools
  prettier:
    enabled: true
    fix_mode: true
  eslint:
    enabled: true
    
  # Universal tools
  gitleaks:
    enabled: true

hooks:
  pre_commit:
    enabled: true
    tools: ["gofmt", "black", "prettier"]
    
  pre_push:
    enabled: true
    tools: ["golangci-lint", "flake8", "eslint", "gitleaks"]
```

## Migration and Updates

When updating Go Quality Gate, configuration files are automatically migrated. However, you may need to update your configuration for new features:

```bash
# Check for configuration updates
quality-gate config check

# Update configuration with new defaults
quality-gate config update

# Validate current configuration
quality-gate config validate
```
