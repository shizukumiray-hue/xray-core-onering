# Changelog: SNI-Based Multi-CDN Implementation

## [2.0.0] - 2026-08-23

### Added

#### New Features
- **SNI-Based Multi-CDN Format**: Users can now configure multiple CDN providers directly in the SNI field using comma-separated format
  - Format: `label1=cdn1.domain,label2=cdn2.domain,server.com`
  - Example: `onering=zoom.us,ruangguru=ruangguru.com,myserver.com`
  - No JSON editing required - works directly in v2rayNG UI

#### Parser Implementation
- New `parseMultiCDNFromSNI()` function in `common/onering/onering.go`
- Automatic label generation for unlabeled CDN entries (`cdn1`, `cdn2`, etc.)
- Automatic priority assignment (100, 90, 80, 70...)
- Automatic MultiCDNManager creation with round-robin strategy
- Space trimming and input validation

#### Format Support
- **With labels**: `onering=zoom.us,ruangguru=ruangguru.com,server.com`
- **Without labels**: `zoom.us,ruangguru.com,server.com`
- **Mixed**: `onering=zoom.us,cloudflare.com,backup=fastly.net,server.com`
- **Minimum**: Two values (1 CDN + 1 server)

### Changed

#### Modified Functions
- `Parse()` in `onering.go`: Added comma detection for new Multi-CDN format
  - Detection order: Comma → `onering-multi:` → `onering:` → Plain domain
  - Maintains full backward compatibility

### Testing

#### New Tests
- `TestParseMultiCDNFromSNI`: 9 test cases covering all format variations
- `TestBackwardCompatibility`: 4 test cases ensuring old formats still work
- All tests passing (100% success rate)

#### Test Coverage
- Multi-CDN with labels ✅
- Multi-CDN without labels ✅
- Multi-CDN mixed format ✅
- Space trimming ✅
- Input validation (empty domains, invalid chars) ✅
- Backward compatibility (old formats) ✅

### Documentation

#### New Documentation Files
- `SNI_MULTICDN_USER_GUIDE.md`: Comprehensive user guide with examples
- `SNI_MULTICDN_IMPLEMENTATION_REPORT.md`: Technical implementation details
- `MULTICDN_SNI_EXAMPLE.txt`: Copy-paste examples for users
- `CHANGELOG_SNI_MULTICDN.md`: This changelog

### Backward Compatibility

#### Supported Formats (All Working)
- ✅ Old single-CDN: `onering:server.com:bug.com`
- ✅ Old multi-CDN: `onering-multi:server.com`
- ✅ New multi-CDN: `onering=bug.com,server.com`
- ✅ Plain domain: `example.com`

### Performance

#### Parser Performance
- O(n) complexity where n = number of CDN entries
- Minimal memory allocation
- No regex (uses simple string operations)
- Early validation (fails fast)

### Security

#### Input Validation
- Rejects control characters (newlines, tabs, etc.)
- Rejects injection characters (`"`, `'`, `<`, `>`)
- Validates each domain separately
- Safe string splitting (no eval/exec)

### Technical Details

#### Modified Files
- `common/onering/onering.go`: +130 lines (parseMultiCDNFromSNI function)
- `common/onering/onering_test.go`: +251 lines (new test cases)
- Total: ~381 lines of new code

#### Integration
- Works with existing TLS config in `transport/internet/tls/config.go`
- No changes needed to TLS integration (automatic detection)
- Uses existing MultiCDNManager and selection strategies

#### Default Settings
- **Strategy**: Round-robin (automatic rotation)
- **Priority**: Auto-assigned (100, 90, 80...)
- **Health Check**: Disabled (optional via JSON)
- **Failover**: 3 retries, 5-minute blacklist
- **ISP Targeting**: All ISPs

### Migration Guide

#### For End Users
**Before (Complex):**
```json
{
  "multicdn": {
    "enabled": true,
    "providers": [
      {"name": "onering", "bugDomain": "zoom.us", "priority": 100}
    ]
  }
}
```

**After (Simple):**
```
onering=zoom.us,server.com
```
Just type in v2rayNG SNI field!

#### For Developers
No migration needed - backward compatible with all existing code.

### Known Limitations

1. Health checks disabled by default for SNI-based config (can enable via JSON)
2. Strategy fixed to round-robin (can change via JSON)
3. No ISP-specific routing in SNI format (can add via JSON)

### Future Enhancements (Not Implemented)

Possible future extensions:
- Strategy in SNI: `[rr]onering=zoom.us,server.com`
- Priority override: `onering=zoom.us:100,backup:50,server.com`
- ISP targeting: `onering=zoom.us@telkomsel,server.com`
- Health check flag: `[hc]onering=zoom.us,server.com`

### Verification

#### Build Status
- ✅ All packages compile successfully
- ✅ No breaking changes
- ✅ All tests passing (0 failures)

#### Test Results
```
PASS: TestParse (7/7 tests)
PASS: TestParseMultiCDNFromSNI (9/9 tests)
PASS: TestBackwardCompatibility (4/4 tests)
PASS: All onering tests (15+ tests, 0.899s)
```

### Credits

- Implementation: Kiro (AI Agent)
- Date: 2026-08-23
- Repository: xray-core-onering
- Module: common/onering

### References

- User Guide: `SNI_MULTICDN_USER_GUIDE.md`
- Technical Report: `SNI_MULTICDN_IMPLEMENTATION_REPORT.md`
- Examples: `MULTICDN_SNI_EXAMPLE.txt`
