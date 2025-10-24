# What NOT to Sync Across Machines

When using goenv across multiple machines (via dotfiles sync, network drives, or container volumes), it's critical to understand which directories should and shouldn't be synchronized. Syncing the wrong directories can cause corruption, performance issues, and hard-to-debug errors.

## ⚠️ Critical: NEVER Sync These

### 1. Build Caches (`GOCACHE` / `go-build` directories)

**Location**: `$GOENV_ROOT/versions/*/go-build*`

**Why NOT to sync**:

- **Architecture-specific**: Contains compiled binaries for specific OS/arch combinations
- **Machine-specific paths**: Absolute paths baked into cache entries
- **File locking**: Can cause corruption when accessed from multiple machines
- **Performance**: Network syncing kills the performance benefit of caching

**What happens if you sync it**:

```
# Typical error
go: inconsistency in file cache
cache entry has incorrect hash

# Or worse
version mismatch error
```

**Solution**: Each machine should have its own build cache.

```bash
# Check your build cache location
go env GOCACHE

# With goenv, it's isolated per version and architecture
goenv cache status  # See your cache layout
```

### 2. Shims Directory (`$GOENV_ROOT/shims`)

**Location**: `$GOENV_ROOT/shims`

**Why NOT to sync**:

- **Binary executables**: OS/architecture-specific
- **Symlinks**: May not work across different systems
- **Auto-generated**: Recreated by `goenv rehash`

**What happens if you sync it**:

```
# macOS → Linux sync
zsh: exec format error: go

# Or
/bin/sh: go: cannot execute binary file
```

**Solution**: Run `goenv rehash` on each machine after syncing other files.

```bash
# After pulling dotfiles
goenv rehash
```

### 3. Version Binaries (`$GOENV_ROOT/versions/*/bin`)

**Location**: `$GOENV_ROOT/versions/1.23.2/bin`

**Why NOT to sync**:

- **Platform-specific**: Go binaries are OS/architecture-specific
- **Large files**: Slow to sync, defeat caching
- **Different toolchains**: Windows `.exe` vs Linux ELF vs macOS Mach-O

**What happens if you sync it**:

```
# Cross-platform sync
-bash: /path/to/go: cannot execute binary file: Exec format error
```

**Solution**: Install Go versions independently on each machine.

```bash
# On each machine
goenv install 1.23.2
```

## ✅ Safe to Sync (With Caveats)

### 1. Version Selection Files

**✅ SAFE**: `.go-version`, `.tool-versions`

These are plain text files that specify which Go version to use.

```bash
# .go-version
1.23.2
```

**Recommended**: Sync these! They ensure consistent Go versions across your team/machines.

### 2. Module Cache (`GOMODCACHE`)

**⚠️ CONDITIONAL**: `~/go/pkg/mod` or `$GOMODCACHE`

**Read-only sync is OK**:

- Module downloads are identical across platforms
- Safe to share as a cache layer

**Write sync is DANGEROUS**:

- File locking issues with simultaneous access
- Race conditions during module downloads
- Corruption if multiple machines write

**Best Practice**:

```bash
# Option 1: Don't sync at all (safest)
# Each machine downloads modules independently

# Option 2: Read-only sync for performance
# One machine is the "source", others mount read-only
# (Advanced setup, typically in CI/dev containers)
```

**In containers**:

```dockerfile
# Mount module cache read-only
docker run -v ~/.cache/go-mod:/go/pkg/mod:ro golang:1.23
```

### 3. Configuration Files

**✅ SAFE**: `$GOENV_ROOT/version`, `$GOENV_ROOT/.goenv-version`

These files track the global Go version.

**✅ SAFE**: `$GOENV_ROOT/.goenv.json` (if you add custom config)

**Recommended**: Sync these for consistent default versions.

## 🎯 Recommended Sync Strategy

### For Dotfiles Repositories

**DO sync** (safe, small, machine-independent):

```bash
# In your dotfiles repo
~/.go-version              # Project-level version
~/.tool-versions           # ASDF-compatible format
~/code/project/.go-version # Per-project versions
```

**DON'T sync** (dangerous, large, machine-specific):

```bash
# Exclude from dotfiles sync
~/.goenv/versions/    # Go installations
~/.goenv/shims/       # Generated shims
~/.cache/go-build/    # Build cache
~/go/pkg/mod/         # Module cache (usually)
```

### Example `.gitignore` for Dotfiles

```gitignore
# In your dotfiles repo
.goenv/versions/
.goenv/shims/
.cache/go-build/

# Optionally exclude module cache
go/pkg/mod/

# DO commit
!.go-version
!.tool-versions
!.goenv-version
```

### Example `rsync` Excludes

