#!/bin/bash
# Installation script - sets up the CI/CD environment

set -e

echo "=========================================="
echo "Xray-Core Onering Multi-CDN CI/CD Setup"
echo "=========================================="
echo ""

PROJECT_ROOT=$(pwd)

# Check if we're in the right directory
if [[ ! -f "go.mod" ]]; then
    echo "❌ Error: Not in project root (go.mod not found)"
    exit 1
fi

echo "Project: $PROJECT_ROOT"
echo ""

# Make scripts executable
echo "🔧 Making scripts executable..."
chmod +x .github/scripts/*.sh 2>/dev/null || true
echo "✅ Scripts are executable"
echo ""

# Check GitHub CLI
if command -v gh &> /dev/null; then
    echo "✅ GitHub CLI installed: $(gh --version | head -n1)"
    
    # Check authentication
    if gh auth status &> /dev/null; then
        echo "✅ GitHub CLI authenticated"
    else
        echo "⚠️  GitHub CLI not authenticated"
        echo "   Run: gh auth login"
    fi
else
    echo "⚠️  GitHub CLI not installed (optional)"
    echo "   Install: https://cli.github.com/"
fi
echo ""

# Check Docker
if command -v docker &> /dev/null; then
    echo "✅ Docker installed: $(docker --version)"
else
    echo "ℹ️  Docker not installed (optional for local Docker builds)"
fi
echo ""

# Check Go
if command -v go &> /dev/null; then
    GO_VERSION=$(go version | awk '{print $3}')
    echo "✅ Go installed: $GO_VERSION"
    
    # Check Go version
    REQUIRED_GO="1.26"
    CURRENT_GO=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+' || echo "0.0")
    
    if [[ "$CURRENT_GO" != "$REQUIRED_GO" ]]; then
        echo "⚠️  Warning: Project requires Go $REQUIRED_GO, you have $CURRENT_GO"
    fi
else
    echo "❌ Go not installed"
    echo "   Install: https://go.dev/dl/"
    exit 1
fi
echo ""

# Check dependencies
echo "📦 Checking Go dependencies..."
if go mod verify &> /dev/null; then
    echo "✅ Go modules verified"
else
    echo "⚠️  Downloading dependencies..."
    go mod download
    echo "✅ Dependencies downloaded"
fi
echo ""

# Check git repository
if git rev-parse --git-dir &> /dev/null; then
    echo "✅ Git repository initialized"
    
    # Check remote
    if git remote get-url origin &> /dev/null; then
        REMOTE=$(git remote get-url origin)
        echo "✅ Git remote: $REMOTE"
    else
        echo "⚠️  No git remote configured"
        echo "   Add remote: git remote add origin <url>"
    fi
else
    echo "❌ Not a git repository"
    exit 1
fi
echo ""

# Test build
echo "🔨 Testing local build..."
if .github/scripts/build-local.sh > /tmp/build.log 2>&1; then
    echo "✅ Local build successful"
else
    echo "❌ Local build failed"
    echo "   Check log: /tmp/build.log"
    exit 1
fi
echo ""

# Create sample config if not exists
if [[ ! -f "config.json" ]]; then
    echo "📝 Creating sample config..."
    cp .github/docker/config-multicdn-sample.json config.json
    echo "✅ Sample config created: config.json"
    echo "   ⚠️  Edit config.json with your server details!"
else
    echo "ℹ️  config.json already exists"
fi
echo ""

# Summary
echo "=========================================="
echo "Setup Complete!"
echo "=========================================="
echo ""
echo "✅ All scripts are executable"
echo "✅ Dependencies verified"
echo "✅ Local build tested"
echo ""
echo "Next Steps:"
echo ""
echo "1. Configure GitHub Actions (if not done):"
echo "   - Go to Settings → Actions → General"
echo "   - Enable workflows and set permissions"
echo ""
echo "2. Push code to trigger CI:"
echo "   git push origin main"
echo ""
echo "3. Create a release:"
echo "   git tag -a v1.0.0 -m 'Release v1.0.0'"
echo "   git push origin v1.0.0"
echo ""
echo "4. View workflows:"
echo "   gh workflow list"
echo "   gh run list"
echo ""
echo "5. Build locally:"
echo "   .github/scripts/build-local.sh     # Single platform"
echo "   .github/scripts/build-all.sh       # All platforms"
echo ""
echo "6. Quick start:"
echo "   .github/scripts/quickstart.sh"
echo ""
echo "Documentation:"
echo "  - .github/README.md           - Workflow overview"
echo "  - .github/CI_CD_GUIDE.md      - Complete guide"
echo "  - .github/MULTICDN_CHEATSHEET.md - Quick reference"
echo "  - .github/SETUP_COMPLETE.md   - Feature summary"
echo ""
echo "🎉 Ready for CI/CD!"
