[![en](https://img.shields.io/badge/lang-en-red.svg)](README.md)
[![pt-br](https://img.shields.io/badge/lang-pt--br-green.svg)](README.pt-BR.md)

<div align="center">

<img src="docs/gopher.png" alt="Go Quality Gate Logo" width="400">

# Go Quality Gate

[![Go Version](https://img.shields.io/badge/Go-1.24+-00ADD8?style=flat&logo=go)](https://golang.org/)
[![License](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![CI](https://github.com/dmux/go-quality-gate/workflows/CI/badge.svg)](https://github.com/dmux/go-quality-gate/actions/workflows/ci.yml)
[![Release](https://github.com/dmux/go-quality-gate/workflows/Release/badge.svg)](https://github.com/dmux/go-quality-gate/actions/workflows/release.yml)
[![Docker](https://img.shields.io/badge/Docker-ghcr.io-blue?logo=docker)](https://github.com/dmux/go-quality-gate/pkgs/container/go-quality-gate)
[![PRs Welcome](https://img.shields.io/badge/PRs-welcome-brightgreen.svg)](https://github.com/dmux/go-quality-gate/pulls)

</div>

**Language-agnostic code quality control tool with Git hooks**

A code quality control tool built in Go, distributed as a single binary with no external runtime dependencies. Provides enhanced visual feedback with spinners, execution timing, and structured JSON output.

## ✨ Key Features

- **🏗️ Single Binary**: Zero runtime dependencies (Python, Node.js)
- **🔧 Automatic Setup**: Installs quality tools automatically
- **🌍 Multi-language**: Supports multiple languages in the same repository
- **📊 Observability**: Spinners, timing, and real-time visual feedback
- **🔒 Built-in Security**: Secret scanning in commit workflow
- **⚡ Native Performance**: Instant execution without interpreters
- **🚀 CI/CD Ready**: Clean JSON output for automation pipelines
- **🤖 MCP Server**: Model Context Protocol support for AI coding agents (Claude, Cursor, etc.)

## 🚀 Quick Start

### 1. Installation

#### Option A: Download Pre-built Binary (Recommended)

```bash
# Download the latest release for your platform
# Linux (x64)
wget https://github.com/dmux/go-quality-gate/releases/latest/download/quality-gate-linux-amd64
chmod +x quality-gate-linux-amd64
sudo mv quality-gate-linux-amd64 /usr/local/bin/quality-gate

# Linux (ARM64)
wget https://github.com/dmux/go-quality-gate/releases/latest/download/quality-gate-linux-arm64
chmod +x quality-gate-linux-arm64
sudo mv quality-gate-linux-arm64 /usr/local/bin/quality-gate

# macOS (Intel)
wget https://github.com/dmux/go-quality-gate/releases/latest/download/quality-gate-darwin-amd64
chmod +x quality-gate-darwin-amd64
sudo mv quality-gate-darwin-amd64 /usr/local/bin/quality-gate

# macOS (Apple Silicon)
wget https://github.com/dmux/go-quality-gate/releases/latest/download/quality-gate-darwin-arm64
chmod +x quality-gate-darwin-arm64
sudo mv quality-gate-darwin-arm64 /usr/local/bin/quality-gate

# Windows (download .exe from releases page)
# https://github.com/dmux/go-quality-gate/releases/latest
```

#### Option B: Using Go Install

```bash
go install github.com/dmux/go-quality-gate/cmd/quality-gate@latest
```

#### Option C: Using Docker

```bash
# Run directly with Docker
docker run --rm ghcr.io/dmux/go-quality-gate:latest --version

# Create alias for easier usage
echo 'alias quality-gate="docker run --rm -v $(pwd):/workspace -w /workspace ghcr.io/dmux/go-quality-gate:latest"' >> ~/.bashrc
source ~/.bashrc
```

#### Option D: Build from Source

```bash
# Clone and build
git clone https://github.com/dmux/go-quality-gate.git
cd go-quality-gate
make build

# Install hooks in your project
./quality-gate --install
```

### 2. Configuration

Create a `quality.yml` in your project:

```bash
# Generate initial configuration based on your project
./quality-gate --init
```

### 3. Usage

```bash
# Automatic execution via Git hooks
git commit -m "feat: new feature"

# Manual execution
./quality-gate pre-commit

# JSON output for CI/CD
./quality-gate --output=json pre-commit

# Auto-fix
./quality-gate --fix pre-commit

# Run as Model Context Protocol (MCP) Server for AI Agents
./quality-gate mcp
```

## ⚙️ Configuration (quality.yml)

```yaml
tools:
  - name: "Gitleaks"
    check_command: "gitleaks version"
    install_command: "go install github.com/zricethezav/gitleaks/v8@latest"
  - name: "Ruff (Python)"
    check_command: "ruff --version"
    install_command: "pip install ruff"

hooks:
  security:
    pre-commit:
      - name: "🔒 Security Check"
        command: "gitleaks detect --no-git --source . --verbose"
        output_rules:
          on_failure_message: "Secret leak detected!"

  python-backend:
    pre-commit:
      - name: "🎨 Format Check (Ruff)"
        command: "ruff format ./backend --check"
        fix_command: "ruff format ./backend"
        output_rules:
          show_on: failure
          on_failure_message: "Run './quality-gate --fix' to format."

  typescript-frontend:
    pre-commit:
      - name: "🎨 Format Check (Prettier)"
        command: "npx prettier --check 'frontend/**/*.{ts,tsx}'"
        fix_command: "npx prettier --write 'frontend/**/*.{ts,tsx}'"
```

## 📘 How to Use

### 1. Build

```bash
go build -o quality-gate ./cmd/quality-gate
```

This will create an executable named `quality-gate` in the current directory.

### 2. Install Git Hooks

```bash
./quality-gate --install
```

The program will automatically configure `pre-commit` and `pre-push` hooks.

### 3. Advanced Commands

- **`./quality-gate --init`**: (Experimental) Analyzes your project structure and generates an initial `quality.yml` file with suggestions
- **`./quality-gate --fix`**: Executes automatic fix commands (`fix_command`) defined in your `quality.yml`
- **`./quality-gate pre-commit --output=json`**: Executes the specified hook and returns the result in JSON format

### 4. Configuration (quality.yml)

The configuration is divided into two main sections:

- **`tools`**: List of tools required for the project

  - `name`: Human-readable tool name
  - `check_command`: Command that returns success (exit code 0) if the tool is installed
  - `install_command`: Command executed to install the tool if `check_command` fails

- **`hooks`**: Quality check configuration

#### Tool Validation Cache

On the first `pre-commit` or `pre-push` execution, Quality Gate runs each
configured `check_command` and installs any missing tool with its
`install_command`. After every tool has been successfully validated, it stores
a SHA-256 hash of the `tools` configuration in:

```text
.git/quality-gate/tools.sha256
```

Subsequent executions skip the installation checks while the configuration is
unchanged. Changing a tool's `name`, `check_command`, or `install_command`, as
well as adding, removing, or reordering tools, invalidates the cache and causes
all tools to be validated again.

The cache is only updated after a successful validation. If it is missing,
outdated, unreadable, or cannot be written, Quality Gate falls back to checking
the tools normally without blocking the configured hooks. To force a new
validation—for example, after manually uninstalling a tool—delete the cache
file before running a hook:

```bash
rm .git/quality-gate/tools.sha256
```

#### Complete Example

```yaml
tools:
  - name: "Gitleaks"
    check_command: "gitleaks version"
    install_command: "go install github.com/zricethezav/gitleaks/v8@latest"
  - name: "Ruff (Python Linter/Formatter)"
    check_command: "ruff --version"
    install_command: "pip install ruff"
  - name: "Prettier (Code Formatter)"
    check_command: "npx prettier --version"
    install_command: "npm install --global prettier"

hooks:
  security:
    pre-commit:
      - name: "🔒 Security Check (Gitleaks)"
        command: "gitleaks detect --no-git --source . --verbose"
        output_rules:
          on_failure_message: "Secret leak detected! Review code before committing."

  python-backend:
    pre-commit:
      - name: "🎨 Format Check (Ruff)"
        command: "ruff format ./backend --check"
        fix_command: "ruff format ./backend"
        output_rules:
          show_on: failure
          on_failure_message: "Code formatting issue. Run './quality-gate --fix' to fix."
      - name: "🧪 Tests (Pytest)"
        command: "pytest ./backend"
        output_rules:
          show_on: always

  typescript-frontend:
    pre-commit:
      - name: "🎨 Format Check (Prettier)"
        command: "npx prettier --check 'frontend/**/*.{ts,tsx}'"
        fix_command: "npx prettier --write 'frontend/**/*.{ts,tsx}'"
```

> With `--parallel`, the three `pre-commit` hooks above (Gitleaks, Ruff format
> check, Pytest) run concurrently instead of one after another — useful here
> since Pytest can be the slow one. Run with:
> `./quality-gate --parallel pre-commit`

## 📋 Available Commands

| Command         | Description                                             | Example                                   |
| --------------- | ------------------------------------------------------- | ----------------------------------------- |
| `--install`     | Installs Git hooks in repository                        | `./quality-gate --install`                |
| `--init`        | Generates initial quality.yml with intelligent analysis | `./quality-gate --init`                   |
| `--fix`         | Executes automatic fixes                                | `./quality-gate --fix pre-commit`         |
| `mcp`           | Runs as an MCP server for AI integration                | `./quality-gate mcp`                      |
| `--version, -v` | Shows version information                               | `./quality-gate --version`                |
| `--output=json` | Structured output for CI/CD                             | `./quality-gate --output=json pre-commit` |
| `--parallel`    | Runs independent hooks concurrently (opt-in)            | `./quality-gate --parallel pre-commit`    |

### 📊 Version Information

```bash
# Simple version
./quality-gate --version
# Output: quality-gate version 1.2.0

# JSON version with build details
./quality-gate --version --output json
# Output:
{
  "version": "1.2.0",
  "build_date": "2025-10-21T16:34:44Z",
  "git_commit": "f7b01a2"
}
```

## 🎯 JSON Output for CI/CD

```json
{
  "status": "success",
  "results": [
    {
      "hook": {
        "Name": "🔒 Security Check",
        "Command": "gitleaks detect --source ."
      },
      "success": true,
      "output": "",
      "duration_ms": 150,
      "duration": "150ms"
    }
  ]
}
```

## 🤖 Model Context Protocol (MCP) Integration

Go Quality Gate can act as an MCP server, providing standard AI coding agents (like Claude Desktop, Cursor, or Cline) with the ability to interact with your codebase's quality tools natively.

### Connecting to an Agent
To use `go-quality-gate` with Cursor or Claude, simply configure the MCP server to run via standard I/O (stdio). For Cursor, create or update `.mcp.json` in your workspace:

```json
{
  "mcpServers": {
    "go-quality-gate": {
      "command": "./quality-gate",
      "args": ["mcp"]
    }
  }
}
```

The server exposes the following MCP Tools to your AI:
1. `run_quality_checks`: Executes linters and formatting checks, returning a parsed diagnostic for the AI.
2. `run_auto_fix`: Allows the AI to automatically trigger the formatting fixes configured in `quality.yml`.

## 🛠️ Development

### Prerequisites

- Go 1.18+
- Git
- Package managers for your project languages (pip, npm, etc.)

### Local Setup

```bash
# Clone the repository
git clone <repo>
cd go-quality-gate

# Install dependencies
go mod tidy

# Build
go build -o quality-gate ./cmd/quality-gate

# Run tests
go test ./...

# Test locally
./quality-gate --init
./quality-gate --install
```

### Architecture

```text
cmd/quality-gate/     # Main application
internal/
  domain/            # Entities and business rules
  service/           # Application logic
  infra/             # Infrastructure (git, shell, logger)
  repository/        # Persistence interfaces
  config/            # Configuration and parsing
```

## 🤝 Contributing

1. **Fork** the project
2. **Create** a branch: `git checkout -b feature/new-feature`
3. **Commit** your changes: `git commit -m 'feat: new feature'`
4. **Push** to the branch: `git push origin feature/new-feature`
5. **Open** a Pull Request

See [TODO.md](TODO.md) for detailed roadmap and available tasks.

## 📄 License

This project is under the MIT license. See the [LICENSE](LICENSE) file for more details.

---

**Version**: v1.1.x  
**Status**: Active development  
**Complete documentation**: [TODO.md](TODO.md)
