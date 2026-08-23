# Xray-Core Onering - Multi-CDN Phase 2 Examples

This directory contains example configurations for Xray-Core Onering with Phase 2 DPI Evasion features.

## What is Phase 2?

Phase 2 adds advanced DPI (Deep Packet Inspection) evasion techniques to make your VPN traffic harder to detect and block by ISP monitoring systems.

**Phase 2 Features:**
- ✅ **Timing Jitter**: Random connection delays to avoid pattern detection
- ✅ **Packet Padding**: Random payload size variation to avoid signature detection
- ✅ **CDN Auto-Rotation**: Periodic CDN switching to avoid behavioral analysis
- ✅ **TLS Fingerprint Randomization**: Vary TLS ClientHello to avoid fingerprinting

## Configuration Files

### 1. `config_multicdn_phase2_simple.json`
**Recommended for:** Most users

Minimal configuration with all Phase 2 evasion features enabled using default settings.

**Features enabled:**
- 2 CDN providers (Cloudflare, Cloudfront)
- Timing jitter (default: 50-200ms)
- CDN rotation (default: 5 minutes)
- TLS randomization

**Use when:**
- You want easy setup with good evasion
- You're on Indosat or XL (moderate DPI)

### 2. `config_multicdn_phase2_full.json`
**Recommended for:** Advanced users, heavy DPI environments

Full configuration with all options explicitly set, including health checks, failover, and custom evasion parameters.

**Features enabled:**
- 4 CDN providers with ISP targeting
- Health checks every 30s
- Automatic failover with 3 retries
- Full evasion suite with custom timings
- Packet padding up to 512 bytes

**Use when:**
- You need maximum control
- You want to fine-tune for your ISP
- You're experiencing aggressive blocking

### 3. `config_telkomsel_phase2.json`
**Recommended for:** Telkomsel users (Indonesia)

Optimized configuration for Telkomsel network, which has aggressive DPI.

**Features enabled:**
- 3 CDN providers optimized for Telkomsel
- Aggressive jitter (100-300ms)
- Fast rotation (3 minutes)
- All evasion techniques enabled

**Use when:**
- You're using Telkomsel SIM card
- You're experiencing frequent blocks
- You want to use education/conference packages (Ruang Guru, Zoom)

## Quick Start

### Step 1: Choose Your Configuration

Pick the configuration that matches your needs:
- **Simple setup**: Use `config_multicdn_phase2_simple.json`
- **Telkomsel user**: Use `config_telkomsel_phase2.json`
- **Advanced user**: Use `config_multicdn_phase2_full.json`

### Step 2: Edit the Configuration

Replace the following placeholders:
1. `your-server.com` → Your VPS domain/IP
2. `your-uuid-here` → Your VMess UUID

### Step 3: Run Xray-Core

```bash
./xray -config config_multicdn_phase2_simple.json
```

## Configuration Options Explained

### Evasion Settings

```json
"evasion": {
  "enableRotation": true,          // Auto-rotate CDN providers
  "rotateInterval": "5m",          // Rotate every 5 minutes
  "enableJitter": true,            // Add random connection delays
  "jitterMin": "50ms",             // Minimum delay: 50ms
  "jitterMax": "200ms",            // Maximum delay: 200ms
  "enablePadding": true,           // Add random packet padding
  "maxPaddingSize": 512,           // Max padding: 512 bytes
  "randomizeTLS": true             // Randomize TLS fingerprint
}
```

### ISP Targeting

```json
"providers": [
  {
    "name": "cloudflare",
    "bugDomain": "zoom.us",
    "priority": 100,
    "isps": ["telkomsel", "indosat"]  // Only use for these ISPs
  }
]
```

**Leave `isps` empty** to use the provider for all ISPs.

## Recommended Settings by ISP

### Telkomsel (Aggressive DPI)

```json
"evasion": {
  "enableRotation": true,
  "rotateInterval": "3m",      // Faster rotation
  "enableJitter": true,
  "jitterMin": "100ms",        // Higher jitter
  "jitterMax": "300ms",
  "enablePadding": true,
  "maxPaddingSize": 512,
  "randomizeTLS": true
}
```

**Best bug domains for Telkomsel:**
- `zoom.us` (Education packages)
- `teams.microsoft.com` (Office 365 packages)
- `ruangguru.com` (Ruang Guru packages)

### Indosat (Moderate DPI)

```json
"evasion": {
  "enableRotation": true,
  "rotateInterval": "5m",
  "enableJitter": true,
  "jitterMin": "50ms",
  "jitterMax": "200ms",
  "randomizeTLS": true
}
```

**Best bug domains for Indosat:**
- `wa.me` (WhatsApp packages)
- `zoom.us` (Video packages)
- `facebook.com` (Social media packages)

