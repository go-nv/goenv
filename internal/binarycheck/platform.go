package binarycheck

import (
	"debug/macho"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// PlatformInfo contains platform-specific information
type PlatformInfo struct {
	OS      string
	Arch    string
	Details map[string]string
}

// macOSInfo contains macOS-specific information
type macOSInfo struct {
	DeploymentTarget string // Minimum macOS version required
	SDKVersion       string // SDK version used to build
	Platform         string // macosx, iphoneos, etc.
	HasVersionMin    bool   // Has LC_VERSION_MIN_* load command
}

// WindowsInfo contains Windows-specific information
type WindowsInfo struct {
	Compiler     string // "MSVC", "MinGW", "unknown"
	HasCLExe     bool   // cl.exe available
	HasVCRuntime bool   // Visual C++ runtime available
	ProcessMode  string // "ARM64", "ARM64EC", "x64", "x86"
	IsARM64EC    bool   // Running in ARM64EC mode
}

// LinuxInfo contains Linux-specific information
type LinuxInfo struct {
	KernelVersion string
	KernelMajor   int
	KernelMinor   int
	KernelPatch   int
	GlibcVersion  string
	LibcType      string // "glibc" or "musl"
}

// CheckMacOSDeploymentTarget checks macOS binary compatibility
func CheckMacOSDeploymentTarget(binaryPath string) (*macOSInfo, []CompatibilityIssue) {
	if runtime.GOOS != "darwin" {
		return nil, nil
	}

	info := &macOSInfo{}
	issues := []CompatibilityIssue{}

	// Open Mach-O file
	file, err := macho.Open(binaryPath)
	if err != nil {
		// Not a Mach-O binary
		return nil, nil
	}
	defer file.Close()

	// Check load commands for version min
	// We can't directly check LC_VERSION_MIN_* as the types are not exported
	// Instead, we'll use otool to check the binary
	cmd := exec.Command("otool", "-l", binaryPath)
	output, err := cmd.Output()
	if err == nil {
		outputStr := string(output)

		// Look for LC_VERSION_MIN commands
		if strings.Contains(outputStr, "LC_VERSION_MIN_MACOSX") {
			info.HasVersionMin = true
			// Parse version from output
			lines := strings.Split(outputStr, "\n")
			for i, line := range lines {
				if strings.Contains(line, "LC_VERSION_MIN_MACOSX") {
					// Look ahead for version line
					for j := i + 1; j < len(lines) && j < i+5; j++ {
						if strings.Contains(lines[j], "version") {
							// Extract version (format: "version 10.15")
							fields := strings.Fields(lines[j])
							for k, field := range fields {
								if field == "version" && k+1 < len(fields) {
									info.DeploymentTarget = fields[k+1]
									break
								}
							}
							break
						}
					}
					break
				}
			}
		} else if strings.Contains(outputStr, "LC_BUILD_VERSION") {
			// Newer format (macOS 10.14+)
			info.HasVersionMin = true
			lines := strings.Split(outputStr, "\n")
			for i, line := range lines {
				if strings.Contains(line, "LC_BUILD_VERSION") {
					// Look ahead for minos line
					for j := i + 1; j < len(lines) && j < i+10; j++ {
						if strings.Contains(lines[j], "minos") {
							// Extract minos version
							fields := strings.Fields(lines[j])
							for k, field := range fields {
								if field == "minos" && k+1 < len(fields) {
									info.DeploymentTarget = fields[k+1]
									break
								}
							}
							break
						}
					}
					break
				}
			}
		}
	}

	// Check against current OS version if we found a deployment target
	if info.DeploymentTarget != "" {
		currentVersion := getCurrentMacOSVersion()
		if currentVersion != "" {
			if !isVersionCompatible(currentVersion, info.DeploymentTarget) {
				issues = append(issues, CompatibilityIssue{
					Severity: "warning",
					Message:  fmt.Sprintf("Binary requires macOS %s but current version is %s", info.DeploymentTarget, currentVersion),
					Hint:     "Binary may not run on this system. Consider rebuilding with a lower MACOSX_DEPLOYMENT_TARGET.",
				})
			}
		}
	}

	// Check MACOSX_DEPLOYMENT_TARGET environment variable
	deployTarget := os.Getenv("MACOSX_DEPLOYMENT_TARGET")
	if deployTarget != "" && info.DeploymentTarget != "" {
		if deployTarget != info.DeploymentTarget {
			issues = append(issues, CompatibilityIssue{
				Severity: "info",
				Message:  fmt.Sprintf("MACOSX_DEPLOYMENT_TARGET=%s but binary was built for %s", deployTarget, info.DeploymentTarget),
				Hint:     "Ensure consistent deployment target settings to avoid compatibility issues.",
			})
		}
	}

	if !info.HasVersionMin && info.DeploymentTarget == "" {
		issues = append(issues, CompatibilityIssue{
			Severity: "info",
			Message:  "Binary has no macOS version minimum requirement",
			Hint:     "Set MACOSX_DEPLOYMENT_TARGET when building to ensure compatibility.",
		})
	}

	return info, issues
}

// getCurrentMacOSVersion gets the current macOS version
func getCurrentMacOSVersion() string {
	cmd := exec.Command("sw_vers", "-productVersion")
	output, err := cmd.Output()
	if err != nil {
		return ""
	}
	return strings.TrimSpace(string(output))
}

// isVersionCompatible checks if current version meets minimum requirement
func isVersionCompatible(current, minimum string) bool {
	currentParts := strings.Split(current, ".")
	minimumParts := strings.Split(minimum, ".")

	for i := 0; i < 3; i++ {
		var c, m int
		if i < len(currentParts) {
			c, _ = strconv.Atoi(currentParts[i])
		}
		if i < len(minimumParts) {
			m, _ = strconv.Atoi(minimumParts[i])
		}

		if c > m {
			return true
		}
		if c < m {
			return false
		}
	}
	return true
}

// CheckWindowsCompiler detects Windows compiler and runtime availability
func CheckWindowsCompiler() (*WindowsInfo, []CompatibilityIssue) {
	if runtime.GOOS != "windows" {
		return nil, nil
	}

	info := &WindowsInfo{}
	issues := []CompatibilityIssue{}

	// Check for cl.exe (MSVC)
	clPath, err := exec.LookPath("cl.exe")
	if err == nil {
		info.HasCLExe = true
		info.Compiler = "MSVC"

		// Try to get MSVC version
		cmd := exec.Command(clPath)
		output, err := cmd.CombinedOutput()
		if err == nil {
			outputStr := string(output)
			if strings.Contains(outputStr, "Microsoft") {
				info.Compiler = "MSVC"
			}
		}
	} else {
		// Check for gcc/g++ (MinGW)
		if _, err := exec.LookPath("gcc.exe"); err == nil {
			info.Compiler = "MinGW"
		} else if _, err := exec.LookPath("g++.exe"); err == nil {
			info.Compiler = "MinGW"
		} else {
			info.Compiler = "unknown"
			issues = append(issues, CompatibilityIssue{
				Severity: "warning",
				Message:  "No C compiler detected (neither MSVC nor MinGW)",
				Hint:     "CGO-based builds will fail. Install Visual Studio Build Tools or MinGW-w64.",
			})
		}
	}

	// Check for Visual C++ runtime
	systemRoot := os.Getenv("SystemRoot")
	if systemRoot != "" {
		vcRuntimePaths := []string{
			filepath.Join(systemRoot, "System32", "vcruntime140.dll"),
			filepath.Join(systemRoot, "System32", "msvcp140.dll"),
		}
		hasAnyRuntime := false
		for _, path := range vcRuntimePaths {
			if _, err := os.Stat(path); err == nil {
				hasAnyRuntime = true
				break
			}
		}
		info.HasVCRuntime = hasAnyRuntime

		if !hasAnyRuntime {
			issues = append(issues, CompatibilityIssue{
				Severity: "info",
				Message:  "Visual C++ runtime not detected",
				Hint:     "Some binaries may require VC++ redistributable. Install from microsoft.com/cpp if needed.",
			})
		}
	}

	return info, issues
}

// CheckWindowsARM64 checks Windows ARM64/ARM64EC process mode
func CheckWindowsARM64() (*WindowsInfo, []CompatibilityIssue) {
	if runtime.GOOS != "windows" {
		return nil, nil
	}

	info := &WindowsInfo{}
	issues := []CompatibilityIssue{}

	// Detect current architecture
	info.ProcessMode = runtime.GOARCH

	// Check if running under emulation
	if runtime.GOARCH == "amd64" {
		// Check if we're on ARM64 hardware running x64 emulation
		cmd := exec.Command("cmd", "/c", "echo", "%PROCESSOR_ARCHITECTURE%")
		output, err := cmd.Output()
		if err == nil {
			arch := strings.TrimSpace(string(output))
			if arch == "ARM64" {
				info.ProcessMode = "x64-on-ARM64"
				issues = append(issues, CompatibilityIssue{
					Severity: "info",
					Message:  "Running x64 binaries on ARM64 via emulation",
					Hint:     "For better performance, use native ARM64 Go binaries. Install arm64 version with: goenv install <version>",
				})
			}
		}
	}

	// ARM64EC detection (Windows 11 22H2+)
	if runtime.GOARCH == "arm64" {
		// ARM64EC allows mixing ARM64 and x64 code in the same process
		// This is detected by checking for certain environment indicators
		programFiles := os.Getenv("ProgramFiles(Arm)")
		if programFiles != "" {
			info.IsARM64EC = true
			issues = append(issues, CompatibilityIssue{
				Severity: "info",
				Message:  "ARM64EC mode available (can run x64 and ARM64 code)",
				Hint:     "Both ARM64 and x64 Go versions will work, but ARM64 native is recommended for best performance.",
			})
		}
	}

	return info, issues
}

// CheckLinuxKernelVersion checks Linux kernel version compatibility
func CheckLinuxKernelVersion() (*LinuxInfo, []CompatibilityIssue) {
	if runtime.GOOS != "linux" {
		return nil, nil
	}

	info := &LinuxInfo{}
	issues := []CompatibilityIssue{}

	// Get kernel version
	cmd := exec.Command("uname", "-r")
	output, err := cmd.Output()
	if err != nil {
		return nil, nil
	}

	info.KernelVersion = strings.TrimSpace(string(output))

	// Parse version (e.g., "5.15.0-91-generic")
	parts := strings.Split(info.KernelVersion, ".")
	if len(parts) >= 2 {
		info.KernelMajor, _ = strconv.Atoi(parts[0])
		info.KernelMinor, _ = strconv.Atoi(parts[1])
		if len(parts) >= 3 {
			// Extract patch number (strip any suffix like "-generic")
			patchParts := strings.Split(parts[2], "-")
			info.KernelPatch, _ = strconv.Atoi(patchParts[0])
		}
	}

	// Check for old kernels
	if info.KernelMajor < 3 || (info.KernelMajor == 3 && info.KernelMinor < 10) {
		issues = append(issues, CompatibilityIssue{
			Severity: "warning",
			Message:  fmt.Sprintf("Old Linux kernel detected: %s", info.KernelVersion),
			Hint:     "Kernel < 3.10 may have issues with modern Go binaries. Consider upgrading or using statically linked binaries (CGO_ENABLED=0).",
		})
	}

	// Check for very old kernels with static binaries
	if info.KernelMajor < 2 || (info.KernelMajor == 2 && info.KernelMinor < 6) {
		issues = append(issues, CompatibilityIssue{
			Severity: "error",
			Message:  fmt.Sprintf("Very old Linux kernel: %s", info.KernelVersion),
			Hint:     "Kernel < 2.6 is not supported by modern Go. Upgrade your kernel or use Go 1.4 or earlier.",
		})
	}

	// Get libc info
	libcInfo := DetectLibc()
	info.LibcType = libcInfo.Type
	info.GlibcVersion = libcInfo.Version

	return info, issues
}

// CheckWindowsScriptShims checks if scripts need .cmd/.ps1 wrapper shims
func CheckWindowsScriptShims(scriptPath string) []CompatibilityIssue {
	if runtime.GOOS != "windows" {
		return nil
	}

	issues := []CompatibilityIssue{}

	// Check if file is a script (has shebang)
	data, err := os.ReadFile(scriptPath)
	if err != nil {
		return nil
	}

	// Check for shebang
	if len(data) >= 2 && data[0] == '#' && data[1] == '!' {
		// This is a script file
		ext := strings.ToLower(filepath.Ext(scriptPath))
		if ext != ".cmd" && ext != ".bat" && ext != ".ps1" {
			issues = append(issues, CompatibilityIssue{
				Severity: "warning",
				Message:  fmt.Sprintf("Script file without Windows extension: %s", filepath.Base(scriptPath)),
				Hint:     "Create a .cmd or .ps1 wrapper shim for this script to make it executable on Windows. Example: " + filepath.Base(scriptPath) + ".cmd",
			})
		}

		// Check shebang interpreter
		shebangEnd := 0
		for i := 2; i < len(data) && i < 256; i++ {
			if data[i] == '\n' || data[i] == '\r' {
				shebangEnd = i
				break
			}
		}
		if shebangEnd > 2 {
			shebang := string(data[2:shebangEnd])
			if strings.Contains(shebang, "/bin/bash") || strings.Contains(shebang, "/bin/sh") {
				issues = append(issues, CompatibilityIssue{
					Severity: "info",
					Message:  "Script requires bash/sh interpreter",
					Hint:     "Install Git Bash, WSL, or use the script from a Unix-like environment.",
				})
			} else if strings.Contains(shebang, "python") {
				issues = append(issues, CompatibilityIssue{
					Severity: "info",
					Message:  "Script requires Python interpreter",
					Hint:     "Ensure Python is installed and in PATH.",
				})
			}
		}
	}

	return issues
}

// SuggestGlibcCompatibility suggests remediation for glibc version mismatches
func SuggestGlibcCompatibility(requiredGlibc, currentGlibc string) []CompatibilityIssue {
	if runtime.GOOS != "linux" {
		return nil
	}

	issues := []CompatibilityIssue{}

	// Parse versions
	requiredParts := parseVersion(requiredGlibc)
	currentParts := parseVersion(currentGlibc)

	if len(requiredParts) < 2 || len(currentParts) < 2 {
		return nil
	}

	// Compare versions
	reqMajor, reqMinor := requiredParts[0], requiredParts[1]
	curMajor, curMinor := currentParts[0], currentParts[1]

	if reqMajor > curMajor || (reqMajor == curMajor && reqMinor > curMinor) {
		issues = append(issues, CompatibilityIssue{
			Severity: "error",
			Message:  fmt.Sprintf("Binary requires glibc %s but system has %s", requiredGlibc, currentGlibc),
			Hint:     fmt.Sprintf("Build in a container with older glibc: docker run -v $PWD:/app -w /app ubuntu:18.04 go build"),
		})

		issues = append(issues, CompatibilityIssue{
			Severity: "info",
			Message:  "Alternative: Use static linking",
			Hint:     "Build with CGO_ENABLED=0 to create a statically linked binary that doesn't depend on glibc version.",
		})

		issues = append(issues, CompatibilityIssue{
			Severity: "info",
			Message:  "Alternative: Use older distro container",
			Hint:     "Common choices: debian:buster (glibc 2.28), ubuntu:18.04 (glibc 2.27), centos:7 (glibc 2.17)",
		})
	}

	return issues
}

// parseVersion parses a version string like "2.31" into []int{2, 31}
func parseVersion(version string) []int {
	parts := strings.Split(version, ".")
	result := make([]int, 0, len(parts))
	for _, part := range parts {
		if num, err := strconv.Atoi(part); err == nil {
			result = append(result, num)
		}
	}
	return result
}

// GetPlatformInfo returns comprehensive platform information
func GetPlatformInfo() *PlatformInfo {
	info := &PlatformInfo{
		OS:      runtime.GOOS,
		Arch:    runtime.GOARCH,
		Details: make(map[string]string),
	}

	switch runtime.GOOS {
	case "darwin":
		if version := getCurrentMacOSVersion(); version != "" {
			info.Details["macos_version"] = version
		}
		if target := os.Getenv("MACOSX_DEPLOYMENT_TARGET"); target != "" {
			info.Details["deployment_target"] = target
		}

	case "windows":
		winInfo, _ := CheckWindowsCompiler()
		if winInfo != nil {
			info.Details["compiler"] = winInfo.Compiler
			info.Details["process_mode"] = winInfo.ProcessMode
			if winInfo.IsARM64EC {
				info.Details["arm64ec"] = "true"
			}
		}

	case "linux":
		linuxInfo, _ := CheckLinuxKernelVersion()
		if linuxInfo != nil {
			info.Details["kernel_version"] = linuxInfo.KernelVersion
			info.Details["libc_type"] = linuxInfo.LibcType
			if linuxInfo.GlibcVersion != "" {
				info.Details["glibc_version"] = linuxInfo.GlibcVersion
			}
		}
	}

	return info
}
