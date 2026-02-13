// Package cache provides utilities and operations for managing Go build and module caches.
package cache

import (
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/go-nv/goenv/internal/utils"
)

// ParseByteSize parses byte size strings like "1GB", "500MB", "1.5GB" to bytes.
// Supported units: B, KB/K, MB/M, GB/G, TB/T (case-insensitive).
//
// Examples:
//
//	ParseByteSize("1GB")    // 1073741824
//	ParseByteSize("500MB")  // 524288000
//	ParseByteSize("1.5G")   // 1610612736
func ParseByteSize(s string) (int64, error) {
	s = strings.TrimSpace(strings.ToUpper(s))
	if s == "" {
		return 0, fmt.Errorf("empty byte size")
	}

	// Extract numeric part and unit
	var numStr string
	var unit string

	for i, c := range s {
		if (c >= '0' && c <= '9') || c == '.' {
			numStr += string(c)
		} else {
			unit = s[i:]
			break
		}
	}

	if numStr == "" {
		return 0, fmt.Errorf("invalid byte size format: %s", s)
	}

	// Parse the number
	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in byte size: %w", err)
	}

	// Parse the unit
	var multiplier float64
	switch unit {
	case "B", "":
		multiplier = 1
	case "KB", "K":
		multiplier = 1024
	case "MB", "M":
		multiplier = 1024 * 1024
	case "GB", "G":
		multiplier = 1024 * 1024 * 1024
	case "TB", "T":
		multiplier = 1024 * 1024 * 1024 * 1024
	default:
		return 0, fmt.Errorf("unknown unit in byte size: %s (valid units: B, KB, MB, GB, TB)", unit)
	}

	return int64(num * multiplier), nil
}

// ParseDuration parses duration strings like "30d", "1w", "24h".
// Supports standard time.ParseDuration units (h, m, s, ms, us, ns) plus:
//   - d, day, days (days)
//   - w, week, weeks (weeks)
//
// Examples:
//
//	ParseDuration("30d")    // 720 hours
//	ParseDuration("1w")     // 168 hours
//	ParseDuration("24h")    // 24 hours
func ParseDuration(s string) (time.Duration, error) {
	s = strings.TrimSpace(strings.ToLower(s))
	if s == "" {
		return 0, fmt.Errorf("empty duration")
	}

	// Try standard time.ParseDuration first (handles h, m, s, ms, us, ns)
	if d, err := time.ParseDuration(s); err == nil {
		return d, nil
	}

	// Handle custom units: d (days), w (weeks)
	var numStr string
	var unit string

	for i, c := range s {
		if (c >= '0' && c <= '9') || c == '.' {
			numStr += string(c)
		} else {
			unit = s[i:]
			break
		}
	}

	if numStr == "" {
		return 0, fmt.Errorf("invalid duration format: %s", s)
	}

	num, err := strconv.ParseFloat(numStr, 64)
	if err != nil {
		return 0, fmt.Errorf("invalid number in duration: %w", err)
	}

	switch unit {
	case "d", "day", "days":
		return time.Duration(num * float64(24*time.Hour)), nil
	case "w", "week", "weeks":
		return time.Duration(num * float64(7*24*time.Hour)), nil
	default:
		return 0, fmt.Errorf("unknown unit in duration: %s (valid units: s, m, h, d, w)", unit)
	}
}

// ParseABIFromCacheName extracts GOOS, GOARCH, and ABI variants from cache directory name.
//
// Examples:
//
//	ParseABIFromCacheName("go-build-linux-amd64-v3")
//	  -> goos="linux", goarch="amd64", abi={"GOAMD64":"v3"}
//
//	ParseABIFromCacheName("go-build-darwin-arm64")
//	  -> goos="darwin", goarch="arm64", abi=nil
//
//	ParseABIFromCacheName("go-build")
//	  -> goos="", goarch="", abi=nil (old format)
func ParseABIFromCacheName(cacheName string) (goos, goarch string, abi map[string]string) {
	if cacheName == "go-build" {
		return "", "", nil
	}

	// Remove "go-build-" prefix
	suffix := strings.TrimPrefix(cacheName, "go-build-")
	if suffix == cacheName {
		return "", "", nil // Invalid format
	}

	// Split by '-' to get components
	parts := strings.Split(suffix, "-")
	if len(parts) < 2 {
		return "", "", nil
	}

	goos = parts[0]
	goarch = parts[1]

	// Check for ABI variants in remaining parts
	if len(parts) > 2 {
		abi = make(map[string]string)
		remaining := parts[2:]

		// Process remaining parts for ABI variants, experiments, and CGO hash
		i := 0
		for i < len(remaining) {
			part := remaining[i]

			// Check for CGO hash (format: "cgo-<hash>")
			if part == "cgo" && i+1 < len(remaining) {
				abi["CGO_HASH"] = remaining[i+1]
				i += 2 // Skip both "cgo" and hash
				continue
			}

			// Check for GOEXPERIMENT (format: "exp-<experiments>")
			if part == "exp" && i+1 < len(remaining) {
				expValue := strings.ReplaceAll(remaining[i+1], "-", ",")
				abi["GOEXPERIMENT"] = expValue
				i += 2 // Skip both "exp" and value
				continue
			}

			// Check for architecture-specific ABI variants
			switch goarch {
			case "amd64":
				if strings.HasPrefix(part, "v") {
					abi["GOAMD64"] = part
				}
			case "arm":
				// Accept both "v6", "v7" and "6", "7" formats
				if strings.HasPrefix(part, "v") {
					abi["GOARM"] = strings.TrimPrefix(part, "v")
				} else if len(part) == 1 && part >= "5" && part <= "7" {
					// Accept bare digits 5-7 for ARM variants
					abi["GOARM"] = part
				}
			case "386":
				if part == "sse2" || part == "softfloat" {
					abi["GO386"] = part
				}
			case "mips", "mipsle":
				if part == "hardfloat" || part == "softfloat" {
					abi["GOMIPS"] = part
				}
			case "mips64", "mips64le":
				if part == "hardfloat" || part == "softfloat" {
					abi["GOMIPS64"] = part
				}
			case "ppc64", "ppc64le":
				if part == "power8" || part == "power9" || part == "power10" {
					abi["GOPPC64"] = part
				}
			case "riscv64":
				abi["GORISCV64"] = part
			case "wasm":
				abi["GOWASM"] = part
			}

			i++
		}
	}

	return goos, goarch, abi
}