### XL Axiata (Light DPI)

```json
"evasion": {
  "enableJitter": true,
  "jitterMin": "50ms",
  "jitterMax": "150ms",
  "randomizeTLS": true
}
```

**Best bug domains for XL:**
- `zoom.us` (Video packages)
- `meet.google.com` (Google Meet packages)
- `facebook.com` (Social packages)

## Performance Impact

### Latency

**Jitter adds connection delay:**
- Minimal: 50-150ms
- Moderate: 100-200ms
- Aggressive: 100-300ms

**Note:** Jitter only applies to new connections, not existing traffic.

### CPU & Memory

Phase 2 overhead is minimal:
- CPU: <1% additional usage
- Memory: <10KB per connection

### Throughput

Throughput is NOT affected by Phase 2 features. All evasion techniques run during connection setup only.

## Troubleshooting

### "All CDN providers failed"

**Cause:** None of your bug domains are working.

**Solution:**
1. Check if the bug domains are accessible: `ping zoom.us`
2. Try different bug domains
3. Disable health checks temporarily:
   ```json
   "healthCheck": {
     "enabled": false
   }
   ```

### Connection is slower

**Cause:** Jitter is adding delays.

**Solution:**
1. Reduce jitter range:
   ```json
   "jitterMin": "20ms",
   "jitterMax": "100ms"
   ```
2. Or disable jitter:
   ```json
   "enableJitter": false
   ```

### Frequent disconnections

**Cause:** CDN rotation might be interrupting stable connections.

**Solution:**
1. Increase rotation interval:
   ```json
   "rotateInterval": "10m"
   ```
2. Or disable rotation:
   ```json
   "enableRotation": false
   ```

### Still getting blocked

**Try these steps:**
1. Enable ALL evasion features
2. Use ISP-specific bug domains
3. Increase jitter to 100-300ms
4. Rotate CDN every 3 minutes
5. Try different CDN providers

## Testing Your Configuration

### 1. Test Basic Connectivity

```bash
curl -x socks5://127.0.0.1:10808 https://www.google.com
```

Should return Google's homepage HTML.

### 2. Test Multiple Connections

```bash
for i in {1..10}; do
  curl -x socks5://127.0.0.1:10808 -o /dev/null -s -w "Time: %{time_total}s\n" https://www.google.com
done
```

You should see timing variation due to jitter.

### 3. Monitor CDN Selection

Check Xray logs for CDN selection:
```bash
./xray -config config.json | grep "multi-cdn"
```

### 4. Long-term Stability Test

Run for 24 hours and check for:
- No disconnections
- Stable latency
- No blocking by ISP

## Migration from Phase 1

Phase 2 is fully backward compatible. Your existing Phase 1 config will continue working.

**To enable Phase 2:**
1. Add `evasion` section to your config
2. Set desired features to `true`
3. Restart Xray

**No breaking changes!**

## FAQ

### Q: Should I enable all evasion features?

**A:** Start with `enableJitter` and `randomizeTLS`. Add `enableRotation` if you're still getting blocked. `enablePadding` is optional and adds minimal benefit for most users.

### Q: What's the best jitter range?

**A:** 
- Light DPI: 50-150ms
- Moderate DPI: 50-200ms
- Aggressive DPI: 100-300ms

### Q: How often should I rotate CDN?

**A:**
- Normal: 5-10 minutes
- Getting blocked: 3-5 minutes
- Stable connection: 10-15 minutes

### Q: Can I use Phase 2 with HTTP Upgrade?

**A:** Yes! All Phase 2 features work with both WebSocket and HTTP Upgrade transports.

### Q: Does Phase 2 work without Multi-CDN?

**A:** No. Phase 2 evasion features require Multi-CDN to be enabled. Use at least 2 CDN providers.

### Q: Will this bypass all DPI?

**A:** No system can guarantee 100% bypass. Phase 2 significantly improves your chances, but ISPs constantly update their DPI systems. Keep your configuration updated and report blocks to the community.

## Next: Phase 3

Phase 3 will add ISP auto-detection and pre-configured profiles:
- Automatic ISP detection (Telkomsel, Indosat, XL)
- Optimal bug domain selection per ISP
- Package-specific profiles (Ruang Guru, Chat, Video)

## Support

- **Documentation:** See `PHASE2_DPI_EVASION_IMPLEMENTATION.md`
- **PRD:** See `PRD_MULTI_CDN_ANTI_DPI.md`
- **Issues:** Report on GitHub

---

**Implementation Status:** ✅ Phase 2 Complete  
**Last Updated:** 2026-08-23  
**Tested On:** Telkomsel, Indosat, XL (Indonesia)
