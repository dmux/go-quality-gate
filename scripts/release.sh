#!/bin/bash

# Release script for go-quality-gate
# This script helps create a new release with proper tagging

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Function to print colored output
print_info() {
    echo -e "${BLUE}ℹ️  $1${NC}"
}

print_success() {
    echo -e "${GREEN}✅ $1${NC}"
}

print_warning() {
    echo -e "${YELLOW}⚠️  $1${NC}"
}

print_error() {
    echo -e "${RED}❌ $1${NC}"
}

# Check if we're in the right directory
if [[ ! -f "go.mod" ]] || [[ ! -f "cmd/quality-gate/main.go" ]]; then
    print_error "This script must be run from the project root directory"
    exit 1
fi

# Check if git is clean
if [[ -n $(git status --porcelain) ]]; then
    print_error "Git working directory is not clean. Please commit or stash your changes."
    git status --short
    exit 1
fi

# Check if we're on main branch
CURRENT_BRANCH=$(git branch --show-current)
if [[ "$CURRENT_BRANCH" != "main" ]]; then
    print_warning "You're not on the main branch (current: $CURRENT_BRANCH)"
    read -p "Do you want to continue? (y/N): " -n 1 -r
    echo
    if [[ ! $REPLY =~ ^[Yy]$ ]]; then
        print_info "Release cancelled"
        exit 0
    fi
fi

# Get current version from git tags
CURRENT_VERSION=$(git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
print_info "Current version: $CURRENT_VERSION"

# Ask for new version
echo
read -p "Enter new version (e.g., v1.2.1): " NEW_VERSION

# Validate version format
if [[ ! $NEW_VERSION =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
    print_error "Invalid version format. Please use semantic versioning (e.g., v1.2.1)"
    exit 1
fi

# Check if tag already exists
if git rev-parse "$NEW_VERSION" >/dev/null 2>&1; then
    print_error "Tag $NEW_VERSION already exists"
    exit 1
fi

# Confirm release
echo
print_info "Release Summary:"
print_info "  Current version: $CURRENT_VERSION"
print_info "  New version: $NEW_VERSION"
print_info "  Branch: $CURRENT_BRANCH"
echo

read -p "Create release $NEW_VERSION? (y/N): " -n 1 -r
echo
if [[ ! $REPLY =~ ^[Yy]$ ]]; then
    print_info "Release cancelled"
    exit 0
fi

# Update version in Makefile if needed
if [[ -f "Makefile" ]]; then
    MAKEFILE_VERSION=$(grep "VERSION?=" Makefile | cut -d'=' -f2)
    if [[ "$MAKEFILE_VERSION" != "${NEW_VERSION#v}" ]]; then
        print_info "Updating Makefile version from $MAKEFILE_VERSION to ${NEW_VERSION#v}"
        sed -i.bak "s/VERSION?=.*/VERSION?=${NEW_VERSION#v}/" Makefile
        rm -f Makefile.bak
        git add Makefile
        git commit -m "chore: bump version to $NEW_VERSION"
    fi
fi

# Create and push tag
print_info "Creating tag $NEW_VERSION..."
git tag -a "$NEW_VERSION" -m "Release $NEW_VERSION"

print_info "Pushing tag to origin..."
git push origin "$NEW_VERSION"

# Push any pending commits
if [[ -n $(git log origin/main..HEAD) ]]; then
    print_info "Pushing commits to origin..."
    git push origin "$CURRENT_BRANCH"
fi

print_success "Release $NEW_VERSION created successfully!"
echo
print_info "The GitHub Actions workflow will now:"
print_info "  1. Run tests"
print_info "  2. Build binaries for multiple platforms"
print_info "  3. Create GitHub release with assets"
print_info "  4. Build and push Docker images"
echo
print_info "You can monitor the progress at:"
print_info "  https://github.com/dmux/go-quality-gate/actions"
echo
print_success "🚀 Release process initiated!"