# Smart Caching Strategy

## Cache Management

### Clear Cache

If you want to force a fresh fetch from the API (e.g., right after a new Go release):

```bash
# Clear all caches and force fresh fetch
goenv refresh

# See detailed output
goenv refresh --verbose
```

This removes both cache files:

- `versions-cache.json` - Version list cache
- `releases-cache.json` - Full release metadata cache

The next version fetch will get fresh data from go.dev.

### Offline Mode

For maximum efficiency, air-gapped environments, or reproducible CI/CD pipelines, you can disable all online fetching and use only embedded versions:

```bash
# Enable offline mode
export GOENV_OFFLINE=1

# Now all version commands use embedded data (no network calls)
goenv install --list
goenv list --remote
```

**When to use offline mode:**

- **Air-gapped environments** - Systems without internet access
- **CI/CD pipelines** - Guaranteed reproducibility and maximum speed
- **Security requirements** - No outbound network calls
- **Performance critical** - Fastest possible operation (< 40ms)
- **Bandwidth constrained** - Mobile hotspots, metered connections

**How it works:**

When `GOENV_OFFLINE=1` is set, goenv completely bypasses the network layer and cache system, using only the embedded versions compiled into the binary. These embedded versions are:

- Generated at build time from go.dev API
- Comprehensive (331 versions at last update)
- Updated with each goenv release
- Complete with file hashes and metadata

**Performance:**

```bash
# Online mode (with cache)
$ time goenv install --list > /dev/null
real    0m0.042s    # Cache hit: 42ms

# Offline mode
$ GOENV_OFFLINE=1 time goenv install --list > /dev/null
real    0m0.008s    # Embedded: 8ms (5x faster!)
```

**Limitations:**

- Embedded versions are only updated when you update goenv itself
- Won't see new Go releases until you update goenv
- No smart cache freshness checking

**Debug output:**

```bash
$ GOENV_OFFLINE=1 GOENV_DEBUG=1 goenv install --list
Debug: Fetching available Go versions...
Debug: GOENV_OFFLINE=1, skipping online fetch and using embedded versions
```

## How It Works

The smart caching system uses three tiers based on cache age:

## Strategy

### Tier 1: Fresh Cache (< 6 hours old)

**Use cached data immediately, no API calls**

```
User: goenv install --list
  → Cache age: 2 hours
  → Action: Return cached data
  → API calls: 0
  → Time: ~40ms
```

**Reasoning**: Go releases are infrequent. If cache was updated recently, it's almost certainly still current.

### Tier 2: Recent Cache (6 hours to 7 days old)

**Quick freshness check using lightweight API**

```
User: goenv install --list
  → Cache age: 2 days
  → Action: Quick check - fetch latest 2 versions only
  → Compare: cached[0] vs latest[0]

  IF MATCH:
    → Cache is current, use it
    → API calls: 1 (lightweight, ~200ms)
    → Time: ~240ms total

  IF MISMATCH:
    → New version detected!
    → Fetch all versions with include=all
    → Update cache
    → API calls: 2 (quick + full, ~700ms)
    → Time: ~740ms total
```

**Reasoning**: This is your brilliant idea! Check if there's a new version using the fast endpoint (just 2 versions). If cache is still current, avoid expensive `include=all` fetch. If new version exists, do full refresh to get ALL new versions (not just the latest 2).

### Tier 3: Stale Cache (> 7 days old)

**Force full refresh without checking**

```
User: goenv install --list
  → Cache age: 8 days
  → Action: Force full refresh (include=all)
  → API calls: 1 (full, ~500ms)
  → Time: ~540ms total
```

**Reasoning**: If cache hasn't been used in a week, just refresh it completely. Avoid the quick check since it's likely outdated anyway.

## API Endpoints Used

### Lightweight Endpoint (Quick Check)

```
GET https://go.dev/dl/?mode=json
Response: ~10KB (2 versions, latest stable + previous)
Time: ~200ms
Use: Check if cache is current
```

### Full Endpoint (Complete Refresh)

```
GET https://go.dev/dl/?mode=json&include=all
Response: ~2MB (331 versions, all history)
Time: ~500ms
Use: Initial fetch, new version detected, or stale cache
```

## Real-World Scenarios

### Scenario 1: Daily Active User

