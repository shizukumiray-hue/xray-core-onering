# Xray-Core Onering Multi-CDN - CI/CD Documentation

## Overview

This document describes the complete CI/CD setup for Xray-Core Onering with Multi-CDN anti-DPI bypass features.

## Workflows Summary

| Workflow | Purpose | Trigger | Duration |
|----------|---------|---------|----------|
| **build-multicdn.yml** | Build & test all platforms | Push, PR, Manual | ~15 min |
| **release-multicdn.yml** | Create releases | Tags, Manual | ~45 min |
| **docker-multicdn.yml** | Build Docker images | Push, Tags, Manual | ~20 min |
| **test-multicdn.yml** | Comprehensive testing | Push, PR, Schedule | ~30 min |

## Quick Start

### 1. Enable GitHub Actions

1. Go to repository Settings → Actions → General
2. Enable "Allow all actions and reusable workflows"
3. Set workflow permissions to "Read and write permissions"

### 2. First Build

```bash
# Clone repository
git clone https://github.com/YOUR_USERNAME/xray-core-onering.git
cd xray-core-onering

# Push to trigger CI
git push origin main
```

### 3. View Build Results

```bash
# Using GitHub CLI
gh run list --workflow=build-multicdn.yml
gh run view --log

# Or visit: https://github.com/YOUR_USERNAME/xray-core-onering/actions
```

## Detailed Workflow Guide

### Build Workflow (build-multicdn.yml)

**Purpose**: Continuous integration for every code change

**Jobs**:
1. **Lint** (2 min)
   - Go formatting check
   - Go vet
   - Staticcheck

2. **Test** (5 min)
   - Unit tests on Linux, macOS, Windows
   - Race condition detection
   - Code coverage reporting

3. **Build Multi-Platform** (8 min)
   - Cross-compile for 30+ platforms
   - Upload artifacts (7 day retention)

4. **Build Android AAR** (3 min)
   - Android library with gomobile
   - Supports arm64 and amd64

5. **Integration Test** (2 min)
   - Test binary execution
   - Config validation

**Artifacts Generated**:
```
xray-linux-64/
xray-linux-arm64-v8a/
xray-android-arm64-v8a/
xray-windows-64/
xray-macos-arm64-v8a/
... (30+ variants)
```

### Release Workflow (release-multicdn.yml)

**Purpose**: Automated release creation and distribution

**Trigger Methods**:

**Method 1: Git Tags**
```bash
# Create tag
git tag -a v1.0.0 -m "Multi-CDN Release v1.0.0"
git push origin v1.0.0
```

**Method 2: Manual Dispatch**
```bash
gh workflow run release-multicdn.yml \
  -f tag=v1.0.0 \
  -f prerelease=false
```

**Release Process**:
1. Create GitHub Release (draft)
2. Build all platform binaries
3. Create ZIP archives with:
   - Binary
   - README.md
   - LICENSE
   - Sample Multi-CDN config
   - Windows helper scripts
4. Generate SHA256 checksums
5. Upload to GitHub Release
6. Publish release

**Release Assets**:
```
Xray-Onering-MultiCDN-linux-64.zip
Xray-Onering-MultiCDN-linux-64.zip.sha256
Xray-Onering-MultiCDN-android-arm64-v8a.zip
Xray-Onering-MultiCDN-windows-64.zip
Xray-Onering-MultiCDN-macos-arm64-v8a.zip
... (60+ files: 30 platforms × 2)
```

### Docker Workflow (docker-multicdn.yml)

**Purpose**: Build and publish multi-architecture Docker images

**Supported Architectures**:
- linux/amd64
- linux/arm64
- linux/arm/v7
- linux/arm/v6
- linux/ppc64le
- linux/s390x
- linux/riscv64

**Image Tags**:
- `latest` (on main branch)
- `develop` (on develop branch)
- `v1.0.0` (on version tags)
- `1.0` (semver major.minor)
- `1` (semver major)

**Usage**:
```bash
# Pull latest
docker pull ghcr.io/YOUR_USERNAME/xray-core-onering:latest

# Pull specific version
docker pull ghcr.io/YOUR_USERNAME/xray-core-onering:v1.0.0

# Run
docker run -d -p 10808:10808 \
  -v $(pwd)/config.json:/etc/xray/config.json \
  ghcr.io/YOUR_USERNAME/xray-core-onering:latest
```

### Test Workflow (test-multicdn.yml)

