package envdetect

import (
	"os"
	"os/exec"
	"runtime"
	"strings"

	"github.com/go-nv/goenv/internal/utils"
)

// IsAppleSilicon detects if running on Apple Silicon (M1/M2/M3/etc)
func IsAppleSilicon() bool {
	if runtime.GOOS != "darwin" {
		return false
	}

	// Check for arm64 support via sysctl
	cmd := exec.Command("sysctl", "-n", "hw.optional.arm64")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) == "1"
}

// GetBinaryArchitecture returns the architecture of a binary file
// Returns "arm64", "x86_64", or "" if unable to determine
func GetBinaryArchitecture(binaryPath string) string {
	if runtime.GOOS != "darwin" {
		return ""
	}

	// Use 'file' command to check binary architecture
	cmd := exec.Command("file", binaryPath)
	output, err := cmd.Output()
	if err != nil {
		return ""
	}

	fileStr := string(output)

	// Check for arm64
	if strings.Contains(fileStr, "arm64") {
		return "arm64"
	}

	// Check for x86_64
	if strings.Contains(fileStr, "x86_64") {
		return "x86_64"
	}

	return ""
}

// CheckRosettaMixedArchitecture checks if we're mixing native arm64 and Rosetta x86_64
// Returns a warning message if there's a problematic mix, empty string otherwise
func CheckRosettaMixedArchitecture(toolPath string) string {
	if !IsAppleSilicon() {
		return ""
	}

	// Get goenv binary architecture
	goenvBinary, err := os.Executable()
	if err != nil {
		return ""
	}

	goenvArch := GetBinaryArchitecture(goenvBinary)
	toolArch := GetBinaryArchitecture(toolPath)

	// If we can't determine either, don't warn
	if goenvArch == "" || toolArch == "" {
		return ""
	}

	// Check for architecture mismatch
	if goenvArch != toolArch {
		if goenvArch == "arm64" && toolArch == "x86_64" {
			return utils.Emoji("⚠️  ") + "Mixing architectures: goenv is native arm64 but tool is x86_64 (will run under Rosetta).\n" +
				"   This may cause performance issues and cache conflicts.\n" +
				"   Consider: Rebuilding the tool with native arm64 Go version"
		}

		if goenvArch == "x86_64" && toolArch == "arm64" {
			return utils.Emoji("⚠️  ") + "Architecture mismatch: goenv is x86_64 (Rosetta) but tool is native arm64.\n" +
				"   This unusual configuration may cause issues.\n" +
				"   Consider: Reinstalling goenv as native arm64: brew reinstall goenv"
		}
	}

	// Warn if both are x86_64 on Apple Silicon
	if goenvArch == "x86_64" && toolArch == "x86_64" {
		return utils.Emoji("ℹ️  ") + "Both goenv and tool are x86_64 (running under Rosetta on Apple Silicon).\n" +
			"   Consider: Using native arm64 versions for better performance"
	}

	return ""
}

// IsRosetta detects if the current process is running under Rosetta 2 translation
func IsRosetta() bool {
	if runtime.GOOS != "darwin" {
		return false
	}

	// Check if running under Rosetta 2 translation
	// sysctl.proc_translated returns 1 when under Rosetta, 0 for native, or error if not applicable
	cmd := exec.Command("sysctl", "-n", "sysctl.proc_translated")
	output, err := cmd.Output()
	if err != nil {
		return false
	}

	return strings.TrimSpace(string(output)) == "1"
}
