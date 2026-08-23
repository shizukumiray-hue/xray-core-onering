#!/bin/bash
# Build all platforms locally (for testing multi-platform builds)

set -e

echo "========================================"
echo "Multi-Platform Build Script"
echo "========================================"
echo ""

# Build matrix
declare -a BUILDS=(
    "linux:amd64:linux-64"
    "linux:arm64:linux-arm64-v8a"
    "linux:arm:linux-arm32-v7a:7"
    "darwin:amd64:macos-64"
    "darwin:arm64:macos-arm64-v8a"
    "windows:amd64:windows-64"
)

# Output directory
OUTPUT_DIR="dist"
mkdir -p "$OUTPUT_DIR"

# Get version
VERSION=$(git describe --tags --always --dirty 2>/dev/null || echo "dev")
BUILD_DATE=$(date -u +"%Y-%m-%dT%H:%M:%SZ")

echo "Version: $VERSION"
echo "Build Date: $BUILD_DATE"
echo "Output Directory: $OUTPUT_DIR"
echo ""

# LDFLAGS
LDFLAGS="-X github.com/xtls/xray-core/core.build=${VERSION}"
LDFLAGS="${LDFLAGS} -s -w -buildid="

# Build counter
SUCCESS=0
FAILED=0

# Build each platform
for BUILD in "${BUILDS[@]}"; do
    IFS=':' read -r GOOS GOARCH NAME GOARM <<< "$BUILD"
    
    OUTPUT="xray-$NAME"
    if [[ "$GOOS" == "windows" ]]; then
        OUTPUT="${OUTPUT}.exe"
    fi
    
    echo "Building: $NAME ($GOOS/$GOARCH)"
    
    if CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH ${GOARM:+GOARM=$GOARM} \
        go build \
        -o "$OUTPUT_DIR/$OUTPUT" \
        -trimpath \
        -buildvcs=false \
        -gcflags="all=-l=4" \
        -ldflags="$LDFLAGS" \
        ./main 2>/dev/null; then
        
        SIZE=$(ls -lh "$OUTPUT_DIR/$OUTPUT" | awk '{print $5}')
        echo "  ✅ Success ($SIZE)"
        ((SUCCESS++))
    else
        echo "  ❌ Failed"
        ((FAILED++))
    fi
    echo ""
done

# Summary
echo "========================================"
echo "Build Summary"
echo "========================================"
echo "Success: $SUCCESS"
echo "Failed: $FAILED"
echo ""

if [[ $FAILED -eq 0 ]]; then
    echo "🎉 All builds successful!"
    echo ""
    echo "Binaries in: $OUTPUT_DIR/"
    ls -lh "$OUTPUT_DIR/"
    exit 0
else
    echo "⚠️  Some builds failed"
    exit 1
fi
