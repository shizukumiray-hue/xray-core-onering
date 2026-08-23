# SNI-Based Multi-CDN Configuration Guide

## Overview

Onering now supports a simplified Multi-CDN configuration format directly in the SNI (Server Name Indication) field. Users can specify multiple CDN providers using a comma-separated format without needing to edit JSON configuration files.

## Format Specification

### Comma-Separated Multi-CDN Format

```
label1=cdn1.domain,label2=cdn2.domain,label3=cdn3.domain,real-server.com
```

**Key Points:**
- Multiple CDNs separated by comma `,`
- Each CDN entry: `label=domain` or just `domain`
- Label is optional (e.g., `onering=`, `ruangguru=`, `zenius=`)
- **Last value** after comma is the **actual server host** (real domain)
- Spaces after commas are automatically trimmed

## Examples

### Example 1: Multi-CDN with Labels

```
onering=bug-zoom.us,ruangguru=ruangguru.com,zenius=zenius.net,myserver.example.com
```

**Parsed as:**
- CDN 1: `bug-zoom.us` (label: `onering`, priority: 100)
- CDN 2: `ruangguru.com` (label: `ruangguru`, priority: 90)
- CDN 3: `zenius.net` (label: `zenius`, priority: 80)
- Real Server: `myserver.example.com`

### Example 2: Multi-CDN without Labels

```
zoom.us,cloudflare.com,fastly.net,myserver.example.com
```

**Parsed as:**
- CDN 1: `zoom.us` (auto-label: `cdn1`, priority: 100)
- CDN 2: `cloudflare.com` (auto-label: `cdn2`, priority: 90)
- CDN 3: `fastly.net` (auto-label: `cdn3`, priority: 80)
- Real Server: `myserver.example.com`

### Example 3: Mixed Format (With and Without Labels)

```
onering=zoom.us,cloudflare.com,zenius=zenius.net,myserver.example.com
```

**Parsed as:**
- CDN 1: `zoom.us` (label: `onering`, priority: 100)
- CDN 2: `cloudflare.com` (auto-label: `cdn2`, priority: 90)
- CDN 3: `zenius.net` (label: `zenius`, priority: 80)
- Real Server: `myserver.example.com`

### Example 4: Simple Two-CDN Setup

```
zoom.us,myserver.example.com
```

**Parsed as:**
- CDN 1: `zoom.us` (auto-label: `cdn1`, priority: 100)
- Real Server: `myserver.example.com`

## How to Use in v2rayNG

1. Open v2rayNG app
2. Select or create a server configuration
3. Go to **TLS Settings** or **Transport Settings**
4. Find the **SNI** or **Server Name** field
5. Enter your Multi-CDN format directly:
   ```
   onering=zoom.us,ruangguru=ruangguru.com,myserver.example.com
   ```
6. Save the configuration
7. Connect

**That's it!** No JSON editing required.

## Backward Compatibility

The new format is fully backward compatible with existing formats:

### Old Format 1: Single-CDN (Colon-Separated)
```
onering:myserver.example.com:zoom.us
```
Still works as before.

### Old Format 2: Multi-CDN Prefix
```
onering-multi:myserver.example.com
```
Still works, but requires separate CDN provider configuration.

### Old Format 3: Plain Domain (No Onering)
```
myserver.example.com
```
Still works as a regular connection without Onering.

### New Format: Comma-Separated Multi-CDN
```
onering=zoom.us,ruangguru=ruangguru.com,myserver.example.com
```
New simplified format with automatic Multi-CDN setup.

## How It Works

1. **Parsing:** When you enter a comma-separated value in the SNI field, xray-core automatically detects it as Multi-CDN format
2. **CDN Providers:** Each CDN entry becomes a CDN provider with:
   - Name/Label (from `label=` or auto-generated)
   - Bug Domain (the CDN domain to connect to)
   - Priority (automatically assigned: first=100, second=90, third=80, etc.)