```bash
# Sync dotfiles excluding dangerous directories
rsync -av \
  --exclude '.goenv/versions/' \
  --exclude '.goenv/shims/' \
  --exclude '.cache/go-build/' \
  --exclude 'go/pkg/mod/' \
  ~/dotfiles/ user@remote:~/
```

## 📦 Per-Host Binary Isolation

For tools installed with `go install`:

**Problem**: Binaries in `~/go/bin` are architecture-specific but share a common path.

**Solution 1**: Per-host GOPATH (Recommended)

```bash
# In your shell RC file
if [ -n "$GOENV_ROOT" ]; then
  # Use host-specific bin directory
  export GOPATH="$GOENV_ROOT/hosts/$(uname -s)-$(uname -m)"
  export PATH="$GOPATH/bin:$PATH"
else
  export GOPATH="$HOME/go"
  export PATH="$GOPATH/bin:$PATH"
fi
```

This gives you:

```
$GOENV_ROOT/hosts/
  Linux-x86_64/bin/
  Darwin-arm64/bin/
  Windows-x86_64/bin/
```

**Solution 2**: Exclude from sync

```bash
# Add to .gitignore or rsync --exclude
go/bin/
```

Then run `go install` independently on each machine.

## 🐳 Container & NFS Considerations

### Docker Bind Mounts

**DON'T** bind-mount these from host:

```yaml
# ❌ BAD - causes issues
volumes:
  - ~/.goenv/versions:/root/.goenv/versions # Wrong!
  - ~/.cache/go-build:/root/.cache/go-build # Wrong!
```

**DO** use volumes or let container create its own:

```yaml
# ✅ GOOD - isolated caches
volumes:
  - goenv-versions:/root/.goenv/versions
  - go-build-cache:/root/.cache/go-build
  - go-mod-cache:/go/pkg/mod
```

### NFS / Network Drives

**Issue**: File locking doesn't work reliably over NFS.

**Symptoms**:

```
fatal error: concurrent map writes
runtime error: slice bounds out of range
```

**Solution**: Use local storage for caches.

```bash
# Force local cache (not on NFS mount)
export GOCACHE="/tmp/go-build-$USER"
export GOMODCACHE="/tmp/go-mod-$USER"

# Or with goenv
export GOENV_ROOT="$HOME/.local/goenv"  # Local, not NFS
```

Check your filesystem:

```bash
# See what type of filesystem goenv is on
goenv doctor  # Shows filesystem warnings
```

## 🔍 Detecting Sync Issues

### Symptoms of Bad Sync

1. **Build cache errors**:

   ```
   go: inconsistency in file cache
   ```

2. **Version mismatch**:

   ```
   version mismatch error
   ```

3. **Exec format errors**:

   ```
   cannot execute binary file: Exec format error
   ```

4. **Performance degradation**:
   - Builds taking forever
   - Constant re-downloading of modules

### Diagnosis

```bash
# Check if caches are on problematic filesystem
goenv doctor

# See cache locations and sizes
goenv cache status

# Check for cross-platform binaries
file ~/.goenv/versions/1.23.2/bin/go
# Should match your current platform

# Verify build cache architecture
goenv cache info
```

### Fix

```bash
# Nuclear option: clean everything
goenv cache clean all
goenv rehash

# Or more targeted
goenv cache clean build
goenv cache migrate  # Move to architecture-aware caches
```

## 📝 Summary

### ✅ Safe to Sync

- `.go-version` files
- `.tool-versions` files
- `$GOENV_ROOT/version` (global version)
- Custom configuration files

### ❌ Never Sync

- `$GOENV_ROOT/versions/*/go-build*` (build caches)
- `$GOENV_ROOT/shims/` (generated shims)
- `$GOENV_ROOT/versions/*/bin/` (Go binaries)
- `$GOCACHE` (build cache)

### ⚠️ Conditional Sync

- `$GOMODCACHE` (module cache) - Read-only only
- `$GOPATH/bin` (installed tools) - Use per-host isolation

### 🎯 Best Practice

1. **Sync version files** (`.go-version`) for consistency
2. **Exclude binary directories** from dotfiles sync
3. **Use per-host paths** for `GOPATH/bin`
4. **Run `goenv rehash`** after syncing
5. **Keep caches local** (not on NFS)
6. **Use `goenv doctor`** to detect filesystem issues

## See Also

- [Cache Management](../reference/COMMANDS.md#goenv-cache) - Manage goenv caches
- [Environment Detection](../../ENVIRONMENT_DETECTION.md) - Detect container/NFS issues
- [Smart Caching](SMART_CACHING.md) - How goenv's caching works
- [Cross-Building](CROSS_BUILDING.md) - Platform-specific build strategies