```
Day 1, 9:00 AM:  First use → Full fetch (500ms)
Day 1, 2:00 PM:  Tier 1 → Cached (40ms)
Day 1, 6:00 PM:  Tier 1 → Cached (40ms)
Day 2, 9:00 AM:  Tier 2 → Quick check, still current (240ms)
Day 3, 9:00 AM:  Tier 2 → Quick check, still current (240ms)
Day 4, 9:00 AM:  Tier 2 → Quick check, still current (240ms)

Weekly API load: 1 full fetch + 6 quick checks
```

### Scenario 2: Weekly User

```
Day 1:  First use → Full fetch (500ms)
Day 8:  Tier 3 → Stale, full refresh (540ms)
Day 15: Tier 3 → Stale, full refresh (540ms)

Weekly API load: 1 full fetch
```

### Scenario 3: New Release During Day 2-7

```
Day 1:  First use → Full fetch, cache has go1.25.1 (500ms)
Day 3:  Go 1.25.2 released
Day 4:  User runs command
        → Tier 2: Quick check
        → Cached: go1.25.1
        → Latest: go1.25.2
        → MISMATCH DETECTED!
        → Full refresh with include=all
        → Gets ALL new versions (might be go1.25.2, go1.25.3, etc.)
        → Time: 740ms

Result: User sees new version within 1-7 days of release
```

### Scenario 4: Multiple Releases in 4 Days (Your Question!)

```
Day 0:  Cache created with 331 versions, latest: go1.25.0
Day 1:  go1.25.1 released
Day 2:  go1.25.2 released
Day 3:  go1.25.3 released
Day 4:  go1.25.4 and go1.25.5 released

Day 4 User runs command:
  → Tier 2: Quick check
  → Fetch latest 2: [go1.25.5, go1.25.4]
  → Compare: go1.25.0 (cached) != go1.25.5 (latest)
  → NEW VERSION DETECTED!
  → Full fetch with include=all
  → Gets ALL 336 versions (331 + 5 new)
  → Cache updated

✅ Result: User sees ALL 5 new versions, not just latest 2!
```

## Performance Comparison

| Scenario                      | Old (24hr TTL)     | New (Smart)          | Improvement     |
| ----------------------------- | ------------------ | -------------------- | --------------- |
| **Within 6h**                 | Full fetch (500ms) | Cached (40ms)        | 12x faster      |
| **Within 7d, no new release** | Full fetch (500ms) | Quick check (240ms)  | 2x faster       |
| **Within 7d, new release**    | Full fetch (500ms) | Quick + full (740ms) | Slightly slower |
| **7+ days**                   | Full fetch (500ms) | Full fetch (540ms)   | Same            |

## API Load Reduction

### Daily Active User

- **Old**: 7 full fetches/week = 7 × 2MB = 14MB
- **New**: 1 full fetch + 6 quick checks = (1 × 2MB) + (6 × 10KB) = 2.06MB
- **Savings**: 85% less bandwidth

### Weekly User

- **Old**: 1 full fetch/week = 2MB
- **New**: 1 full fetch/week = 2MB
- **Savings**: Same (optimized for active users)

## Configuration

Currently hardcoded, but could be made configurable:

```bash
# Environment variables (future)
export GOENV_CACHE_FRESH_TTL=6h        # Tier 1 threshold
export GOENV_CACHE_QUICK_TTL=168h      # Tier 2 threshold (7 days)
export GOENV_CACHE_DISABLE=false       # Force always fetch
```

## Debug Output

```bash
# Fresh cache (< 6 hours)
$ GOENV_DEBUG=1 goenv install --list
Debug: Fetching available Go versions...
Debug: Cache is fresh (< 6 hours old)
Debug: Using cached versions

# Recent cache, still current (6h-7d, no new version)
$ GOENV_DEBUG=1 goenv install --list
Debug: Fetching available Go versions...
Debug: Cache is 48h0m0s old, doing quick freshness check...
Debug: Cache is current (latest: go1.25.2)
Debug: Using cached versions

# Recent cache, new version detected (6h-7d, new release)
$ GOENV_DEBUG=1 goenv install --list
Debug: Fetching available Go versions...
Debug: Cache is 48h0m0s old, doing quick freshness check...
Debug: New version detected (cached: go1.25.1, latest: go1.25.2), forcing full refresh
Debug: Cache miss or expired: new version detected, need full refresh
Debug: Fetching all versions from go.dev API...

# Stale cache (> 7 days)
$ GOENV_DEBUG=1 goenv install --list
Debug: Fetching available Go versions...
Debug: Cache is stale (> 7 days old), forcing full refresh
Debug: Cache miss or expired: cache expired
Debug: Fetching all versions from go.dev API...

# Network error during quick check (fallback to cache)
$ GOENV_DEBUG=1 goenv install --list
Debug: Fetching available Go versions...
Debug: Cache is 48h0m0s old, doing quick freshness check...
Debug: Quick check failed (connection timeout), using cache anyway
Debug: Using cached versions
```

