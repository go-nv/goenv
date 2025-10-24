# Environment Detection Quick Reference

## Quick Start

```bash
# Check your environment
goenv doctor

# Look for these sections:
# ✅ Runtime environment
# ✅ GOENV_ROOT filesystem
```

## API Quick Reference

```go
import "github.com/go-nv/goenv/internal/envdetect"

// Basic detection
info := envdetect.Detect()

// Filesystem detection
fsInfo := envdetect.DetectFilesystem("/path/to/goenv_root")

// Check for problems
if info.IsProblematicEnvironment() {
    for _, warning := range info.GetWarnings() {
        fmt.Println(warning)
    }
}
```

## Environment Types

| Type | Description | Detection Method |
|------|-------------|------------------|
| `native` | Normal OS execution | Default |
| `container` | Docker, Podman, K8s, etc. | `/.dockerenv`, cgroups, env vars |
| `wsl` | Windows Subsystem for Linux | `/proc/version`, env vars |

## Container Types Detected

| Container | Detection Method |
|-----------|------------------|
| Docker | `/.dockerenv` file or cgroups |
| Podman | `/run/.containerenv` or `$CONTAINER` env var |
| Kubernetes | cgroups or `$KUBERNETES_SERVICE_HOST` |
| LXC | cgroups |
| BuildKit | `$BUILDKIT_SANDBOX_HOSTNAME` |

## Filesystem Types

| Type | Common Examples | Issues |
|------|----------------|--------|
| `local` | ext4, xfs, btrfs, APFS | ✅ None (optimal) |
| `nfs` | NFS, NFS4 | ⚠️ File locking, caching issues |
| `smb` | CIFS, Samba | ⚠️ Symlinks, permissions don't work |
| `bind` | Container mounts | ⚠️ Persistence, permissions |
| `fuse` | sshfs, Google Drive | ⚠️ Very slow performance |

## Common Warnings

### Container Warning
**Message**: "Running in {type} container: Ensure volumes are properly mounted"

**Action**: Verify your Docker/Podman volume configuration

### WSL 1 Warning
**Message**: "WSL 1 detected: Performance may be slower than WSL 2"

**Action**: Upgrade to WSL 2 for better performance

### NFS Warning
**Message**: "NFS filesystem detected: File locking and permissions may behave differently"

**Action**: Move GOENV_ROOT to a local filesystem

### SMB Warning
**Message**: "SMB/CIFS filesystem detected: Symbolic links may not work"

**Action**: Use a local filesystem or Linux-native storage

### FUSE Warning
**Message**: "FUSE filesystem detected: Performance may be impacted"

**Action**: Move GOENV_ROOT to a local filesystem

## Best Practices

### Containers
```dockerfile
# Use named volumes
volumes:
  - goenv_root:/root/.goenv

# Or bind mount to persistent storage
volumes:
  - ./goenv:/root/.goenv
```

### WSL
```bash
# ✅ Good: Linux filesystem
export GOENV_ROOT="$HOME/.goenv"

# ❌ Bad: Windows filesystem
export GOENV_ROOT="/mnt/c/Users/username/.goenv"
```

### Network Filesystems
```bash
# ❌ Avoid NFS/SMB for GOENV_ROOT
export GOENV_ROOT="/mnt/nfs/goenv"  # Don't do this

# ✅ Use local storage
export GOENV_ROOT="$HOME/.goenv"    # Do this instead
```

## Troubleshooting

### No warnings but experiencing issues
```bash
# Run full diagnostic
goenv doctor

# Check filesystem manually (Linux)
df -T "$GOENV_ROOT"

# Check filesystem manually (macOS)
mount | grep "$GOENV_ROOT"
```

### False positive warnings
If you get a warning but everything works:
- The warning is informational, not blocking
- You can ignore it if not experiencing issues
- Keep it in mind for future troubleshooting

### Performance issues
If builds are slow:
1. Check filesystem type: `goenv doctor`
2. Verify you're not on NFS/FUSE
3. Move GOENV_ROOT to local storage
4. Clean caches: `goenv cache clean build`

## Code Examples

### Check if in container
```go
info := envdetect.Detect()
if info.IsContainer {
    log.Printf("Running in %s container", info.ContainerType)
}
```

### Check if in WSL
```go
info := envdetect.Detect()
if info.IsWSL {
    log.Printf("Running in WSL %s (%s)", info.WSLVersion, info.WSLDistro)
}
```

### Validate filesystem
```go
fsInfo := envdetect.DetectFilesystem(cfg.Root)
if fsInfo.FilesystemType == envdetect.FSTypeNFS {
    log.Warn("NFS detected: Performance may be degraded")
}
```

### Get all warnings
```go
info := envdetect.DetectFilesystem(path)
if info.IsProblematicEnvironment() {
    for _, warning := range info.GetWarnings() {
        fmt.Fprintf(os.Stderr, "⚠️  %s\n", warning)
    }
}
```

## Related Commands

```bash
# Full diagnostic
goenv doctor

# Check current version
goenv version

# Clean caches if having issues
goenv cache clean build
goenv cache clean all

# Check PATH
goenv which go
```

## Related Documentation

- [ENVIRONMENT_DETECTION.md](../ENVIRONMENT_DETECTION.md) - Full documentation
- [DOCTOR_COMMAND_FEATURE.md](../DOCTOR_COMMAND_FEATURE.md) - Doctor command details
- [WINDOWS_SUPPORT.md](../WINDOWS_SUPPORT.md) - Windows/WSL support
