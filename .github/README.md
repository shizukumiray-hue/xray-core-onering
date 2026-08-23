# GitHub Actions CI/CD for Xray-Core Onering Multi-CDN

This directory contains GitHub Actions workflows for building, testing, and releasing Xray-Core Onering with Multi-CDN support.

## 📋 Workflows

### 1. **build-multicdn.yml** - Main Build & Test Pipeline

**Triggers:**
- Push to `main`, `develop`, `feature/**`, `multicdn/**` branches
- Pull requests
- Manual dispatch

**Jobs:**
- **Lint**: Code quality checks (gofmt, go vet, staticcheck)
- **Test**: Unit tests on Linux, macOS, Windows with code coverage
- **Build Multi-Platform**: Cross-compile for all supported platforms
- **Build Android AAR**: Android library for integration
- **Integration Test**: End-to-end testing with real binaries
- **Summary**: Build status report

**Artifacts:** Binaries for all platforms (retained for 7 days)

### 2. **release-multicdn.yml** - Release Automation

**Triggers:**
- Git tags: `v*`, `multicdn-v*`
- Manual dispatch with custom tag

**Features:**
- Automatic GitHub Release creation
- Multi-platform binary builds (30+ architectures)
- Android AAR library
- SHA256 & MD5 checksums
- Sample Multi-CDN configuration included
- Windows helper scripts (no-window variants)

**Release Assets:**
```
Xray-Onering-MultiCDN-linux-64.zip
Xray-Onering-MultiCDN-android-arm64-v8a.zip
Xray-Onering-MultiCDN-windows-64.zip
Xray-Onering-MultiCDN-macos-arm64-v8a.zip
... (30+ platform variants)
```

### 3. **docker-multicdn.yml** - Docker Build & Push

**Triggers:**
- Push to `main`, `develop`
- Git tags: `v*`, `multicdn-v*`
- Pull requests (build only, no push)
- Manual dispatch

**Features:**
- Multi-architecture Docker images
- Push to GitHub Container Registry (`ghcr.io`)
- Automatic tagging (latest, semver, branch)
- Image testing with health checks

**Supported Architectures:**
- linux/amd64
- linux/arm64
- linux/arm/v7
- linux/arm/v6
- linux/ppc64le
- linux/s390x
- linux/riscv64

## 🚀 Usage

### Trigger a Build

**On code push:**
```bash
git push origin main
```

**Manual workflow dispatch:**
```bash
gh workflow run build-multicdn.yml
```

### Create a Release

**Using git tags:**
```bash
git tag -a v1.0.0 -m "Multi-CDN Release v1.0.0"
git push origin v1.0.0
```

**Manual release:**
```bash
gh workflow run release-multicdn.yml -f tag=v1.0.0 -f prerelease=false
```

### Build Docker Image

**Automatic (on push to main):**
```bash
git push origin main
```

**Manual with custom tag:**
```bash
gh workflow run docker-multicdn.yml -f tag=latest
```

**Pull the image:**
```bash
docker pull ghcr.io/YOUR_USERNAME/xray-core-onering:latest
```

## 📦 Build Artifacts

### Download from GitHub Actions

**Using GitHub CLI:**
```bash
# List recent workflow runs
gh run list --workflow=build-multicdn.yml --limit 5

# Download artifacts from a specific run
gh run download RUN_ID
```

**Using web interface:**
1. Go to Actions tab
2. Select workflow run
3. Download artifacts from "Artifacts" section

### Download from Releases

**Using GitHub CLI:**
```bash
gh release download v1.0.0
```

**Using wget:**
```bash
wget https://github.com/YOUR_USERNAME/xray-core-onering/releases/download/v1.0.0/Xray-Onering-MultiCDN-linux-64.zip
```

## 🔧 Configuration

### Environment Variables

Set these in GitHub repository settings → Secrets and variables → Actions:

- `GITHUB_TOKEN`: Automatically provided by GitHub Actions
- (Optional) Custom secrets for external services

### Workflow Customization

**Change Go version:**
Edit the `GO_VERSION` env variable in each workflow:
```yaml
env:
  GO_VERSION: '1.26'
```

**Add new build platforms:**
Edit the matrix in `build-multicdn.yml`:
```yaml
matrix:
  include:
    - goos: yourOS
      goarch: yourArch
      name: friendly-name
```

**Modify Docker platforms:**
Edit `docker-multicdn.yml`:
```yaml
platforms: |
  linux/amd64
  linux/arm64
  your/platform
```

## 📊 Build Status Badges

Add these to your README.md:

