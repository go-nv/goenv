package toolupdater

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/manager"
	"github.com/go-nv/goenv/internal/utils"
)

// CompatibilityCheck represents a compatibility check result
type CompatibilityCheck struct {
	Compatible    bool
	Reason        string
	MinGoVersion  string
	MaxGoVersion  string
	RecommendedGo string
}

// CheckCompatibility checks if a tool version is compatible with a Go version
func CheckCompatibility(packagePath, toolVersion, goVersion string) (bool, string) {
	// Query module info to get go.mod requirements
	moduleInfo, err := queryModuleInfo(packagePath, toolVersion)
	if err != nil {
		// If we can't query, assume compatible (fail open)
		return true, ""
	}

	// Check if tool requires a minimum Go version
	if moduleInfo.GoVersion != "" {
		compatible, reason := checkGoVersionRequirement(moduleInfo.GoVersion, goVersion)
		if !compatible {
			return false, reason
		}
	}

	return true, ""
}

// CheckCompatibilityDetailed performs a detailed compatibility check
func CheckCompatibilityDetailed(packagePath, toolVersion, goVersion string) (*CompatibilityCheck, error) {
	result := &CompatibilityCheck{
		Compatible: true,
	}

	// Query module info
	moduleInfo, err := queryModuleInfo(packagePath, toolVersion)
	if err != nil {
		// If we can't query, assume compatible but note the reason
		result.Reason = fmt.Sprintf("unable to verify: %v", err)
		return result, nil
	}

	// Check Go version requirement from go.mod
	if moduleInfo.GoVersion != "" {
		result.MinGoVersion = moduleInfo.GoVersion
		compatible, reason := checkGoVersionRequirement(moduleInfo.GoVersion, goVersion)
		result.Compatible = compatible
		result.Reason = reason

		if !compatible {
			// Suggest minimum required version
			result.RecommendedGo = moduleInfo.GoVersion
		}
	}

	return result, nil
}

// ModuleInfo contains information about a Go module
type ModuleInfo struct {
	Path       string
	Version    string
	GoVersion  string // Minimum Go version from go.mod
	Time       string
	Deprecated string
}

// queryModuleInfo queries information about a module version
func queryModuleInfo(packagePath, version string) (*ModuleInfo, error) {
	// Use go list to query module information
	moduleQuery := packagePath
	if version != "" && version != manager.LatestVersion {
		moduleQuery = packagePath + "@" + version
	}

	output, err := utils.RunCommandOutput("go", "list", "-m", "-json", moduleQuery)
	if err != nil {
		return nil, errors.FailedTo("query module info", err)
	}

	var info ModuleInfo
	if err := json.Unmarshal([]byte(output), &info); err != nil {
		return nil, errors.FailedTo("parse module info", err)
	}

	return &info, nil
}

// checkGoVersionRequirement checks if the current Go version meets the requirement
func checkGoVersionRequirement(required, current string) (bool, string) {
	// Parse versions
	reqMajor, reqMinor, err := parseGoVersion(required)
	if err != nil {
		// Can't parse requirement, assume compatible
		return true, ""
	}

	curMajor, curMinor, err := parseGoVersion(current)
	if err != nil {
		// Can't parse current version, assume compatible
		return true, ""
	}

	// Compare versions
	if curMajor < reqMajor {
		return false, fmt.Sprintf("requires Go %s+ (you have Go %s)", required, current)
	}

	if curMajor == reqMajor && curMinor < reqMinor {
		return false, fmt.Sprintf("requires Go %s+ (you have Go %s)", required, current)
	}

	return true, ""
}

// parseGoVersion parses a Go version string into major and minor components
// Examples: "1.21", "1.21.5", "1.21rc1"
func parseGoVersion(version string) (major, minor int, err error) {
	// Remove 'go' prefix if present
	version = utils.NormalizeGoVersion(version)

	// Handle system version
	if version == manager.SystemVersion {
		// Get actual system Go version
		return getSystemGoVersion()
	}

	// Remove any rc/beta/alpha suffixes
	re := regexp.MustCompile(`^(\d+)\.(\d+)`)
	matches := re.FindStringSubmatch(version)
	if len(matches) < 3 {
		return 0, 0, fmt.Errorf("invalid Go version format: %s", version)
	}

	major, err = strconv.Atoi(matches[1])
	if err != nil {
		return 0, 0, err
	}

	minor, err = strconv.Atoi(matches[2])
	if err != nil {
		return 0, 0, err
	}

	return major, minor, nil
}

// getSystemGoVersion gets the version of the system Go installation
func getSystemGoVersion() (major, minor int, err error) {
	output, err := utils.RunCommandOutput("go", "version")
	if err != nil {
		return 0, 0, errors.FailedTo("get system Go version", err)
	}

	// Parse output like "go version go1.21.5 darwin/arm64"
	parts := strings.Fields(output)
	if len(parts) < 3 {
		return 0, 0, fmt.Errorf("unexpected go version output: %s", output)
	}

	return parseGoVersion(parts[2])
}

// IsGoVersionCompatible checks if two Go versions are compatible
// (within the same major.minor version)
func IsGoVersionCompatible(v1, v2 string) bool {
	major1, minor1, err1 := parseGoVersion(v1)
	major2, minor2, err2 := parseGoVersion(v2)

	if err1 != nil || err2 != nil {
		// If we can't parse, assume compatible
		return true
	}

	return major1 == major2 && minor1 == minor2
}

// GetCompatibleGoVersions returns a list of Go versions compatible with a tool
func GetCompatibleGoVersions(packagePath, toolVersion string, installedVersions []string) ([]string, error) {
	// Get tool requirements
	check, err := CheckCompatibilityDetailed(packagePath, toolVersion, "")
	if err != nil {
		return nil, err
	}

	if check.MinGoVersion == "" {
		// No specific requirement, all versions compatible
		return installedVersions, nil
	}

	// Filter versions that meet the minimum requirement
	compatible := []string{}
	for _, version := range installedVersions {
		if utils.CompareGoVersions(version, check.MinGoVersion) >= 0 {
			compatible = append(compatible, version)
		}
	}

	return compatible, nil
}

// ValidateToolVersion checks if a tool version string is valid
func ValidateToolVersion(version string) bool {
	if version == "" || version == manager.LatestVersion {
		return true
	}

	// Should start with 'v' and be a semver-like format
	if !strings.HasPrefix(version, "v") {
		return false
	}

	// Basic semver validation
	re := regexp.MustCompile(`^v\d+\.\d+\.\d+`)
	return re.MatchString(version)
}

// SuggestGoVersionForTool suggests the best Go version to use for a tool
func SuggestGoVersionForTool(packagePath, toolVersion string, installedVersions []string) (string, error) {
	// Get compatible versions
	compatible, err := GetCompatibleGoVersions(packagePath, toolVersion, installedVersions)
	if err != nil {
		return "", err
	}

	if len(compatible) == 0 {
		return "", fmt.Errorf("no compatible Go versions installed")
	}

	// Return the latest compatible version
	latest := compatible[0]
	for _, version := range compatible[1:] {
		if utils.CompareGoVersions(version, latest) > 0 {
			latest = version
		}
	}

	return latest, nil
}
