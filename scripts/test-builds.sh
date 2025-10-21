#!/bin/bash

# Local build test script
# This script simulates the multi-architecture builds locally

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

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

print_info "Starting local multi-architecture build test..."

# Clean previous builds
print_info "Cleaning previous builds..."
rm -f quality-gate-*
rm -f *.sha256

# Test configurations
declare -a configs=(
    "linux:amd64"
    "linux:arm64"
    "darwin:amd64"
    "darwin:arm64" 
    "windows:amd64"
)

SUCCESS_COUNT=0
TOTAL_COUNT=${#configs[@]}

print_info "Testing builds for ${TOTAL_COUNT} platforms..."
echo

# Build for each configuration
for config in "${configs[@]}"; do
    IFS=':' read -r goos goarch <<< "$config"
    
    EXTENSION=""
    if [[ "$goos" == "windows" ]]; then
        EXTENSION=".exe"
    fi
    
    BINARY_NAME="quality-gate-${goos}-${goarch}${EXTENSION}"
    
    print_info "Building for ${goos}/${goarch}..."
    
    # Set environment and build
    if GOOS=$goos GOARCH=$goarch CGO_ENABLED=0 go build \
        -ldflags "-X main.Version=test -X main.BuildDate=$(date -u +%Y-%m-%dT%H:%M:%SZ) -X main.GitCommit=test123 -s -w" \
        -o "$BINARY_NAME" ./cmd/quality-gate; then
        
        print_success "✅ ${goos}/${goarch} build successful"
        
        # Generate checksum
        if command -v sha256sum > /dev/null; then
            sha256sum "$BINARY_NAME" > "${BINARY_NAME}.sha256"
        elif command -v shasum > /dev/null; then
            shasum -a 256 "$BINARY_NAME" > "${BINARY_NAME}.sha256"
        else
            print_warning "No SHA256 utility found, skipping checksum"
        fi
        
        # Show file info
        ls -la "$BINARY_NAME"
        
        # Test execution (only for native platform)
        if [[ "$goos" == "$(go env GOOS)" ]] && [[ "$goarch" == "$(go env GOARCH)" ]]; then
            print_info "Testing binary execution..."
            if ./"$BINARY_NAME" --version > /dev/null 2>&1; then
                print_success "Binary executes correctly"
            else
                print_warning "Binary execution test failed (might be expected for cross-compilation)"
            fi
        fi
        
        ((SUCCESS_COUNT++))
        
    else
        print_error "❌ ${goos}/${goarch} build failed"
    fi
    
    echo
done

# Test Docker build if Docker is available
if command -v docker > /dev/null; then
    print_info "Testing Docker build..."
    
    if docker build \
        --build-arg VERSION=test \
        --build-arg BUILD_DATE="$(date -u +%Y-%m-%dT%H:%M:%SZ)" \
        --build-arg GIT_COMMIT=test123 \
        -t quality-gate:test .; then
        
        print_success "Docker build successful"
        
        # Test Docker run
        print_info "Testing Docker execution..."
        if docker run --rm quality-gate:test --version > /dev/null 2>&1; then
            print_success "Docker image executes correctly"
        else
            print_warning "Docker execution test failed"
        fi
        
    else
        print_error "Docker build failed"
    fi
else
    print_warning "Docker not available, skipping Docker build test"
fi

# Summary
echo
print_info "=== Build Test Summary ==="
print_success "Successful builds: ${SUCCESS_COUNT}/${TOTAL_COUNT}"

if [[ $SUCCESS_COUNT -eq $TOTAL_COUNT ]]; then
    print_success "🎉 All builds passed! Ready for release."
else
    print_error "❌ Some builds failed. Check the output above."
    exit 1
fi

# List generated files
echo
print_info "Generated files:"
ls -la quality-gate-* 2>/dev/null || echo "No build artifacts found"

echo
print_info "To clean up: rm -f quality-gate-* *.sha256"