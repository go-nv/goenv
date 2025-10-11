package main

import (
	"encoding/json"
	"fmt"
	"os"
	"sort"
	"strconv"
	"strings"
)

// This utility generates embedded version data from the legacy build definitions
func main() {
	if len(os.Args) < 2 {
		fmt.Println("Usage: go run generate_embedded_versions.go <path-to-go-build-share>")
		os.Exit(1)
	}

	buildPath := os.Args[1]

	entries, err := os.ReadDir(buildPath)
	if err != nil {
		fmt.Printf("Error reading directory: %v\n", err)
		os.Exit(1)
	}

	type GoFile struct {
		Filename string `json:"filename"`
		OS       string `json:"os"`
		Arch     string `json:"arch"`
		Version  string `json:"version"`
		SHA256   string `json:"sha256"`
		Size     int64  `json:"size"`
		Kind     string `json:"kind"`
	}

	type GoRelease struct {
		Version string   `json:"version"`
		Stable  bool     `json:"stable"`
		Files   []GoFile `json:"files"`
	}

	var releases []GoRelease

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		version := entry.Name()

		// Skip non-standard version formats
		if strings.Contains(version, "beta") || strings.Contains(version, "rc") {
			continue
		}

		// Parse version to determine if it's recent enough (1.19+)
		versionParts := strings.Split(version, ".")
		if len(versionParts) < 2 {
			continue
		}

		minorInt, err := strconv.Atoi(versionParts[1])
		if err != nil || minorInt < 19 {
			continue // Skip versions older than 1.19
		}

		// Create a basic release entry (we'll use placeholder data since we can't easily parse the build files)
		release := GoRelease{
			Version: "go" + version,
			Stable:  true,
			Files: []GoFile{
				{Filename: fmt.Sprintf("go%s.darwin-amd64.tar.gz", version), OS: "darwin", Arch: "amd64", Kind: "archive", SHA256: "placeholder"},
				{Filename: fmt.Sprintf("go%s.darwin-arm64.tar.gz", version), OS: "darwin", Arch: "arm64", Kind: "archive", SHA256: "placeholder"},
				{Filename: fmt.Sprintf("go%s.linux-amd64.tar.gz", version), OS: "linux", Arch: "amd64", Kind: "archive", SHA256: "placeholder"},
				{Filename: fmt.Sprintf("go%s.linux-arm64.tar.gz", version), OS: "linux", Arch: "arm64", Kind: "archive", SHA256: "placeholder"},
				{Filename: fmt.Sprintf("go%s.freebsd-amd64.tar.gz", version), OS: "freebsd", Arch: "amd64", Kind: "archive", SHA256: "placeholder"},
			},
		}

		releases = append(releases, release)
	}

	// Sort releases by version (newest first)
	sort.Slice(releases, func(i, j int) bool {
		return compareVersions(releases[i].Version, releases[j].Version) > 0
	})

	// Generate Go code
	fmt.Println("// Generated embedded version data")
	fmt.Println("var EmbeddedVersions = []GoRelease{")

	// Only include the most recent versions to keep binary size reasonable
	count := 0
	for _, release := range releases {
		if count >= 20 { // Limit to most recent 20 versions
			break
		}

		data, _ := json.MarshalIndent(release, "\t", "\t")
		fmt.Printf("\t%s,\n", string(data))
		count++
	}

	fmt.Println("}")
}

func compareVersions(v1, v2 string) int {
	// Simple version comparison (remove go prefix)
	v1 = strings.TrimPrefix(v1, "go")
	v2 = strings.TrimPrefix(v2, "go")

	parts1 := strings.Split(v1, ".")
	parts2 := strings.Split(v2, ".")

	maxLen := len(parts1)
	if len(parts2) > maxLen {
		maxLen = len(parts2)
	}

	for i := 0; i < maxLen; i++ {
		var n1, n2 int
		if i < len(parts1) {
			n1, _ = strconv.Atoi(parts1[i])
		}
		if i < len(parts2) {
			n2, _ = strconv.Atoi(parts2[i])
		}

		if n1 > n2 {
			return 1
		} else if n1 < n2 {
			return -1
		}
	}

	return 0
}