**Purpose**: Comprehensive testing suite

**Test Types**:

1. **Unit Tests**
   - Multi-CDN core components
   - Transport layer
   - Code coverage >80%

2. **Integration Tests**
   - Binary execution on all OS
   - Config validation
   - End-to-end flows

3. **CDN Provider Tests**
   - Cloudflare
   - Cloudfront
   - Fastly
   - Akamai
   - GCore

4. **Strategy Tests**
   - Round-robin
   - Failover
   - Latency-based
   - Health-based
   - Random

5. **ISP Profile Tests**
   - Telkomsel profile
   - Indosat profile
   - XL profile

6. **Performance Tests**
   - Benchmark CDN selection
   - Memory profiling
   - Latency measurement

7. **Stress Tests** (scheduled)
   - 1-hour continuous operation
   - Memory leak detection
   - Failover reliability

**Schedule**: Daily at 00:00 UTC for stress tests

## Configuration

### GitHub Repository Settings

**1. Secrets** (Settings → Secrets and variables → Actions)

No additional secrets required. `GITHUB_TOKEN` is auto-provided.

**2. Permissions** (Settings → Actions → General)

Enable:
- ✅ Read and write permissions
- ✅ Allow GitHub Actions to create and approve pull requests

**3. Environments** (Optional)

Create environments for staged releases:
- `production` (requires approval)
- `staging` (auto-deploy)

### Workflow Variables

Edit in workflow files:

```yaml
env:
  GO_VERSION: '1.26'  # Go version
  REGISTRY: ghcr.io   # Docker registry
```

### Build Matrix Customization

**Add new platform**:
```yaml
# In build-multicdn.yml or release-multicdn.yml
matrix:
  include:
    - goos: your_os
      goarch: your_arch
      name: friendly-name
```

**Remove platform**:
Comment out or delete the matrix entry.

## Badge Integration

Add to README.md:

```markdown
[![Build](https://github.com/YOUR_USERNAME/xray-core-onering/actions/workflows/build-multicdn.yml/badge.svg)](https://github.com/YOUR_USERNAME/xray-core-onering/actions/workflows/build-multicdn.yml)

[![Release](https://github.com/YOUR_USERNAME/xray-core-onering/actions/workflows/release-multicdn.yml/badge.svg)](https://github.com/YOUR_USERNAME/xray-core-onering/actions/workflows/release-multicdn.yml)

[![Docker](https://github.com/YOUR_USERNAME/xray-core-onering/actions/workflows/docker-multicdn.yml/badge.svg)](https://github.com/YOUR_USERNAME/xray-core-onering/actions/workflows/docker-multicdn.yml)

[![Tests](https://github.com/YOUR_USERNAME/xray-core-onering/actions/workflows/test-multicdn.yml/badge.svg)](https://github.com/YOUR_USERNAME/xray-core-onering/actions/workflows/test-multicdn.yml)
```

## Common Tasks

### Download Build Artifacts

```bash
# List recent runs
gh run list --workflow=build-multicdn.yml --limit 5

# Download from specific run
gh run download 123456789

# Download specific artifact
gh run download 123456789 -n xray-linux-64
```

### Create a Release

```bash
# Method 1: Git tag (triggers automatically)
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0

# Method 2: Manual workflow dispatch
gh workflow run release-multicdn.yml \
  -f tag=v1.0.0 \
  -f prerelease=false

# View release
gh release view v1.0.0

# Download release assets
gh release download v1.0.0
```

### Build Docker Image

```bash
# Trigger manually
gh workflow run docker-multicdn.yml -f tag=custom-tag

# Check build status
gh run list --workflow=docker-multicdn.yml

# View logs
gh run view --log
```

### Run Tests Manually

```bash
# Trigger test workflow
gh workflow run test-multicdn.yml

# Watch test progress
gh run watch

# View test results
gh run view --log
```

## Troubleshooting

### Build Failures

**Issue**: Go version mismatch
```bash
# Fix: Update go.mod
go mod edit -go=1.26
go mod tidy
git commit -am "Update Go version"
```

**Issue**: Missing dependencies
```bash
# Fix: Update dependencies
go mod download
go mod tidy
git commit -am "Update dependencies"
```

**Issue**: Test failures
```bash
# Run tests locally
go test ./...

# Run specific test
go test -v -run TestMultiCDN ./common/onering/...
```

### Release Issues

