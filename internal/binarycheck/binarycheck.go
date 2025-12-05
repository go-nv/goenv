package binarycheck

import (
	"debug/elf"
	"fmt"
	"github.com/go-nv/goenv/internal/osinfo"
	"os"
	"path/filepath"
	"strings"

	"github.com/go-nv/goenv/internal/utils"
)

// BinaryInfo contains information about a binary file
type BinaryInfo struct {
	Path        string
	IsELF       bool
	IsScript    bool
	ELFClass    string // "32-bit" or "64-bit"
	Interpreter string // Dynamic linker path (e.g., /lib64/ld-linux-x86-64.so.2)
	Machine     string // Architecture (e.g., "x86-64", "ARM")
	Shebang     string // For scripts
}

// CompatibilityIssue describes a compatibility problem
type CompatibilityIssue struct {
	Severity string // "error", "warning", "info"
	Message  string
	Hint     string // Actionable suggestion
}

// CheckBinary inspects a binary file and returns information about it
func CheckBinary(path string) (*BinaryInfo, error) {
	info := &BinaryInfo{
		Path: path,
	}

	// Open file
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("cannot open binary: %w", err)
	}
	defer f.Close()

	// Read first few bytes to determine type
	header := make([]byte, 4)
	n, err := f.Read(header)
	if err != nil || n < 4 {
		return info, nil // Not enough data, not a recognized format
	}

	// Check for shebang (#!)
	if header[0] == '#' && header[1] == '!' {
		info.IsScript = true
		// Read shebang line
		f.Seek(0, 0)
		shebangLine := make([]byte, 256)
		n, _ := f.Read(shebangLine)
		if n > 2 {
			// Find newline
			end := n
			for i := 2; i < n; i++ {
				if shebangLine[i] == '\n' || shebangLine[i] == '\r' {
					end = i
					break
				}
			}
			info.Shebang = string(shebangLine[2:end])
		}
		return info, nil
	}

	// Check for ELF magic number
	if header[0] == 0x7f && header[1] == 'E' && header[2] == 'L' && header[3] == 'F' {
		info.IsELF = true

		// Parse ELF file
		f.Seek(0, 0)
		elfFile, err := elf.Open(path)
		if err != nil {
			return info, fmt.Errorf("cannot parse ELF file: %w", err)
		}
		defer elfFile.Close()

		// Get ELF class
		switch elfFile.Class {
		case elf.ELFCLASS32:
			info.ELFClass = "32-bit"
		case elf.ELFCLASS64:
			info.ELFClass = "64-bit"
		default:
			info.ELFClass = "unknown"
		}

		// Get machine architecture
		info.Machine = elfFile.Machine.String()

		// Get interpreter (dynamic linker)
		for _, prog := range elfFile.Progs {
			if prog.Type == elf.PT_INTERP {
				// Read interpreter path
				interpBytes := make([]byte, prog.Filesz)
				_, err := prog.ReadAt(interpBytes, 0)
				if err == nil {
					// Remove null terminator
					interpPath := string(interpBytes)
					if idx := strings.IndexByte(interpPath, 0); idx >= 0 {
						interpPath = interpPath[:idx]
					}
					info.Interpreter = interpPath
				}
				break
			}
		}

		return info, nil
	}

	// Not a recognized format
	return info, nil
}

// CheckCompatibility checks if a binary is compatible with the host system
func CheckCompatibility(info *BinaryInfo) []CompatibilityIssue {
	var issues []CompatibilityIssue

	if !info.IsELF {
		// For non-ELF files (scripts, etc.), we can't do much checking
		if info.IsScript && info.Shebang != "" {
			// Check if shebang interpreter exists
			shebangParts := strings.Fields(info.Shebang)
			if len(shebangParts) > 0 {
				interpreterPath := shebangParts[0]
				if utils.FileNotExists(interpreterPath) {
					issues = append(issues, CompatibilityIssue{
						Severity: "error",
						Message:  fmt.Sprintf("Script interpreter not found: %s", interpreterPath),
						Hint:     fmt.Sprintf("Install the required interpreter or update the shebang line in %s", info.Path),
					})
				}
			}
		}
		return issues
	}

	// Check ELF class (32-bit vs 64-bit)
	hostBits := "64-bit"
	if osinfo.Arch() == "386" || osinfo.Arch() == "arm" {
		hostBits = "32-bit"
	}

	if info.ELFClass != "" && info.ELFClass != "unknown" && info.ELFClass != hostBits {
		issues = append(issues, CompatibilityIssue{
			Severity: "error",
			Message:  fmt.Sprintf("Binary is %s but host is %s", info.ELFClass, hostBits),
			Hint:     fmt.Sprintf("Rebuild the binary for %s architecture", hostBits),
		})
	}

	// Check interpreter (glibc vs musl)
	if info.Interpreter != "" {
		// Check if interpreter exists
		if utils.FileNotExists(info.Interpreter) {
			// Try to determine what's wrong
			isMusl := strings.Contains(info.Interpreter, "musl")
			isGlibc := strings.Contains(info.Interpreter, "ld-linux")

			var hint string
			if isGlibc {
				// Binary wants glibc but it's not available (probably on musl system like Alpine)
				hint = "This binary requires glibc but you may be on a musl-based system (like Alpine). "
				hint += "Try rebuilding with CGO_ENABLED=0 for a static binary, or install glibc compatibility packages."
			} else if isMusl {
				// Binary wants musl but it's not available
				hint = "This binary requires musl libc but it's not available. "
				hint += "Install musl or rebuild for your system's libc."
			} else {
				hint = fmt.Sprintf("Install the required dynamic linker: %s", info.Interpreter)
			}

			issues = append(issues, CompatibilityIssue{
				Severity: "error",
				Message:  fmt.Sprintf("Dynamic linker not found: %s", info.Interpreter),
				Hint:     hint,
			})
		} else {
			// Interpreter exists, but let's check if it's the expected type
			if osinfo.IsLinux() {
				expectedMusl := isMuslSystem()
				hasMusl := strings.Contains(info.Interpreter, "musl")

				if expectedMusl && !hasMusl {
					issues = append(issues, CompatibilityIssue{
						Severity: "warning",
						Message:  "Binary built for glibc but host uses musl libc",
						Hint:     "This may work but could have compatibility issues. Consider rebuilding with CGO_ENABLED=0 or for musl.",
					})
				} else if !expectedMusl && hasMusl {
					issues = append(issues, CompatibilityIssue{
						Severity: "warning",
						Message:  "Binary built for musl but host uses glibc",
						Hint:     "This may work but could have compatibility issues. Consider rebuilding for glibc.",
					})
				}
			}
		}
	}

	return issues
}