```markdown
[![Build](https://github.com/YOUR_USERNAME/xray-core-onering/actions/workflows/build-multicdn.yml/badge.svg)](https://github.com/YOUR_USERNAME/xray-core-onering/actions/workflows/build-multicdn.yml)
[![Release](https://github.com/YOUR_USERNAME/xray-core-onering/actions/workflows/release-multicdn.yml/badge.svg)](https://github.com/YOUR_USERNAME/xray-core-onering/actions/workflows/release-multicdn.yml)
[![Docker](https://github.com/YOUR_USERNAME/xray-core-onering/actions/workflows/docker-multicdn.yml/badge.svg)](https://github.com/YOUR_USERNAME/xray-core-onering/actions/workflows/docker-multicdn.yml)
```

## 🐳 Docker Usage

### Pull and Run

```bash
# Pull latest image
docker pull ghcr.io/YOUR_USERNAME/xray-core-onering:latest

# Run with custom config
docker run -d \
  --name xray-multicdn \
  -p 10808:10808 \
  -v $(pwd)/config.json:/etc/xray/config.json \
  ghcr.io/YOUR_USERNAME/xray-core-onering:latest

# View logs
docker logs -f xray-multicdn

# Stop container
docker stop xray-multicdn
```

### Docker Compose

Create `docker-compose.yml`:
```yaml
version: '3.8'

services:
  xray:
    image: ghcr.io/YOUR_USERNAME/xray-core-onering:latest
    container_name: xray-multicdn
    restart: unless-stopped
    ports:
      - "10808:10808"
    volumes:
      - ./config.json:/etc/xray/config.json
      - ./logs:/var/log/xray
    environment:
      - TZ=Asia/Jakarta
```

Run:
```bash
docker-compose up -d
```

## 🧪 Testing

### Run Tests Locally

```bash
# Run all tests
go test ./...

# Run with coverage
go test -coverprofile=coverage.out ./...
go tool cover -html=coverage.out

# Run specific package tests
go test -v ./common/onering/...
```

### Integration Tests

```bash
# Build binary first
go build -o xray ./main

# Run with test config
./xray run -test -config test-config.json
```

## 🔍 Troubleshooting

### Build Failures

**Go version mismatch:**
```bash
# Check required Go version
cat go.mod | grep "^go "

# Update workflow if needed
```

**Missing dependencies:**
```bash
# Ensure go.sum is up to date
go mod tidy
git add go.sum
git commit -m "Update dependencies"
```

### Docker Build Issues

**Platform not supported:**
- Check if QEMU supports the architecture
- Remove unsupported platforms from workflow

**Image too large:**
- Use multi-stage builds (already implemented)
- Add `.dockerignore` file

### Release Issues

**Missing assets:**
- Check if build jobs completed successfully
- Verify upload_url is correctly passed between jobs

**Tag already exists:**
```bash
# Delete tag locally and remotely
git tag -d v1.0.0
git push origin :refs/tags/v1.0.0

# Create new tag
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```

## 📝 Development Workflow

### Feature Development

1. Create feature branch:
```bash
git checkout -b feature/my-feature
```

2. Make changes and commit:
```bash
git add .
git commit -m "feat: add new feature"
git push origin feature/my-feature
```

3. CI automatically runs on push
4. Create PR when ready
5. Merge after CI passes

### Release Process

1. Update version in code
2. Update CHANGELOG.md
3. Create and push tag:
```bash
git tag -a v1.0.0 -m "Release v1.0.0"
git push origin v1.0.0
```
4. Release workflow automatically creates GitHub Release
5. Docker images automatically built and pushed

## 🔐 Security

### Secrets Management

- Never commit secrets to repository
- Use GitHub Secrets for sensitive data
- Docker images use non-root user
- Health checks enabled for monitoring

### Dependency Scanning

Consider adding:
- Dependabot for dependency updates
- CodeQL for security scanning
- OSSF Scorecard for security metrics

## 📚 Additional Resources

- [GitHub Actions Documentation](https://docs.github.com/actions)
- [Docker Multi-Platform Builds](https://docs.docker.com/build/building/multi-platform/)
- [Go Cross Compilation](https://go.dev/doc/install/source#environment)
- [Xray-Core Documentation](https://xtls.github.io/)
- [Multi-CDN PRD](../../PRD_MULTI_CDN_ANTI_DPI.md)

## 🤝 Contributing

1. Fork the repository
2. Create feature branch
3. Make changes
4. Ensure CI passes
5. Submit pull request

## 📄 License

See [LICENSE](../../LICENSE) file for details.

---

**Note:** Replace `YOUR_USERNAME` with your actual GitHub username throughout this document.
