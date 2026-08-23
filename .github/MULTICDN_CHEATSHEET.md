# Multi-CDN Quick Reference Guide

## 🚀 Quick Start

### Linux/macOS
```bash
chmod +x .github/scripts/quickstart.sh
./.github/scripts/quickstart.sh
```

### Windows
```cmd
.github\scripts\quickstart.bat
```

## 📦 Configuration Templates

### Minimal Config (Auto-Detection)
```json
{
  "inbounds": [{"port": 10808, "protocol": "socks"}],
  "outbounds": [{
    "protocol": "vmess",
    "settings": {"vnext": [{"address": "server.com", "port": 443, "users": [{"id": "uuid"}]}]},
    "streamSettings": {
      "security": "tls",
      "tlsSettings": {
        "serverName": "onering-multi:server.com",
        "multiCDN": {"enabled": true}
      }
    }
  }]
}
```

### Telkomsel Optimized
```json
{
  "multiCDN": {
    "enabled": true,
    "strategy": "failover",
    "providers": [
      {"name": "cloudflare", "bugDomain": "zoom.us", "priority": 100, "isps": ["telkomsel"]},
      {"name": "cloudflare", "bugDomain": "ruangguru.com", "priority": 90, "isps": ["telkomsel"]}
    ]
  }
}
```

### Indosat Optimized
```json
{
  "multiCDN": {
    "enabled": true,
    "strategy": "round-robin",
    "providers": [
      {"name": "fastly", "bugDomain": "wa.me", "priority": 100, "isps": ["indosat"]},
      {"name": "cloudflare", "bugDomain": "zoom.us", "priority": 90, "isps": ["indosat"]}
    ]
  }
}
```

### XL Optimized
```json
{
  "multiCDN": {
    "enabled": true,
    "strategy": "latency-based",
    "providers": [
      {"name": "cloudflare", "bugDomain": "zoom.us", "priority": 100, "isps": ["xl"]},
      {"name": "cloudfront", "bugDomain": "meet.google.com", "priority": 90, "isps": ["xl"]}
    ]
  }
}
```

## 🔧 Selection Strategies

| Strategy | Use Case | Best For |
|----------|----------|----------|
| `round-robin` | Load balancing | General use |
| `failover` | Primary + backup | Stability |
| `latency-based` | Fastest CDN | Streaming |
| `health-based` | Reliability | Production |
| `random` | Anti-detection | Privacy |

## 📊 CDN Providers

| Provider | Bug Domains | ISP |
|----------|-------------|-----|
| **cloudflare** | zoom.us, teams.microsoft.com | Telkomsel, Indosat |
| **cloudfront** | aws.amazon.com, twitch.tv | XL |
| **fastly** | wa.me, github.com | Indosat |
| **akamai** | facebook.com, instagram.com | Telkomsel |
| **gcore** | discord.com | All |

## 🛡️ Evasion Features

```json
{
  "evasion": {
    "enableRotation": true,        // Auto-rotate CDN every 5m
    "rotateInterval": "5m",
    "enableJitter": true,          // Add random delays
    "jitterMaxMs": 50,
    "enablePadding": true,         // Add random padding
    "paddingMaxSize": 128,
    "randomizeFingerprint": true   // Random TLS fingerprint
  }
}
```

## 🏥 Health Check

```json
{
  "healthCheck": {
    "enabled": true,
    "interval": "30s",             // Check every 30s
    "timeout": "5s",               // 5s timeout
    "url": "https://cloudflare.com/cdn-cgi/trace"
  }
}
```

## 🔄 Failover

```json
{
  "failover": {
    "maxRetries": 3,               // Retry 3 times per CDN
    "blacklistDuration": "5m",     // Avoid failed CDN for 5m
    "fallbackToSingle": true       // Use single-CDN if all fail
  }
}
```

## 🌐 ISP Detection

```json
{
  "ispDetection": {
    "enabled": true,
    "method": "plmn",              // PLMN-based (mobile)
    "fallback": "latency"          // Fallback to latency fingerprint
  }
}
```

## 🐳 Docker Commands

```bash
# Pull
docker pull ghcr.io/YOUR_USERNAME/xray-core-onering:latest

# Run
docker run -d --name xray \
  -p 10808:10808 \
  -v $(pwd)/config.json:/etc/xray/config.json \
  ghcr.io/YOUR_USERNAME/xray-core-onering:latest

# Logs
docker logs -f xray

# Stop
docker stop xray && docker rm xray
```

## 📱 Android Integration

```gradle
dependencies {
    implementation files('libs/xray-onering-multicdn.aar')
}
```

## 🔍 Troubleshooting

### Check CDN Health
```bash
# View logs
tail -f /var/log/xray/error.log | grep "multicdn"

# Test config
./xray test -config config.json
```

### Connection Issues
1. Check CDN provider status
2. Try different bug domain
3. Enable debug logging: `"loglevel": "debug"`
4. Switch strategy: `"strategy": "random"`

### Performance Issues
1. Reduce health check interval
2. Disable rotation: `"enableRotation": false`
3. Use failover strategy
4. Optimize CDN priority

## 📞 Support

- Issues: https://github.com/YOUR_USERNAME/xray-core-onering/issues
- Discussions: https://github.com/YOUR_USERNAME/xray-core-onering/discussions
- PRD: [Multi-CDN PRD](../PRD_MULTI_CDN_ANTI_DPI.md)