**Issue**: Tag already exists
```bash
# Delete tag
git tag -d v1.0.0
git push origin :refs/tags/v1.0.0

# Create new tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

**Issue**: Assets not uploaded
- Check workflow permissions (write access required)
- Check if build jobs completed successfully
- View workflow logs for errors

### Docker Issues

**Issue**: Multi-arch build fails
```bash
# Check QEMU setup
docker run --rm --privileged multiarch/qemu-user-static --reset -p yes

# Test local build
docker buildx build --platform linux/amd64,linux/arm64 -t test .
```

**Issue**: Image not found
```bash
# Check image exists
docker pull ghcr.io/YOUR_USERNAME/xray-core-onering:latest

# Login to registry
echo $GITHUB_TOKEN | docker login ghcr.io -u YOUR_USERNAME --password-stdin
```

## Performance Optimization

### Build Speed

**1. Enable caching**:
```yaml
- uses: actions/cache@v5
  with:
    path: |
      ~/.cache/go-build
      ~/go/pkg/mod
    key: ${{ runner.os }}-go-${{ hashFiles('**/go.sum') }}
```

**2. Use build matrix**:
Parallel builds complete in ~8 minutes vs ~2 hours sequential.

**3. Reduce artifact retention**:
```yaml
- uses: actions/upload-artifact@v7
  with:
    retention-days: 3  # Default: 7
```

### Cost Optimization

**Free tier limits** (GitHub Actions):
- 2,000 minutes/month for private repos
- Unlimited for public repos
- 500 MB artifact storage

**Tips**:
- Use artifact retention wisely
- Enable branch protection (run CI only on PRs)
- Use self-hosted runners for heavy builds

## Security Best Practices

### 1. Workflow Permissions

Use minimal permissions:
```yaml
permissions:
  contents: read    # Read repo
  packages: write   # Push Docker images
```

### 2. Secret Management

Never commit secrets:
```bash
# Use GitHub Secrets
gh secret set MY_SECRET
```

### 3. Dependency Scanning

Add Dependabot:
```yaml
# .github/dependabot.yml
version: 2
updates:
  - package-ecosystem: gomod
    directory: /
    schedule:
      interval: weekly
```

### 4. Code Scanning

Add CodeQL:
```yaml
# .github/workflows/codeql.yml
- uses: github/codeql-action/init@v3
  with:
    languages: go
```

## Monitoring

### Workflow Notifications

**Slack integration**:
```yaml
- name: Notify Slack
  if: failure()
  uses: 8398a7/action-slack@v3
  with:
    status: ${{ job.status }}
    webhook_url: ${{ secrets.SLACK_WEBHOOK }}
```

**Email notifications**:
Settings → Notifications → Actions

### Metrics

Track via GitHub API:
```bash
# Workflow runs
gh api repos/YOUR_USERNAME/xray-core-onering/actions/runs

# Success rate
gh api repos/YOUR_USERNAME/xray-core-onering/actions/workflows/build-multicdn.yml/runs \
  --jq '.workflow_runs | map(select(.conclusion=="success")) | length'
```

## Advanced Usage

### Custom Runners

**Self-hosted runner** (for Android builds):
```bash
# Download runner
mkdir actions-runner && cd actions-runner
curl -o actions-runner-linux-x64-2.311.0.tar.gz -L \
  https://github.com/actions/runner/releases/download/v2.311.0/actions-runner-linux-x64-2.311.0.tar.gz
tar xzf ./actions-runner-linux-x64-2.311.0.tar.gz

# Configure
./config.sh --url https://github.com/YOUR_USERNAME/xray-core-onering --token YOUR_TOKEN

# Run
./run.sh
```

### Reusable Workflows

Create `.github/workflows/reusable-build.yml`:
```yaml
on:
  workflow_call:
    inputs:
      platform:
        required: true
        type: string

jobs:
  build:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v6
      - name: Build ${{ inputs.platform }}
        run: go build -o xray-${{ inputs.platform }} ./main
```

Use in other workflows:
```yaml
jobs:
  linux:
    uses: ./.github/workflows/reusable-build.yml
    with:
      platform: linux-64
```

## Support

- GitHub Issues: https://github.com/YOUR_USERNAME/xray-core-onering/issues
- GitHub Discussions: https://github.com/YOUR_USERNAME/xray-core-onering/discussions
- Documentation: https://github.com/YOUR_USERNAME/xray-core-onering/tree/main/.github

---

**Last Updated**: 2026-08-23  
**Version**: 1.0.0