## Benefits

### For Users

- ✅ **Faster listings** (40ms vs 500ms for recent queries)
- ✅ **Works offline** (uses cache when network unavailable)
- ✅ **Always complete** (gets ALL new versions, not just latest)
- ✅ **Auto-updating** (detects new releases automatically)

### For Google's API

- ✅ **85% less bandwidth** for active users
- ✅ **Fewer requests** (quick check vs full fetch)
- ✅ **Smarter polling** (only when likely to be updates)

### For Developers

- ✅ **No maintenance** (auto-detects and updates)
- ✅ **No bot needed** (unlike bash version's commit bot)
- ✅ **Configurable** (can adjust TTLs if needed)

## Edge Cases

### Case 1: Network Error During Quick Check

```
→ Quick check fails (timeout, DNS, etc.)
→ Use cached data anyway (better than failing)
→ Next run will try again
```

### Case 2: Malformed Cache File

```
→ JSON parse error
→ Treat as cache miss
→ Do full refresh
```

### Case 3: Empty Cache

```
→ No cached versions found
→ Skip all checks
→ Do full refresh
```

### Case 4: API Returns Different Version Order

```
→ Compare versions by string match
→ Handles pre-releases correctly
→ Stable versions always prioritized
```

## Future Enhancements

### Potential Improvements

1. **Background refresh**: Update cache in background after returning cached data
2. **Incremental updates**: Fetch only versions newer than cached latest
3. **Multiple caches**: Separate cache per major version (1.21.x, 1.22.x)
4. **Compression**: gzip cache files (2MB → 200KB)
5. **ETag support**: Use HTTP ETags if API supports it

### Not Planned

- ❌ Cloud sync (keep it local-only)
- ❌ Telemetry (privacy first)
- ❌ External dependencies (stdlib only)

## Comparison with Other Tools

### pyenv/rbenv

- Static definition files
- Requires `git pull` to update
- No smart caching

### nvm

- No persistent cache
- Fetches on every `nvm ls-remote`
- Slow repeated listings

### rustup

- Simple cache with fixed TTL
- No smart freshness checking
- Good but not optimal

### goenv (this implementation)

- **Three-tier smart caching**
- **Automatic freshness detection**
- **Optimal bandwidth usage**
- **Best performance for active users**

## Build Cache Isolation

### Problem: "exec format error" When Switching Versions

When using multiple Go versions, you may encounter errors like:

```
fork/exec /Users/username/Library/Caches/go-build/.../staticcheck: exec format error
```

**Primary cause:** By default, Go uses a **shared build cache** (`GOCACHE`) across all Go versions. When you build tools with Go 1.23, then switch to Go 1.24, the cached binaries become incompatible because:
- Tool binaries contain Go runtime code specific to the version they were built with
- Cached intermediate build artifacts may have version-specific formats
- Go 1.23's cache entries may not be compatible with Go 1.24's expectations

**Secondary causes (rare):**
- Migration from Intel to Apple Silicon (cached x86_64 binaries on new arm64 machine)
- Running Go under Rosetta vs natively on Apple Silicon
- Manual `GOARCH` overrides during builds

### Solution: Version-Specific Cache Isolation

Starting in goenv v3, **build caches are automatically isolated per Go version** to prevent these conflicts.

#### How It Works

When you run `goenv exec go build` or any Go command through goenv:

```bash
# Go 1.23.2
$ goenv exec go env GOCACHE
/Users/username/.goenv/versions/1.23.2/go-build

# Go 1.24.4
$ goenv exec go env GOCACHE
/Users/username/.goenv/versions/1.24.4/go-build

# Go 1.25.2
$ goenv exec go env GOCACHE
/Users/username/.goenv/versions/1.25.2/go-build
```

Each Go version gets its own isolated cache directory, preventing architecture/version mismatches.

#### Benefits

- ✅ **No more "exec format error"** when switching Go versions
- ✅ **Clean isolation** - Each Go version has its own build environment
- ✅ **No manual cache cleaning** required between version switches
- ✅ **Automatic and transparent** - works out of the box
- ✅ **Handles edge cases** - Migration scenarios and architecture changes also covered

#### Module Cache Isolation

Module caches (`GOMODCACHE`) are also isolated by default:

```bash
# Go 1.23.2
$ goenv exec go env GOMODCACHE
/Users/username/.goenv/versions/1.23.2/go-mod

# Go 1.24.4
$ goenv exec go env GOMODCACHE
/Users/username/.goenv/versions/1.24.4/go-mod
```

### Configuration

#### Disable Cache Isolation

If you prefer to use Go's default shared cache:

```bash
# Disable build cache isolation
export GOENV_DISABLE_GOCACHE=1

# Disable module cache isolation
export GOENV_DISABLE_GOMODCACHE=1

# Now uses Go's default shared caches
$ goenv exec go env GOCACHE
/Users/username/Library/Caches/go-build
```

#### Custom Cache Locations

You can specify custom base directories for caches:

```bash
# Custom GOCACHE base directory
export GOENV_GOCACHE_DIR=/custom/path/gocache
# Results in: /custom/path/gocache/1.23.2

# Custom GOMODCACHE base directory
export GOENV_GOMODCACHE_DIR=/custom/path/gomodcache
# Results in: /custom/path/gomodcache/1.23.2
```

### Diagnosing Cache Issues

Use the `goenv clean --diagnose` command to check your cache configuration:

```bash
$ goenv clean --diagnose
🔍 Diagnosing cache issues...

Current architecture: darwin/arm64
Current Go version: 1.23.2

GOCACHE location: /Users/username/.goenv/versions/1.23.2/go-build
✅ Using version-specific cache (goenv managed)

GOMODCACHE location: /Users/username/.goenv/versions/1.23.2/go-mod
✅ Using version-specific module cache (goenv managed)

Cache isolation settings:
  ✅ GOENV_DISABLE_GOCACHE not set (cache isolation enabled)
  ✅ GOENV_DISABLE_GOMODCACHE not set (module cache isolation enabled)
```

### Cleaning Caches

If you need to clean caches (e.g., to free disk space):

```bash
# Clean build cache for current version
$ goenv clean build

# Clean module cache for current version
$ goenv clean modcache

# Clean both caches
$ goenv clean all
```

### Troubleshooting

#### Still Getting "exec format error"?

1. **Clean your shared system cache** (one-time migration):
   ```bash
   go clean -cache
   go clean -modcache
   ```

2. **Verify cache isolation is working**:
   ```bash
   goenv exec go env GOCACHE
   # Should show: ~/.goenv/versions/{version}/go-build
   ```

3. **Run diagnostics**:
   ```bash
   goenv doctor
   # Look for "Build cache isolation" and "Cache architecture" checks
   ```

#### Cache Taking Too Much Disk Space?

Each version has its own cache, which can use more disk space. To manage this:

```bash
# Check cache sizes
$ du -sh ~/.goenv/versions/*/go-build
1.2G    /Users/username/.goenv/versions/1.23.2/go-build
890M    /Users/username/.goenv/versions/1.24.4/go-build

# Clean unused versions
$ goenv uninstall 1.21.5
# Also removes that version's cache

# Or clean caches for all versions
$ for v in $(goenv list); do
    GOENV_VERSION=$v goenv clean all
  done
```

### Technical Details

**Cache Directory Structure:**

```
$GOENV_ROOT/
└── versions/
    ├── 1.23.2/
    │   ├── bin/          # Go distribution binaries
    │   ├── gopath/       # Installed tools (gopls, etc.)
    │   ├── go-build/     # Build cache (GOCACHE)
    │   └── go-mod/       # Module cache (GOMODCACHE)
    ├── 1.24.4/
    │   ├── bin/
    │   ├── gopath/
    │   ├── go-build/
    │   └── go-mod/
    └── 1.25.2/
        ├── bin/
        ├── gopath/
        ├── go-build/
        └── go-mod/
```

**When Cache Isolation Applies:**

- ✅ `goenv exec go <command>`
- ✅ Commands run through goenv shims (`go`, `gofmt`, etc.)
- ❌ Direct invocation of Go binary (bypasses goenv)

**Shims automatically use cache isolation** because they internally use `goenv exec`.