// GetDirSize calculates total size and file count for a directory.
// Uses a default timeout of 10 seconds.
//
// Returns:
//   - size: total size in bytes
//   - files: number of files (or -1 if timed out)
func GetDirSize(path string) (size int64, files int, err error) {
	size, files = getDirSizeWithOptions(path, false, 10*time.Second)
	return size, files, nil
}

// GetDirSizeWithOptions calculates directory size with performance options.
//   - fast: if true, skips file counting (returns -1 for files)
//   - timeout: if walk exceeds this duration, returns approximate results (files = -1)
//
// Returns:
//   - size: total size in bytes (always accurate)
//   - files: number of files, or -1 if fast mode or timed out
func GetDirSizeWithOptions(path string, fast bool, timeout time.Duration) (size int64, files int, err error) {
	size, files = getDirSizeWithOptions(path, fast, timeout)
	return size, files, nil
}

// getDirSizeWithOptions is the internal implementation
func getDirSizeWithOptions(path string, fast bool, timeout time.Duration) (size int64, files int) {
	startTime := time.Now()
	timedOut := false

	err := filepath.WalkDir(path, func(filePath string, d os.DirEntry, err error) error {
		if err != nil {
			return nil // Skip errors
		}

		// Check timeout budget every 1000 files to avoid excessive time.Now() calls
		if !timedOut && files%1000 == 0 && time.Since(startTime) > timeout {
			timedOut = true
			// Don't return error - continue with what we have
		}

		if !d.IsDir() {
			info, err := d.Info()
			if err != nil {
				return nil // Skip if we can't get file info
			}
			size += info.Size()

			// In fast mode, skip counting files (or timeout)
			if !fast && !timedOut {
				files++
			}
		}
		return nil
	})

	if err != nil {
		return 0, 0
	}

	// Return -1 for files if in fast mode or timed out (indicates approximate)
	if fast || timedOut {
		return size, -1
	}

	return size, files
}

// GetCacheModTime returns the last modification time of a cache directory.
func GetCacheModTime(path string) (time.Time, error) {
	modTime := utils.GetFileModTime(path)
	if modTime.IsZero() {
		return time.Time{}, fmt.Errorf("failed to get modification time for %s", path)
	}
	return modTime, nil
}

// FormatNumber formats a number with thousand separators.
//
// Examples:
//
//	FormatNumber(1234)      // "1,234"
//	FormatNumber(1234567)   // "1,234,567"
func FormatNumber(n int) string {
	s := fmt.Sprintf("%d", n)
	if len(s) <= 3 {
		return s
	}

	// Add commas
	var result []rune
	for i, digit := range s {
		if i > 0 && (len(s)-i)%3 == 0 {
			result = append(result, ',')
		}
		result = append(result, digit)
	}

	return string(result)
}

// FormatFileCount formats file count, handling -1 for approximate/unknown counts.
//
// Examples:
//
//	FormatFileCount(1234, false)    // "1,234"
//	FormatFileCount(1234, true)     // "~1,234"
//	FormatFileCount(-1, false)      // "~"
func FormatFileCount(n int, approximate bool) string {
	if n < 0 {
		return "~"
	}
	if approximate {
		return "~" + FormatNumber(n)
	}
	return FormatNumber(n)
}

// FormatBytes formats byte count to human-readable string.
//
// Examples:
//
//	FormatBytes(1024)           // "1.0 KB"
//	FormatBytes(1536)           // "1.5 KB"
//	FormatBytes(1073741824)     // "1.0 GB"
func FormatBytes(bytes int64) string {
	return utils.FormatBytes(bytes)
}
