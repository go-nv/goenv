package goenv

import (
	"encoding/json"
	"os/exec"
	"strings"
	"sync"
)

// ABIVariable represents a Go ABI-related environment variable
type ABIVariable struct {
	Name         string
	DefaultValue string
}

// Known ABI variable patterns - used as fallback if go env fails
var knownABIVars = []ABIVariable{
	{"GOAMD64", "v1"},
	{"GOARM", "7"},
	{"GO386", "sse2"},
	{"GOMIPS", "hardfloat"},
	{"GOMIPS64", "hardfloat"},
	{"GOPPC64", "power8"},
	{"GORISCV64", "rva20u64"},
	{"GOWASM", ""},
}

var (
	abiVarsCache      []ABIVariable
	abiVarsCacheMutex sync.RWMutex
)

// GetABIVariables returns all ABI-related environment variables supported by the given Go binary.
// It caches results per Go version for performance.
func GetABIVariables(goBinaryPath string) ([]ABIVariable, error) {
	// Check cache first
	abiVarsCacheMutex.RLock()
	if abiVarsCache != nil {
		defer abiVarsCacheMutex.RUnlock()
		return abiVarsCache, nil
	}
	abiVarsCacheMutex.RUnlock()

	// Run go env -json to get all environment variables
	cmd := exec.Command(goBinaryPath, "env", "-json")
	output, err := cmd.Output()
	if err != nil {
		// Fallback to known variables if go env fails
		return knownABIVars, nil
	}

	// Parse JSON output
	var envVars map[string]string
	if err := json.Unmarshal(output, &envVars); err != nil {
		return knownABIVars, nil
	}

	// Extract ABI-related variables
	// These are variables that affect codegen/ABI but aren't GOOS/GOARCH
	var abiVars []ABIVariable

	// Known patterns for ABI variables
	abiPrefixes := []string{"GOAMD64", "GOARM", "GO386", "GOMIPS", "GOPPC64", "GORISCV64", "GOWASM"}

	for key, value := range envVars {
		// Check if this is an ABI variable
		isABI := false
		for _, prefix := range abiPrefixes {
			if strings.HasPrefix(key, prefix) {
				isABI = true
				break
			}
		}

		if isABI {
			abiVars = append(abiVars, ABIVariable{
				Name:         key,
				DefaultValue: value,
			})
		}
	}

	// If we found variables, cache them
	if len(abiVars) > 0 {
		abiVarsCacheMutex.Lock()
		abiVarsCache = abiVars
		abiVarsCacheMutex.Unlock()
		return abiVars, nil
	}

	// Fallback to known variables
	return knownABIVars, nil
}

// GetABIValue returns the value of an ABI variable from the environment slice,
// or the default value if not set.
func GetABIValue(env []string, abiVar ABIVariable) string {
	prefix := abiVar.Name + "="
	for _, envVar := range env {
		if strings.HasPrefix(envVar, prefix) {
			return strings.TrimPrefix(envVar, prefix)
		}
	}
	return abiVar.DefaultValue
}

// BuildABISuffix constructs the ABI portion of a cache directory name.
// It uses the Go binary to discover supported ABI variables dynamically.
func BuildABISuffix(goBinaryPath string, goarch string, env []string) string {
	// Get ABI variables for this Go version
	abiVars, err := GetABIVariables(goBinaryPath)
	if err != nil {
		// Fallback to manual detection if discovery fails
		return buildABISuffixFallback(goarch, env)
	}

	var suffix string

	// Check each ABI variable
	for _, abiVar := range abiVars {
		// Only include if:
		// 1. The variable affects this architecture
		// 2. The value is different from the default
		if isRelevantForArch(abiVar.Name, goarch) {
			value := GetABIValue(env, abiVar)
			if value != "" && value != abiVar.DefaultValue {
				suffix += "-" + sanitizeABIValue(value)
			}
		}
	}

	return suffix
}

// isRelevantForArch checks if an ABI variable is relevant for the given architecture
func isRelevantForArch(abiVarName, goarch string) bool {
	relevance := map[string][]string{
		"GOAMD64":   {"amd64"},
		"GOARM":     {"arm"},
		"GO386":     {"386"},
		"GOMIPS":    {"mips", "mipsle"},
		"GOMIPS64":  {"mips64", "mips64le"},
		"GOPPC64":   {"ppc64", "ppc64le"},
		"GORISCV64": {"riscv64"},
		"GOWASM":    {"wasm"},
		"GOLOONG64": {"loong64"}, // Future-proofing
	}

	arches, exists := relevance[abiVarName]
	if !exists {
		return false
	}

	for _, arch := range arches {
		if arch == goarch {
			return true
		}
	}

	return false
}

// sanitizeABIValue makes ABI values safe for use in file paths
func sanitizeABIValue(value string) string {
	// Replace problematic characters
	value = strings.ReplaceAll(value, ",", "-")
	value = strings.ReplaceAll(value, " ", "-")
	value = strings.ReplaceAll(value, "/", "-")
	return value
}

// buildABISuffixFallback is the old hardcoded approach, used as fallback
func buildABISuffixFallback(goarch string, env []string) string {
	var suffix string

	switch goarch {
	case "amd64":
		if amd64Level := getEnvValue(env, "GOAMD64"); amd64Level != "" && amd64Level != "v1" {
			suffix += "-" + amd64Level
		}
	case "arm":
		if armVersion := getEnvValue(env, "GOARM"); armVersion != "" && armVersion != "7" {
			suffix += "-v" + armVersion
		}
	case "386":
		if arch386 := getEnvValue(env, "GO386"); arch386 != "" && arch386 != "sse2" {
			suffix += "-" + arch386
		}
	case "mips", "mipsle":
		if mipsABI := getEnvValue(env, "GOMIPS"); mipsABI != "" && mipsABI != "hardfloat" {
			suffix += "-" + mipsABI
		}
	case "mips64", "mips64le":
		if mips64ABI := getEnvValue(env, "GOMIPS64"); mips64ABI != "" && mips64ABI != "hardfloat" {
			suffix += "-" + mips64ABI
		}
	case "ppc64", "ppc64le":
		if ppc64Level := getEnvValue(env, "GOPPC64"); ppc64Level != "" && ppc64Level != "power8" {
			suffix += "-" + ppc64Level
		}
	case "riscv64":
		if riscv64Level := getEnvValue(env, "GORISCV64"); riscv64Level != "" && riscv64Level != "rva20u64" {
			suffix += "-" + riscv64Level
		}
	case "wasm":
		if wasmABI := getEnvValue(env, "GOWASM"); wasmABI != "" {
			suffix += "-" + wasmABI
		}
	}

	return suffix
}

func getEnvValue(env []string, key string) string {
	prefix := key + "="
	for _, envVar := range env {
		if strings.HasPrefix(envVar, prefix) {
			return strings.TrimPrefix(envVar, prefix)
		}
	}
	return ""
}

// ClearCache clears the cached ABI variables (useful for testing)
func ClearCache() {
	abiVarsCacheMutex.Lock()
	abiVarsCache = nil
	abiVarsCacheMutex.Unlock()
}
