package utils

import "fmt"

// GoReleaseNotesURL returns the URL to Go release notes for a given version.
// This consolidates the repeated pattern of constructing release notes URLs (8+ occurrences).
//
// Example:
//
//	url := GoReleaseNotesURL("1.21.0")
//	// Returns: "https://go.dev/doc/go1.21"
func GoReleaseNotesURL(version string) string {
	majorMinor := ExtractMajorMinor(NormalizeGoVersion(version))
	if majorMinor == "" {
		return ""
	}
	return fmt.Sprintf("https://go.dev/doc/go%s", majorMinor)
}

// GoDownloadURL returns the official download URL for a Go release file.
// This is the base URL from go.dev for downloading Go distributions.
//
// Example:
//
//	url := GoDownloadURL("go1.21.0.darwin-arm64.tar.gz")
//	// Returns: "https://go.dev/dl/go1.21.0.darwin-arm64.tar.gz"
func GoDownloadURL(filename string) string {
	return fmt.Sprintf("https://go.dev/dl/%s", filename)
}

// GoenvReleaseURL returns the URL to a goenv release on GitHub.
//
// Example:
//
//	url := GoenvReleaseURL("v2.1.0", "goenv-darwin-arm64")
//	// Returns: "https://github.com/go-nv/goenv/releases/download/v2.1.0/goenv-darwin-arm64"
func GoenvReleaseURL(version, assetName string) string {
	return fmt.Sprintf("https://github.com/go-nv/goenv/releases/download/%s/%s", version, assetName)
}

// GoenvChecksumURL returns the URL to the SHA256SUMS file for a goenv release.
//
// Example:
//
//	url := GoenvChecksumURL("v2.1.0")
//	// Returns: "https://github.com/go-nv/goenv/releases/download/v2.1.0/SHA256SUMS"
func GoenvChecksumURL(version string) string {
	return fmt.Sprintf("https://github.com/go-nv/goenv/releases/download/%s/SHA256SUMS", version)
}
