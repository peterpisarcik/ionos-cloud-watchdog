#!/bin/bash

set -e

# Colors for output
GREEN='\033[0;32m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

# Get version from git tag or use "dev"
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_TIME=$(date -u '+%Y-%m-%d_%H:%M:%S')

# Output directory
DIST_DIR="dist"
BIN_NAME="ionos-cloud-watchdog"

echo -e "${BLUE}Building ${BIN_NAME} ${VERSION}${NC}"
echo ""

# Clean previous builds
if [ -d "$DIST_DIR" ]; then
    echo "Cleaning previous builds..."
    rm -rf "$DIST_DIR"
fi

mkdir -p "$DIST_DIR"

# Build for multiple platforms
build() {
    local GOOS=$1
    local GOARCH=$2
    local OUTPUT_NAME="${BIN_NAME}-${GOOS}-${GOARCH}"

    if [ "$GOOS" = "windows" ]; then
        OUTPUT_NAME="${OUTPUT_NAME}.exe"
    fi

    echo -e "${GREEN}Building${NC} ${GOOS}/${GOARCH}..."

    GOOS=$GOOS GOARCH=$GOARCH go build \
        -ldflags "-X main.Version=${VERSION} -X main.BuildTime=${BUILD_TIME}" \
        -o "${DIST_DIR}/${OUTPUT_NAME}" \
        ./cmd/ionos-cloud-watchdog

    echo "  → ${DIST_DIR}/${OUTPUT_NAME}"
}

# Build for all target platforms
build darwin arm64   # macOS Apple Silicon
build darwin amd64   # macOS Intel
build linux amd64    # Linux
build windows amd64  # Windows

echo ""
echo -e "${GREEN}✓ Build complete!${NC}"
echo ""
echo "Binaries available in ${DIST_DIR}/"
ls -lh "$DIST_DIR"
