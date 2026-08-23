#!/bin/bash
# Quick start script for Xray-Core Onering Multi-CDN

set -e

echo "=========================================="
echo "Xray-Core Onering Multi-CDN Quick Start"
echo "=========================================="
echo ""

# Detect OS and architecture
OS=$(uname -s | tr '[:upper:]' '[:lower:]')
ARCH=$(uname -m)

case $ARCH in
    x86_64)
        ARCH="amd64"
        ;;
    aarch64|arm64)
        ARCH="arm64"
        ;;
    armv7l)
        ARCH="arm32-v7a"
        ;;
    *)
        echo "Unsupported architecture: $ARCH"
        exit 1
        ;;
esac

case $OS in
    linux)
        BINARY_NAME="xray-linux-64"
        if [[ "$ARCH" == "arm64" ]]; then
            BINARY_NAME="xray-linux-arm64-v8a"
        elif [[ "$ARCH" == "arm32-v7a" ]]; then
            BINARY_NAME="xray-linux-arm32-v7a"
        fi
        ;;
    darwin)
        BINARY_NAME="xray-macos-64"
        if [[ "$ARCH" == "arm64" ]]; then
            BINARY_NAME="xray-macos-arm64-v8a"
        fi
        ;;
    *)
        echo "Unsupported OS: $OS"
        exit 1
        ;;
esac

echo "Detected: $OS $ARCH"
echo "Binary: $BINARY_NAME"
echo ""

# Check if running from build artifacts directory
if [[ -f "./xray" || -f "./$BINARY_NAME" ]]; then
    echo "✓ Binary found locally"
    XRAY_BIN="./xray"
    if [[ ! -f "$XRAY_BIN" ]]; then
        XRAY_BIN="./$BINARY_NAME"
    fi
else
    echo "✗ Binary not found. Please download from GitHub Releases or build from source."
    echo ""
    echo "Download from: https://github.com/YOUR_USERNAME/xray-core-onering/releases"
    echo "Or build: go build -o xray ./main"
    exit 1
fi

# Make binary executable
chmod +x "$XRAY_BIN"

# Check if config exists
if [[ ! -f "config.json" ]]; then
    echo ""
    echo "Creating sample Multi-CDN config..."
    cat > config.json << 'EOF'
{
  "log": {
    "loglevel": "warning"
  },
  "inbounds": [{
    "port": 10808,
    "protocol": "socks",
    "settings": {
      "udp": true
    }
  }],
  "outbounds": [{
    "protocol": "vmess",
    "settings": {
      "vnext": [{
        "address": "your-server.com",
        "port": 443,
        "users": [{"id": "YOUR-UUID-HERE"}]
      }]
    },
    "streamSettings": {
      "network": "ws",
      "security": "tls",
      "tlsSettings": {
        "serverName": "onering-multi:your-server.com",
        "fingerprint": "chrome",
        "multiCDN": {
          "enabled": true,
          "strategy": "health-based",
          "providers": [
            {
              "name": "cloudflare",
              "bugDomain": "zoom.us",
              "priority": 100,
              "isps": ["telkomsel"]
            },
            {
              "name": "cloudfront",
              "bugDomain": "teams.microsoft.com",
              "priority": 90,
              "isps": ["xl", "indosat"]
            }
          ],
          "healthCheck": {
            "enabled": true,
            "interval": "30s"
          },
          "evasion": {
            "enableRotation": true,
            "rotateInterval": "5m"
          }
        }
      }
    }
  }]
}
EOF
    echo "✓ Sample config created: config.json"
    echo ""
    echo "⚠️  IMPORTANT: Edit config.json and replace:"
    echo "   - your-server.com with your actual server"
    echo "   - YOUR-UUID-HERE with your UUID"
    echo ""
    read -p "Press Enter to continue after editing config.json..."
fi

# Validate config
echo ""
echo "Validating configuration..."
if $XRAY_BIN test -config config.json 2>&1; then
    echo "✓ Configuration is valid"
else
    echo "✗ Configuration has errors. Please fix config.json"
    exit 1
fi

# Show Multi-CDN info
echo ""
echo "=========================================="
echo "Multi-CDN Features Enabled"
echo "=========================================="
echo ""
echo "✓ Multi-CDN Anti-DPI bypass"
echo "✓ Automatic failover"
echo "✓ Health monitoring"
echo "✓ ISP-specific optimization"
echo "✓ DPI evasion techniques"
echo ""

# Start Xray
echo "Starting Xray-Core Onering Multi-CDN..."
echo ""
echo "SOCKS5 Proxy: 127.0.0.1:10808"
echo ""
echo "Press Ctrl+C to stop"
echo ""

exec $XRAY_BIN run -config config.json
