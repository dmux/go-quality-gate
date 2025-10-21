---
layout: page
title: Installation
permalink: /installation/
---

# Installation Guide

Go Quality Gate can be installed in several ways depending on your needs and environment.

## Option A: Download Pre-built Binary (Recommended)

### Linux (x64)
```bash
wget https://github.com/dmux/go-quality-gate/releases/latest/download/quality-gate-linux-amd64
chmod +x quality-gate-linux-amd64
sudo mv quality-gate-linux-amd64 /usr/local/bin/quality-gate
```

### Linux (ARM64)
```bash
wget https://github.com/dmux/go-quality-gate/releases/latest/download/quality-gate-linux-arm64
chmod +x quality-gate-linux-arm64
sudo mv quality-gate-linux-arm64 /usr/local/bin/quality-gate
```

### macOS (x64)
```bash
wget https://github.com/dmux/go-quality-gate/releases/latest/download/quality-gate-darwin-amd64
chmod +x quality-gate-darwin-amd64
sudo mv quality-gate-darwin-amd64 /usr/local/bin/quality-gate
```

### macOS (ARM64/M1+)
```bash
wget https://github.com/dmux/go-quality-gate/releases/latest/download/quality-gate-darwin-arm64
chmod +x quality-gate-darwin-arm64
sudo mv quality-gate-darwin-arm64 /usr/local/bin/quality-gate
```

### Windows (x64)
```powershell
# Download via PowerShell
Invoke-WebRequest -Uri "https://github.com/dmux/go-quality-gate/releases/latest/download/quality-gate-windows-amd64.exe" -OutFile "quality-gate.exe"
# Move to a directory in your PATH
```

## Option B: Build from Source

### Prerequisites
- Go 1.24 or later
- Git

### Build Steps
```bash
# Clone the repository
git clone https://github.com/dmux/go-quality-gate.git
cd go-quality-gate

# Build the binary
make build

# Install to system
sudo cp quality-gate /usr/local/bin/
```

## Option C: Docker

### Pull from GitHub Container Registry
```bash
docker pull ghcr.io/dmux/go-quality-gate:latest
```

### Run with Docker
```bash
# Run in current directory
docker run --rm -v $(pwd):/workspace -w /workspace ghcr.io/dmux/go-quality-gate:latest

# Create an alias for easier usage
echo 'alias quality-gate="docker run --rm -v $(pwd):/workspace -w /workspace ghcr.io/dmux/go-quality-gate:latest"' >> ~/.bashrc
source ~/.bashrc
```

## Verification

After installation, verify that Go Quality Gate is working correctly:

```bash
quality-gate --version
```

You should see output similar to:
```
Go Quality Gate v1.x.x
```

## Next Steps

- [Learn how to use Go Quality Gate](usage.html)
- [Configure for your project](configuration.html)

## Troubleshooting

### Permission Denied
If you encounter permission errors, make sure the binary is executable:
```bash
chmod +x quality-gate
```

### Command Not Found
Ensure the binary is in a directory that's in your `PATH`. You can check your PATH with:
```bash
echo $PATH
```

Common directories in PATH include:
- `/usr/local/bin`
- `/usr/bin`
- `~/bin`

### Docker Issues
If you're having issues with Docker, make sure:
1. Docker is installed and running
2. You have permission to run Docker commands
3. The volume mount path is correct for your system