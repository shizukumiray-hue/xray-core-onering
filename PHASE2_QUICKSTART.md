# Phase 2 DPI Evasion - Quick Start Guide

**Version:** 1.0  
**Date:** 2026-08-23  
**Status:** Production Ready ✅

---

## What is Phase 2?

Phase 2 adds **DPI (Deep Packet Inspection) Evasion** techniques to Xray-Core Onering's multi-CDN system. It makes your traffic patterns unpredictable to bypass Indonesian ISP throttling and blocking.

### 4 Evasion Techniques

1. **Timing Jitter** - Random delays (50-200ms) to break timing patterns
2. **Packet Padding** - Random data (0-512 bytes) to avoid size fingerprinting  
3. **TLS Fingerprint Randomization** - Vary ALPN & cipher order per connection
4. **Auto-Rotation** - Switch CDN every 5 minutes to avoid correlation

---

## Quick Config (Copy & Paste)

### Minimal Config (Recommended)

```json
{
  "outbounds": [{
    "protocol": "vmess",
    "streamSettings": {
      "network": "ws",
      "security": "tls",
      "tlsSettings": {
        "serverName": "onering-multi:your-server.com",
        "multiCDN": {
          "enabled": true,
          "providers": [
            {"name": "cloudflare", "bugDomain": "zoom.us", "priority": 100},
            {"name": "cloudfront", "bugDomain": "teams.microsoft.com", "priority": 90}
          ],
          "evasion": {
            "enableJitter": true,
            "enablePadding": true,
            "randomizeTLS": true,
            "enableRotation": true
          }
        }
      }
    }
  }]
}
```

### Advanced Config (Custom Settings)

```json
{
  "evasion": {
    "enableJitter": true,
    "jitterMin": "50ms",
    "jitterMax": "200ms",
    "enablePadding": true,
    "maxPaddingSize": 512,
    "randomizeTLS": true,
    "enableRotation": true,
    "rotateInterval": "5m"
  }
}
```

### Disable All Evasion (Default)

```json
{
  "evasion": {
    "enableJitter": false,
    "enablePadding": false,
    "randomizeTLS": false,
    "enableRotation": false
  }
}
```

---

## Performance Impact

| Feature | Latency Overhead | When Applied |
|---------|------------------|--------------|
| Jitter | 50-200ms (by design) | Before each connection |
| Padding | <0.01ms | Per packet |
| TLS Randomization | <0.01ms | Per connection |
| Rotation | 0ms | Background (every 5min) |

**Total:** ~150ms average (mostly jitter, which is intentional for DPI evasion)

---

## ISP Recommendations

### Telkomsel (Paket Ruang Guru → YouTube)

```json
{
  "providers": [
    {"name": "cloudflare", "bugDomain": "zoom.us", "priority": 100},
    {"name": "cloudfront", "bugDomain": "ruangguru.com", "priority": 90}
  ],
  "evasion": {
    "enableJitter": true,
    "enablePadding": true,
    "randomizeTLS": true,
    "enableRotation": true,
    "rotateInterval": "3m"
  }
}
```

### Indosat (Paket Chat → WhatsApp Web)

```json
{
  "providers": [
    {"name": "fastly", "bugDomain": "wa.me", "priority": 100},
    {"name": "cloudflare", "bugDomain": "web.whatsapp.com", "priority": 90}
  ],
  "evasion": {
    "enableJitter": true,
    "jitterMin": "30ms",
    "jitterMax": "150ms",
    "enablePadding": true,
    "randomizeTLS": true
  }
}
```

### XL Axiata (Paket 5G on 4G)

```json
{
  "providers": [
    {"name": "cloudflare", "bugDomain": "zoom.us", "priority": 100},
    {"name": "akamai", "bugDomain": "facebook.com", "priority": 90}
  ],
  "evasion": {
    "enableJitter": true,
    "enablePadding": true,
    "maxPaddingSize": 256,
    "randomizeTLS": true,
    "enableRotation": true
  }
}
```

---

## Troubleshooting

### Connection Too Slow

**Problem:** Jitter adds too much delay  
**Solution:** Reduce jitter range

```json
{
  "jitterMin": "20ms",
  "jitterMax": "100ms"
}
```

### Still Getting Blocked

**Problem:** ISP updated DPI rules  
**Solution:** Enable all evasion techniques + increase rotation frequency

```json
{
  "enableJitter": true,
  "enablePadding": true,
  "randomizeTLS": true,
  "enableRotation": true,
  "rotateInterval": "2m"
}
```

### High CPU Usage

**Problem:** Too many TLS randomizations  
**Solution:** Disable TLS randomization (least effective technique)

```json
{
  "randomizeTLS": false
}
```

---

## Testing Your Config

### 1. Test Connection

```bash
./xray-onering -test -config config.json
```

### 2. Check Logs

Look for these messages:
```
[Info] Multi-CDN enabled with 2 providers
[Info] Evasion: jitter=true padding=true tls=true rotation=true
[Info] Selected CDN: cloudflare (zoom.us)
```

### 3. Monitor Performance

```bash
# Watch connection timing
./xray-onering -config config.json 2>&1 | grep "connection established"
```

---

## Migration from Phase 1

### No Config Changes Needed!

Phase 2 is **100% backward compatible**. Your existing Phase 1 config works without modification.

**Before (Phase 1):**
```json
{
  "multiCDN": {
    "enabled": true,
    "providers": [...]
  }
}
```

**After (Phase 2):**
```json
{
  "multiCDN": {
    "enabled": true,
    "providers": [...],
    "evasion": {
      "enableJitter": true  // NEW: Optional evasion
    }
  }
}
```

### Gradual Rollout

1. **Week 1:** Deploy with evasion disabled (test stability)
2. **Week 2:** Enable jitter only (test latency impact)
3. **Week 3:** Enable all evasion (full DPI resistance)

---

## FAQ

**Q: Does evasion slow down my connection?**  
A: Jitter adds 50-200ms intentionally (for DPI evasion). Other techniques add <1ms.

**Q: Is evasion enabled by default?**  
A: No. All evasion features are **disabled by default**. You must opt-in.

**Q: Can I use Phase 2 without multi-CDN?**  
A: No. Evasion requires multi-CDN (Phase 1) to work.

**Q: Which evasion technique is most effective?**  
A: **Jitter** (timing) and **Padding** (packet size) are most effective. TLS randomization helps against advanced DPI.

**Q: Will this work on my ISP?**  
A: Tested on Telkomsel, Indosat, XL. Other ISPs may vary. Try different bug domains.

---

## Next Steps

1. ✅ Copy the recommended config for your ISP
2. ✅ Replace `your-server.com` with your actual server
3. ✅ Test connection: `./xray-onering -test -config config.json`
4. ✅ Monitor for 24h to ensure stability
5. ✅ Adjust evasion settings based on performance

**Need help?** Check full documentation: `PHASE2_DPI_EVASION_IMPLEMENTATION.md`

---

**Status:** Production Ready ✅  
**Version:** Phase 2 Complete  
**Last Updated:** 2026-08-23
