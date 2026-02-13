# Cache Troubleshooting Guide

Complete guide to understanding, troubleshooting, and managing goenv's caching system.

## Table of Contents

- [Cache Types](#cache-types)
- [Common Issues](#common-issues)
- [Troubleshooting Steps](#troubleshooting-steps)
- [Cache Migration](#cache-migration)
- [Performance Optimization](#performance-optimization)
- [Diagnostic Commands](#diagnostic-commands)

## Cache Types

goenv uses multiple cache layers for different purposes:

### 1. Version List Cache

**Location:** `~/.goenv/cache/versions/`

**Purpose:** Caches available Go versions from golang.org

**Contents:**
- `versions.json` - Full version list
- `versions.json.etag` - ETag for efficient updates
- `versions.json.timestamp` - Last fetch time

**Lifetime:** 6 hours fresh, 7 days valid, then stale

**When used:**
- `goenv list --remote`
- `goenv install <version>`
- Version discovery operations

### 2. Build Cache (GOCACHE)

**Location:** `~/.cache/go-build/` (Linux) or `~/Library/Caches/go-build/` (macOS)

**Purpose:** Compiled package cache for faster builds

**Contents:**
- Compiled object files (`.a` files)
- Build metadata
- Architecture-specific caches

**Managed by:** Go toolchain (goenv provides cleaning tools)

**When used:**
- `go build`
- `go test`
- `go install`

### 3. Module Cache (GOMODCACHE)

**Location:** `~/go/pkg/mod/`

**Purpose:** Downloaded Go modules

**Contents:**
- Downloaded module sources
- Module metadata
- Checksums

**Managed by:** Go toolchain (goenv provides cleaning tools)

**When used:**
- `go get`
- `go mod download`
- Module-aware operations

### 4. Tools Cache

**Location:** `~/.goenv/tools/` or `~/.goenv/versions/<version>/tools/`

**Purpose:** Installed development tools per Go version

**Contents:**
- Tool binaries (gopls, staticcheck, etc.)
- Tool metadata

**When used:**
- `goenv tools install`
- `goenv default-tools`

## Common Issues

### Issue 1: "Stale version list" Warning

**Symptom:**
```
⚠ Version list cache is stale (> 7 days old)
```

**Cause:** Cache hasn't been refreshed in over 7 days

**Solution:**
```bash
goenv refresh
```

**Prevention:**
```bash
# Automatic refresh (happens naturally with use)
goenv list --remote
goenv install 1.25.2
```

### Issue 2: "Cache corruption detected"

**Symptom:**
```
Error: Failed to parse version cache
Error: Inconsistent cache state
```

**Cause:** Corrupted cache files (interrupted download, disk errors)

**Solution:**
```bash
# Clear version cache
rm -rf ~/.goenv/cache/versions/*

# Refresh
goenv refresh
```

**Or use the cache command:**
```bash
goenv cache status
goenv cache clear
```

### Issue 3: Build Cache Full

**Symptom:**
```
Error: disk full
warning: build cache is full
```

**Cause:** Build cache has grown too large

**Solution:**
```bash
# Check cache size
goenv cache status

# Clean old caches (30+ days)
goenv cache clean build --older-than 30d

# Prune to specific size
goenv cache clean build --max-bytes 1GB

# Clear all build caches
goenv cache clean build --force
```

### Issue 4: Module Download Failures

**Symptom:**
```
Error: failed to download module
Error: checksum mismatch
```

**Cause:** Corrupted module cache or proxy issues

**Solution:**
```bash
# Clear module cache
goenv cache clean mod --force

# Or use Go's built-in command
go clean -modcache

# Re-download modules
go mod download
```

### Issue 5: Wrong Go Version After Cache Clear

**Symptom:**
After clearing caches, wrong Go version is active

**Cause:** Shims need regeneration

**Solution:**
```bash
goenv rehash
goenv current  # Verify correct version
```

### Issue 6: Slow Version Listing

**Symptom:**
`goenv list --remote` takes 10+ seconds

**Cause:** Network timeout or stale cache check

**Solution:**
```bash
# Use offline mode for instant response
GOENV_OFFLINE=1 goenv list --remote

# Or refresh cache and retry
goenv refresh
goenv list --remote
```

### Issue 7: Architecture-Specific Cache Conflicts

**Symptom:**
```
Error: incompatible architecture in cache
Binary format mismatch
```

**Cause:** Cached files from different architecture (e.g., ARM64 vs AMD64)

**Solution:**
```bash
# Clean architecture-specific caches
goenv cache clean build --version $(goenv current --bare)

# Or clear all for current version
goenv cache clean all --version $(goenv current --bare)
```

**On Apple Silicon with Rosetta:**
```bash
# Check if running under Rosetta
goenv doctor

# Clear AMD64 caches if switching to native ARM64
arch -arm64 goenv cache clean all --force
```

### Issue 8: ETag Validation Failures

**Symptom:**
```
Warning: ETag validation failed
Error: cache metadata corrupted
```

**Cause:** Corrupted ETag files

**Solution:**
```bash
# Remove ETag files
rm ~/.goenv/cache/versions/*.etag

# Force refresh
goenv refresh
```

## Troubleshooting Steps

### Step 1: Check Cache Status

```bash
# Overall cache health
goenv doctor

# Detailed cache information
goenv cache status
```

**Expected output:**
```
Cache Status:
✓ Version cache: Fresh (last updated 2 hours ago)
✓ Build cache: 1.2 GB (3 versions)
✓ Module cache: 450 MB
```

### Step 2: Verify Cache Locations

```bash
# Version cache
ls -lh ~/.goenv/cache/versions/

# Build cache
du -sh ~/.cache/go-build/  # Linux
du -sh ~/Library/Caches/go-build/  # macOS

# Module cache
du -sh ~/go/pkg/mod/
```

### Step 3: Test Cache Refresh

```bash
# Clear and refresh
goenv cache clear
goenv refresh

# Verify
goenv list --remote
```

### Step 4: Clean Specific Caches

```bash
# Preview what would be cleaned
goenv cache clean all --dry-run

# Clean old caches
goenv cache clean all --older-than 30d --force
```

### Step 5: Check Permissions

```bash
# Ensure cache directories are writable
ls -ld ~/.goenv/cache
ls -ld ~/.cache/go-build  # Linux
ls -ld ~/Library/Caches/go-build  # macOS

# Fix permissions if needed
chmod -R u+w ~/.goenv/cache
```

### Step 6: Verify Network Connectivity

```bash
# Test network access
curl -I https://go.dev/dl/?mode=json

# Check DNS
nslookup go.dev

# Test with goenv
GOENV_DEBUG=1 goenv refresh
```

## Cache Migration

### Migrating from Bash goenv to Go goenv

The Go implementation uses a different cache structure. Here's how to migrate:

#### Before Migration

**Bash goenv cache structure:**
```
~/.goenv/
├── cache/
│   └── <hash>  # Opaque cache files
└── versions/
    └── <version>/
```

**Go goenv cache structure:**
```
~/.goenv/
├── cache/
│   └── versions/
│       ├── versions.json
│       ├── versions.json.etag
│       └── versions.json.timestamp
└── versions/
    └── <version>/
```

#### Migration Steps

**1. Backup existing installation:**
```bash
# Create backup
cp -r ~/.goenv ~/.goenv.backup

# Note current versions
goenv versions > ~/goenv-versions-backup.txt
```

**2. Upgrade to Go implementation:**
```bash
# Install Go-based goenv
curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash

# Or update via git
cd ~/.goenv && git pull
```

**3. Clear old caches:**
```bash
# Remove bash-style cache
rm -rf ~/.goenv/cache/*

# Regenerate with new system
goenv refresh
```

**4. Verify installation:**
```bash
goenv doctor
goenv list
goenv current
```

**5. Rebuild caches:**
```bash
# Regenerate shims
goenv rehash

# Refresh version cache
goenv refresh

# Verify Go versions work
goenv use $(goenv current --bare)
go version
```

### Migrating Between Machines

**Export configuration (source machine):**
```bash
# Save version configuration
cat .go-version > version-config.txt

# Save installed versions list
goenv list --bare > installed-versions.txt

# DON'T copy these (machine-specific):
# - ~/.goenv/versions/ (compiled binaries)
# - ~/.goenv/cache/ (cache data)
# - ~/.goenv/shims/ (generated shims)
```

**Import configuration (target machine):**
```bash
# Copy version configuration
cp version-config.txt .go-version

# Install versions
while read version; do
  goenv install "$version"
done < installed-versions.txt

# Use project version
goenv use $(cat .go-version)
```

See [What NOT to Sync](./advanced/WHAT_NOT_TO_SYNC.md) for detailed guidance.

### Migrating Architecture (e.g., Intel to Apple Silicon)

**On Intel Mac:**
```bash
# Save list of installed versions
goenv list --bare > ~/intel-versions.txt

# Note custom tools
goenv tools list > ~/intel-tools.txt
```

**On Apple Silicon Mac:**
```bash
# Install goenv
curl -sfL https://raw.githubusercontent.com/go-nv/goenv/master/install.sh | bash

# Install versions (will download ARM64 binaries)
while read version; do
  goenv install "$version"
done < ~/intel-versions.txt

# Reinstall tools (will compile for ARM64)
goenv default-tools

# Or sync from old setup
goenv sync-tools $(cat ~/intel-tools.txt)
```

**Clean up old caches:**
```bash
# Remove any lingering AMD64 caches
goenv cache clean all --old-format --force
```

## Performance Optimization

### Optimize for Speed

**1. Enable offline mode for CI/CD:**
```bash
# In CI/CD pipelines
export GOENV_OFFLINE=1
goenv list --remote  # Uses embedded versions (8ms vs 500ms)
```

**2. Pre-warm caches:**
```bash
# Before development session
goenv refresh
goenv list --remote > /dev/null
```

**3. Use appropriate cache lifetimes:**
```bash
# For development (default - good balance)
# Fresh: 6 hours, Valid: 7 days

# For CI/CD (offline mode)
export GOENV_OFFLINE=1  # No network calls
```

### Optimize for Space

**1. Regular cache cleanup:**
```bash
# Clean weekly
goenv cache clean all --older-than 7d --force

# Prune to size limit
goenv cache clean all --max-bytes 2GB --force
```

**2. Per-project module cache:**
```bash
# Use vendor directory instead of global cache
go mod vendor
```

**3. Remove unused Go versions:**
```bash
# List installed versions
goenv list

# Remove old versions
goenv uninstall 1.22.0 --force
goenv uninstall 1.21.3 --force
```

### Optimize for Reliability

**1. Use checksums:**
```bash
# Generate inventory with checksums
goenv inventory go --checksums --json > inventory.json

# Verify on other machines
# (compare checksums to detect corruption)
```

**2. Regular health checks:**
```bash
# Daily/weekly health check
goenv doctor

# Automated in CI/CD
goenv doctor || goenv refresh
```

**3. Backup critical configurations:**
```bash
# Backup .go-version files
find . -name .go-version -exec cp {} {}.backup \;

# Backup tool lists
goenv tools list > ~/.goenv/tools-backup.txt
```

## Diagnostic Commands

### Check Overall Health

```bash
# Full diagnostic
goenv doctor

# Cache-specific status
goenv cache status
```

### Debug Cache Issues

```bash
# Enable debug logging
GOENV_DEBUG=1 goenv refresh

# Verbose cache operations
GOENV_DEBUG=1 goenv cache clean all --dry-run
```

### Inspect Cache Contents

```bash
# Version cache
cat ~/.goenv/cache/versions/versions.json | jq '.[0:5]'

# Check cache age
stat ~/.goenv/cache/versions/versions.json

# Check ETag
cat ~/.goenv/cache/versions/versions.json.etag
```

### Test Cache Performance

```bash
# Measure cache hit
time goenv list --remote  # Should be fast with warm cache

# Measure cache miss
goenv cache clear
time goenv list --remote  # Slower (fetches from network)

# Measure offline mode
time GOENV_OFFLINE=1 goenv list --remote  # Fastest (8ms)
```

### Monitor Cache Growth

```bash
# Track cache sizes over time
echo "$(date): $(du -sh ~/.cache/go-build | cut -f1)" >> cache-growth.log

# Alert if too large
SIZE=$(du -s ~/.cache/go-build | cut -f1)
if [ $SIZE -gt 5000000 ]; then  # 5GB
  echo "Warning: Build cache is large ($SIZE KB)"
fi
```

## Cache Best Practices

### For Development

```bash
# Use normal caching (default)
goenv list --remote
goenv install 1.25.2

# Refresh weekly
goenv refresh

# Clean monthly
goenv cache clean all --older-than 30d
```

### For CI/CD

```bash
# Use offline mode for speed and reproducibility
export GOENV_OFFLINE=1
goenv install 1.25.2

# Cache goenv versions directory between runs
# (not the cache directory - it's not needed in offline mode)

# Clean before archiving
goenv cache clean all --force
```

### For Production Deployments

```bash
# Pin exact versions
goenv install 1.25.2
goenv use 1.25.2

# Verify checksums
goenv inventory go --checksums

# Don't sync cache directories
# (each machine generates its own)
```

## Getting Help

- **Cache issues**: `goenv doctor`
- **General help**: `goenv help cache`
- **Detailed documentation**: [Smart Caching Guide](./advanced/SMART_CACHING.md)
- **Report issues**: [GitHub Issues](https://github.com/go-nv/goenv/issues)

## See Also

- [Smart Caching](./advanced/SMART_CACHING.md) - Cache internals and strategy
- [Platform Support](../reference/PLATFORM_SUPPORT.md) - Platform-specific cache locations
- [CI/CD Guide](CI_CD_GUIDE.md) - CI/CD cache optimization
- [What NOT to Sync](./advanced/WHAT_NOT_TO_SYNC.md) - Cache portability