3. **Selection Strategy:** Round-robin by default (rotates through CDNs)
4. **Connection Flow:**
   - Connects to selected CDN bug domain (e.g., `zoom.us`)
   - Sends Host header with real server domain (e.g., `myserver.example.com`)
   - TLS SNI uses CDN bug domain for handshake

## Selection Strategy

By default, Multi-CDN uses **Round-Robin** strategy:
- First connection → CDN 1
- Second connection → CDN 2
- Third connection → CDN 3
- Fourth connection → CDN 1 (cycles back)

This distributes traffic across all configured CDNs.

## Priority System

CDNs are auto-prioritized based on their position:
- 1st CDN: Priority 100 (highest)
- 2nd CDN: Priority 90
- 3rd CDN: Priority 80
- 4th CDN: Priority 70
- And so on...

Higher priority CDNs are preferred when using priority-based strategies.

## Common Use Cases

### Use Case 1: Indonesia ISP Optimization

Different CDNs work better with different ISPs:

```
telkomsel=zoom.us,indosat=wa.me,xl=ruangguru.com,myserver.example.com
```

### Use Case 2: Failover Configuration

Primary CDN with backup:

```
primary=cloudflare.com,backup=fastly.net,myserver.example.com
```

### Use Case 3: Load Distribution

Distribute load across multiple CDNs:

```
cdn-1=zoom.us,cdn-2=aws.amazon.com,cdn-3=discord.com,myserver.example.com
```

## Validation Rules

The parser validates:
- ✅ At least 2 comma-separated values (minimum 1 CDN + 1 server)
- ✅ No empty domains
- ✅ No invalid characters (newlines, control chars, etc.)
- ✅ Proper domain format

Invalid examples:
- ❌ `zoom.us` (no comma, treated as plain domain)
- ❌ `zoom.us,` (empty server domain)
- ❌ `onering=,server.com` (empty CDN domain)
- ❌ `zoom.us\n,server.com` (contains newline)

## Troubleshooting

### Problem: Connection fails with Multi-CDN

**Solution 1:** Check CDN domains are correct
- Test each CDN domain individually first
- Example: Try `zoom.us,server.com` with just one CDN

**Solution 2:** Try without labels
- Instead of: `onering=zoom.us,server.com`
- Try: `zoom.us,server.com`

**Solution 3:** Fall back to old format
- Use old single-CDN format: `onering:server.com:zoom.us`

### Problem: SNI field not accepting commas

**Solution:** Ensure you're using xray-core-onering (not standard xray-core)
- This feature is specific to the Onering fork
- Check you're using the correct v2rayNG build with Onering support

### Problem: CDN selection not rotating

**Check:** Multi-CDN is enabled properly
- Comma-separated format should auto-enable Multi-CDN
- View logs to confirm CDN selection

## Advanced Configuration

For advanced users who need more control (health checks, custom strategies, etc.), use the old `onering-multi:` format with JSON configuration in the config file.

The SNI-based format is optimized for simplicity and ease of use.

## Technical Details

### Parser Logic

```go
// Detection order:
1. Contains comma? → Parse as Multi-CDN SNI format
2. Starts with "onering-multi:"? → Parse as old Multi-CDN format
3. Starts with "onering:"? → Parse as single-CDN format
4. Otherwise → Plain domain (no Onering)
```

### Config Creation

The parser automatically creates:
- `MultiCDNConfig` with all providers
- `MultiCDNManager` with round-robin strategy
- Default failover settings (3 retries, 5-minute blacklist)
- Health checks disabled by default (can be enabled in JSON config)

## Summary

The new SNI-based Multi-CDN format provides:
- ✅ Simple comma-separated syntax
- ✅ No JSON editing required
- ✅ Direct input in v2rayNG UI
- ✅ Automatic CDN rotation
- ✅ Full backward compatibility
- ✅ Flexible label assignment
- ✅ Automatic priority assignment

**Recommended for:** Most users who want simple Multi-CDN setup

**Use JSON config for:** Advanced users needing health checks, custom strategies, or ISP-specific routing
