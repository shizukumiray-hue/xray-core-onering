#!/bin/bash
# Build script for local development

set -e

echo "========================================"
echo "Xray-Core Onering Multi-CDN Local Build"
echo "========================================"
echo ""

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64)
        GOARCH="amd64"
        ;;
    aarch64|arm64)
        GOARCH="arm64"
        ;;
    armv7l)
        GOARCH="arm"
        GOARM="7"
        ;;
    *)
        echo "❌ Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

case $OS in
    linux)
        GOOS="linux"
        OUTPUT="xray"
        ;;
    darwin)
        GOOS="darwin"
        OUTPUT="xray"
        ;;
    mingw*|msys*|cygwin*)
        GOOS="windows"
        OUTPUT="xray.exe"
        ;;
    *)
        echo "❌ Unsupported OS: $OS"
        exit 1
        ;;
esac

echo "Target: $GOOS/$GOARCH"
echo ""

# Get version
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "Version: $VERSION"
echo "Build Date: $BUILD_DATE"
echo ""

# Check Go installation
if ! command -v go &> /dev/null; then
    echo "❌ Go is not installed"
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo "Go Version: $GO_VERSION"
echo ""

# Download dependencies
echo "📦 Downloading dependencies..."
go mod download
echo "✅ Dependencies downloaded"
echo ""

# Run tests
if [[ "${SKIP_TESTS:-}" != "true" ]]; then
    echo "🧪 Running tests..."
    go test -short ./... || {
        echo "⚠️  Tests failed, but continuing build..."
    }
    echo ""
fi

# Build
echo "🔨 Building binary..."
LDFLAGS="-X github.com/xtls/xray-core/core.build=${VERSION}"
LDFLAGS="${LDFLAGS} -X github.com/xtls/xray-core/core.buildDate=${BUILD_DATE}"
LDFLAGS="${LDFLAGS} -s -w -buildid="

CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH ${GOARM:+GOARM=$GOARM} \
    go build \
    -o "$OUTPUT" \
    -trimpath \
    -buildvcs=false \
    -gcflags="all=-l=4" \
    -ldflags="$LDFLAGS" \
    -v \
    ./main

if [[ -f "$OUTPUT" ]]; then
    SIZE=$(ls -lh "$OUTPUT" | awk '{print $5}')
    echo ""
    echo "✅ Build successful!"
    echo "   Output: $OUTPUT"
    echo "   Size: $SIZE"
    echo ""
    
    # Test binary
    echo "🔍 Testing binary..."
    ./"$OUTPUT" version
    echo ""
    
    echo "🎉 Build complete!"
    echo ""
    echo "Run with:"
    echo "  ./$OUTPUT run -config config.json"
else
    echo "❌ Build failed"
    exit 1
fi
