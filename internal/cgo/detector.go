package cgo

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-nv/goenv/internal/utils"
)

// BuildInfo contains metadata about the CGO toolchain used for a cache
type BuildInfo struct {
	Created       time.Time `json:"created"`
	CC            string    `json:"cc,omitempty"`
	CCVersion     string    `json:"cc_version,omitempty"`
	CXX           string    `json:"cxx,omitempty"`
	CXXVersion    string    `json:"cxx_version,omitempty"`
	CFLAGS        string    `json:"cflags,omitempty"`
	CXXFLAGS      string    `json:"cxxflags,omitempty"`
	LDFLAGS       string    `json:"ldflags,omitempty"`
	PKGConfig     string    `json:"pkg_config,omitempty"`
	PKGConfigPath string    `json:"pkg_config_path,omitempty"`
	CGOEnabled    string    `json:"cgo_enabled,omitempty"`
	Sysroot       string    `json:"sysroot,omitempty"`
	ToolchainHash string    `json:"toolchain_hash"`
}

// IsCGOEnabled checks if CGO is enabled in the environment
func IsCGOEnabled(env []string) bool {
	cgoEnabled := utils.GetEnvValue(env, "CGO_ENABLED")
	// CGO is enabled by default if not explicitly disabled
	return cgoEnabled != "0"
}

// ComputeToolchainHash computes a hash of CGO-relevant environment
func ComputeToolchainHash(env []string) string {
	if !IsCGOEnabled(env) {
		return ""
	}

	var components []string

	// Collect CGO-relevant environment variables
	cgoCriticalVars := []string{
		"CC", "CXX", "CFLAGS", "CXXFLAGS", "LDFLAGS",
		"PKG_CONFIG", "PKG_CONFIG_PATH", "PKG_CONFIG_LIBDIR",
		"CGO_CFLAGS", "CGO_CXXFLAGS", "CGO_LDFLAGS",
		"AR", "SYSROOT",
	}

	for _, key := range cgoCriticalVars {
		if val := utils.GetEnvValue(env, key); val != "" {
			components = append(components, key+"="+val)
		}
	}

	// Get compiler version strings
	if ccPath := utils.GetEnvValue(env, "CC"); ccPath != "" {
		if version := getCompilerVersion(ccPath); version != "" {
			components = append(components, "CC_VERSION="+version)
		}
	}

	if cxxPath := utils.GetEnvValue(env, "CXX"); cxxPath != "" {
		if version := getCompilerVersion(cxxPath); version != "" {
			components = append(components, "CXX_VERSION="+version)
		}
	}

	// If no components, no CGO in use
	if len(components) == 0 {
		return ""
	}

	// Hash all components
	h := sha256.New()
	for _, c := range components {
		h.Write([]byte(c + "\n"))
	}
	return hex.EncodeToString(h.Sum(nil))
}

// GetBuildInfo extracts full CGO toolchain information
func GetBuildInfo(env []string) *BuildInfo {
	info := &BuildInfo{
		Created:    time.Now(),
		CGOEnabled: utils.GetEnvValue(env, "CGO_ENABLED"),
	}

	if !IsCGOEnabled(env) {
		return info
	}

	// Collect environment variables
	info.CC = utils.GetEnvValue(env, "CC")
	info.CXX = utils.GetEnvValue(env, "CXX")
	info.CFLAGS = utils.GetEnvValue(env, "CFLAGS")
	info.CXXFLAGS = utils.GetEnvValue(env, "CXXFLAGS")
	info.LDFLAGS = utils.GetEnvValue(env, "LDFLAGS")
	info.PKGConfig = utils.GetEnvValue(env, "PKG_CONFIG")
	info.PKGConfigPath = utils.GetEnvValue(env, "PKG_CONFIG_PATH")
	info.Sysroot = utils.GetEnvValue(env, "SYSROOT")

	// Get compiler versions
	if info.CC != "" {
		info.CCVersion = getCompilerVersion(info.CC)
	}
	if info.CXX != "" {
		info.CXXVersion = getCompilerVersion(info.CXX)
	}

	// Compute hash
	info.ToolchainHash = ComputeToolchainHash(env)

	return info
}

// WriteBuildInfo writes build info to a JSON file
func WriteBuildInfo(cacheDir string, info *BuildInfo) error {
	infoPath := filepath.Join(cacheDir, "build.info")

	// Create cache directory if it doesn't exist
	if err := os.MkdirAll(cacheDir, 0755); err != nil {
		return fmt.Errorf("failed to create cache directory: %w", err)
	}

	data, err := json.MarshalIndent(info, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal build info: %w", err)
	}

	if err := os.WriteFile(infoPath, data, 0644); err != nil {
		return fmt.Errorf("failed to write build info: %w", err)
	}

	return nil
}

// ReadBuildInfo reads build info from a cache directory
func ReadBuildInfo(cacheDir string) (*BuildInfo, error) {
	infoPath := filepath.Join(cacheDir, "build.info")

	data, err := os.ReadFile(infoPath)
	if err != nil {
		return nil, err
	}

	var info BuildInfo
	if err := json.Unmarshal(data, &info); err != nil {
		return nil, err
	}

	return &info, nil
}

// getCompilerVersion runs the compiler with --version and extracts version info
func getCompilerVersion(compilerPath string) string {
	// Handle cases where CC might be "ccache gcc" or similar
	parts := strings.Fields(compilerPath)
	if len(parts) == 0 {
		return ""
	}

	// Try to find the actual compiler (skip ccache, distcc, etc.)
	actualCompiler := parts[0]
	if strings.Contains(actualCompiler, "ccache") || strings.Contains(actualCompiler, "distcc") {
		if len(parts) > 1 {
			actualCompiler = parts[1]
		}
	}

	// Run compiler --version
	cmd := exec.Command(actualCompiler, "--version")
	output, err := cmd.Output()
	if err != nil {
		// Try -v for some compilers (like clang on macOS)
		cmd = exec.Command(actualCompiler, "-v")
		output, err = cmd.CombinedOutput()
		if err != nil {
			return ""
		}
	}

	// Extract first line which usually contains version
	lines := strings.Split(string(output), "\n")
	if len(lines) > 0 {
		version := strings.TrimSpace(lines[0])
		// Limit length to avoid huge version strings
		if len(version) > 200 {
			version = version[:200]
		}
		return version
	}

	return ""
}