// LibcInfo contains information about the system's C library
type LibcInfo struct {
	Type    string // "glibc", "musl", "unknown"
	Version string // Version string if available
	Path    string // Path to the libc
}

// DetectLibc detects the system's C library type and version
func DetectLibc() *LibcInfo {
	if !osinfo.IsLinux() {
		return &LibcInfo{Type: "unknown"}
	}

	info := &LibcInfo{Type: "unknown"}

	// Method 1: Check for musl loader
	muslPaths := []string{
		"/lib/ld-musl-x86_64.so.1",
		"/lib/ld-musl-aarch64.so.1",
		"/lib/ld-musl-arm.so.1",
		"/lib/ld-musl-i386.so.1",
	}
	for _, path := range muslPaths {
		if utils.PathExists(path) {
			info.Type = "musl"
			info.Path = path
			// Try to get version by executing the loader
			// Most musl loaders print version when executed
			return info
		}
	}

	// Also check /lib directory for any ld-musl-* files
	if entries, err := os.ReadDir("/lib"); err == nil {
		for _, entry := range entries {
			if strings.HasPrefix(entry.Name(), "ld-musl-") {
				info.Type = "musl"
				info.Path = filepath.Join("/lib", entry.Name())
				return info
			}
		}
	}

	// Method 2: Check for glibc
	glibcPaths := []string{
		"/lib/x86_64-linux-gnu/libc.so.6",
		"/lib64/libc.so.6",
		"/lib/libc.so.6",
		"/lib/aarch64-linux-gnu/libc.so.6",
		"/lib/arm-linux-gnueabihf/libc.so.6",
	}
	for _, path := range glibcPaths {
		if utils.PathExists(path) {
			info.Type = "glibc"
			info.Path = path
			// Try to get version
			info.Version = getGlibcVersion(path)
			return info
		}
	}

	// Method 3: Check /etc/os-release for hints
	if data, err := os.ReadFile("/etc/os-release"); err == nil {
		content := string(data)
		if strings.Contains(content, "Alpine") {
			info.Type = "musl"
		} else if strings.Contains(content, "Void Linux") {
			// Void can be either glibc or musl
			if strings.Contains(content, "musl") {
				info.Type = "musl"
			}
		}
	}

	return info
}

// getGlibcVersion attempts to extract the glibc version
func getGlibcVersion(path string) string {
	// Try to read version from the library file
	// glibc libraries typically have version info embedded
	// For now, we'll just return "detected" as parsing ELF version info is complex
	// A full implementation would parse the .gnu.version_d section
	return "detected"
}

// isMuslSystem detects if the system is using musl libc
func isMuslSystem() bool {
	info := DetectLibc()
	return info.Type == "musl"
}

// FormatIssues formats compatibility issues for display
func FormatIssues(issues []CompatibilityIssue) string {
	if len(issues) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString("Binary compatibility issues detected:\n\n")

	for i, issue := range issues {
		// Icon based on severity
		var icon string
		switch issue.Severity {
		case "error":
			icon = utils.Emoji("❌ ")
		case "warning":
			icon = utils.Emoji("⚠️ ")
		case "info":
			icon = utils.Emoji("ℹ️ ")
		default:
			icon = "•"
		}

		sb.WriteString(fmt.Sprintf("%s%s\n", icon, issue.Message))
		if issue.Hint != "" {
			sb.WriteString(fmt.Sprintf("   %s%s\n", utils.Emoji("💡 "), issue.Hint))
		}
		if i < len(issues)-1 {
			sb.WriteString("\n")
		}
	}

	return sb.String()
}
