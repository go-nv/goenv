package install

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	stderrors "errors"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/errors"
	"github.com/go-nv/goenv/internal/pathutil"
	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/internal/version"
	"github.com/schollz/progressbar/v3"
)

// InstallPhase represents the current phase of installation
type InstallPhase string

const (
	PhaseFetching    InstallPhase = "fetching"
	PhaseValidating  InstallPhase = "validating"
	PhaseDownloading InstallPhase = "downloading"
	PhaseExtracting  InstallPhase = "extracting"
	PhaseVerifying   InstallPhase = "verifying"
	PhaseComplete    InstallPhase = "complete"
)

// InstallErrorType categorizes installation errors
type InstallErrorType string

const (
	ErrorTypeFetch            InstallErrorType = "fetch"
	ErrorTypeNotFound         InstallErrorType = "not_found"
	ErrorTypeAlreadyInstalled InstallErrorType = "already_installed"
	ErrorTypeDownload         InstallErrorType = "download"
	ErrorTypeExtract          InstallErrorType = "extract"
	ErrorTypeVerification     InstallErrorType = "verification"
	ErrorTypeChecksum         InstallErrorType = "checksum"
	ErrorTypeNotInstalled     InstallErrorType = "not_installed"
	ErrorTypePlatform         InstallErrorType = "platform"
)

// Installer handles Go installation
type Installer struct {
	config        *config.Config
	client        *http.Client
	Verbose       bool
	Quiet         bool
	KeepBuildPath bool
	MirrorURL     string
}

// InstallError represents a structured installation error
type InstallError struct {
	Type    InstallErrorType
	Phase   InstallPhase
	Message string
	Err     error
}

// Error implements the error interface
func (e *InstallError) Error() string {
	if e.Err != nil {
		return fmt.Sprintf("%s: %s: %v", e.Phase, e.Message, e.Err)
	}
	return fmt.Sprintf("%s: %s", e.Phase, e.Message)
}

// Unwrap implements error unwrapping
func (e *InstallError) Unwrap() error {
	return e.Err
}

// NewInstaller creates a new installer
func NewInstaller(cfg *config.Config) *Installer {
	return &Installer{
		config:    cfg,
		client:    utils.NewHTTPClientForDownloads(),
		MirrorURL: os.Getenv(utils.EnvVarGoBuildMirrorURL),
	}
}

// Install installs a specific Go version
func (i *Installer) Install(goVersion string, force bool) error {
	// Fetch version information with fallback
	fetcher := version.NewFetcherWithOptions(version.FetcherOptions{Debug: false})
	releases, err := fetcher.FetchWithFallback(i.config.Root)
	if err != nil {
		return errors.FailedTo("fetch available versions", err)
	}

	var targetRelease *version.GoRelease
	for _, release := range releases {
		if utils.MatchesVersion(release.Version, goVersion) {
			targetRelease = &release
			break
		}
	}

	if targetRelease == nil {
		return i.createVersionNotFoundError(goVersion, releases)
	}

	// Get download file for current platform
	file, err := targetRelease.GetFileForPlatform("", "")
	if err != nil {
		return fmt.Errorf("no download available for current platform: %w", err)
	}

	versionDir := i.config.VersionDir(utils.NormalizeGoVersion(targetRelease.Version))

	// Check if already installed
	if !force {
		if utils.PathExists(versionDir) {
			return fmt.Errorf("version %s is already installed (use --force to reinstall)", goVersion)
		}
	}

	if !i.Quiet {
		fmt.Printf("Installing Go %s...\n", targetRelease.Version)
	}

	// Construct download URL (use mirror if available)
	var downloadURL string
	if i.MirrorURL != "" {
		mirrorBase := strings.TrimSuffix(i.MirrorURL, "/")
		downloadURL = fmt.Sprintf("%s/%s", mirrorBase, file.Filename)
		if i.Verbose {
			fmt.Printf("Using mirror: %s\n", downloadURL)
		}
	} else {
		downloadURL = fmt.Sprintf("https://go.dev/dl/%s", file.Filename)
		if i.Verbose {
			fmt.Printf("Downloading from: %s\n", downloadURL)
		}
	}

	// Download the file
	tempFile, err := i.downloadFile(downloadURL, file.SHA256, file.Filename)
	if err != nil {
		// If mirror fails, try official source
		if i.MirrorURL != "" {
			if i.Verbose {
				fmt.Printf("Mirror download failed, trying official source...\n")
			}
			downloadURL = fmt.Sprintf("https://go.dev/dl/%s", file.Filename)
			tempFile, err = i.downloadFile(downloadURL, file.SHA256, file.Filename)
			if err != nil {
				return errors.FailedTo("download", err)
			}
		} else {
			return errors.FailedTo("download", err)
		}
	}

	// Keep or remove temp file based on flag
	if i.KeepBuildPath {
		if i.Verbose {
			fmt.Printf("Keeping downloaded file: %s\n", tempFile)
		}
	} else {
		defer os.Remove(tempFile)
	}

	// Extract the archive (ZIP for Windows, tar.gz for others)
	if strings.HasSuffix(file.Filename, ".zip") {
		if err := i.extractZip(tempFile, versionDir); err != nil {
			os.RemoveAll(versionDir) // Clean up on failure
			return errors.FailedTo("extract", err)
		}
	} else {
		if err := i.extractTarGz(tempFile, versionDir); err != nil {
			os.RemoveAll(versionDir) // Clean up on failure
			return errors.FailedTo("extract", err)
		}
	}

	// Verify installation by checking for go binary
	goBinaryBase := filepath.Join(versionDir, "bin", "go")

	// Check if go binary exists (handles .exe on Windows)
	goBinary, err := pathutil.FindExecutable(goBinaryBase)
	if err != nil {
		os.RemoveAll(versionDir) // Clean up corrupted installation
		return fmt.Errorf("installation verification failed: go binary not found (extraction may have failed)")
	}

	if i.Verbose {
		fmt.Printf("Installation verified: %s\n", goBinary)
	}

	fmt.Printf("Successfully installed Go %s to %s\n", targetRelease.Version, versionDir)
	return nil
}

// downloadFile downloads a file and verifies its checksum
func (i *Installer) downloadFile(url, expectedSHA256, filename string) (string, error) {
	resp, err := i.client.Get(url)
	if err != nil {
		return "", fmt.Errorf("failed to download from %s: %w", url, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("unexpected status code %d for %s", resp.StatusCode, url)
	}

	// Create temporary file
	tempFile, err := os.CreateTemp("", "goenv-download-*.tar.gz")
	if err != nil {
		return "", errors.FailedTo("create temp file", err)
	}
	defer tempFile.Close()

	// Create hash writer to verify checksum while downloading
	hasher := sha256.New()

	// Setup progress bar if not quiet mode
	var writer io.Writer
	var bar *progressbar.ProgressBar

	if !i.Quiet && resp.ContentLength > 0 {
		bar = progressbar.NewOptions64(
			resp.ContentLength,
			progressbar.OptionSetDescription(fmt.Sprintf("Downloading %s", filename)),
			progressbar.OptionSetWriter(os.Stderr),
			progressbar.OptionShowBytes(true),
			progressbar.OptionSetWidth(15),
			progressbar.OptionThrottle(65*time.Millisecond),
			progressbar.OptionShowCount(),
			progressbar.OptionOnCompletion(func() {
				fmt.Fprint(os.Stderr, "\n")
			}),
			progressbar.OptionSpinnerType(14),
			progressbar.OptionFullWidth(),
			progressbar.OptionSetRenderBlankState(true),
		)
		writer = io.MultiWriter(tempFile, hasher, bar)
	} else {
		writer = io.MultiWriter(tempFile, hasher)
		if !i.Quiet {
			fmt.Printf("Downloading %s...\n", filename)
		}
	}

	_, err = io.Copy(writer, resp.Body)
	if err != nil {
		os.Remove(tempFile.Name())
		return "", errors.FailedTo("write download", err)
	}

	// Verify checksum
	actualSHA256 := fmt.Sprintf("%x", hasher.Sum(nil))
	if actualSHA256 != expectedSHA256 {
		os.Remove(tempFile.Name())
		return "", fmt.Errorf("checksum verification failed: expected %s, got %s", expectedSHA256, actualSHA256)
	}

	if !i.Quiet && i.Verbose {
		fmt.Println("Download completed and verified")
	}

	return tempFile.Name(), nil
}

// extractTarGz extracts a tar.gz file to the specified directory
func (i *Installer) extractTarGz(tarGzPath, destDir string) error {
	file, err := os.Open(tarGzPath)
	if err != nil {
		return errors.FailedTo("open archive", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return errors.FailedTo("create gzip reader", err)
	}
	defer gzr.Close()

	tr := tar.NewReader(gzr)

	if !i.Quiet {
		fmt.Println("Extracting archive...")
	}

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("error reading tar: %w", err)
		}

		// Remove "go/" prefix from paths since we want the contents directly in destDir
		path := strings.TrimPrefix(header.Name, "go/")
		if path == "" {
			continue // Skip the root "go/" directory
		}

		target := filepath.Join(destDir, path)

		// Ensure the target directory exists
		if err := utils.EnsureDirWithContext(filepath.Dir(target), "create directory"); err != nil {
			return err
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, os.FileMode(header.Mode)); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}
		case tar.TypeReg:
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}

			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("failed to extract file %s: %w", target, err)
			}
			f.Close()
		}
	}

	if !i.Quiet {
		fmt.Println("Extraction completed")
	}
	return nil
}

// extractZip extracts a ZIP file to the specified directory (Windows support)
func (i *Installer) extractZip(zipPath, destDir string) error {
	r, err := zip.OpenReader(zipPath)
	if err != nil {
		return errors.FailedTo("open ZIP archive", err)
	}
	defer r.Close()

	if !i.Quiet {
		fmt.Println("Extracting archive...")
	}

	for _, f := range r.File {
		// Remove "go/" or "go\" prefix from paths
		path := strings.TrimPrefix(f.Name, "go/")
		path = strings.TrimPrefix(path, "go\\")
		if path == "" || path == "go" {
			continue // Skip the root "go/" directory
		}

		target := filepath.Join(destDir, path)

		// Security check: prevent zip slip
		if !strings.HasPrefix(filepath.Clean(target), filepath.Clean(destDir)+string(os.PathSeparator)) {
			return fmt.Errorf("illegal file path in ZIP: %s", f.Name)
		}

		if f.FileInfo().IsDir() {
			// Create directory
			if err := utils.EnsureDir(target); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}
		} else {
			// Ensure parent directory exists
			if err := utils.EnsureDirWithContext(filepath.Dir(target), "create parent directory"); err != nil {
				return err
			}

			// Extract file
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}

			rc, err := f.Open()
			if err != nil {
				outFile.Close()
				return errors.FailedTo("open file in ZIP", err)
			}

			_, err = io.Copy(outFile, rc)
			rc.Close()
			outFile.Close()

			if err != nil {
				return fmt.Errorf("failed to extract file %s: %w", target, err)
			}
		}
	}

	if !i.Quiet {
		fmt.Println("Extraction completed")
	}
	return nil
}

// Uninstall removes a Go version
func (i *Installer) Uninstall(goVersion string) error {
	versionDir := filepath.Join(i.config.VersionsDir(), goVersion)

	if utils.FileNotExists(versionDir) {
		return fmt.Errorf("version %s is not installed", goVersion)
	}

	fmt.Printf("Uninstalling Go %s...\n", goVersion)

	if err := os.RemoveAll(versionDir); err != nil {
		return errors.FailedTo("remove version directory", err)
	}

	fmt.Printf("Successfully uninstalled Go %s\n", goVersion)
	return nil
}

// createVersionNotFoundError creates a helpful error message with version suggestions
func (i *Installer) createVersionNotFoundError(requestedVersion string, availableReleases []version.GoRelease) error {
	// Basic error message
	errMsg := fmt.Sprintf("version %s not found", requestedVersion)

	// Find similar versions using fuzzy matching
	suggestions := i.findSimilarVersions(requestedVersion, availableReleases, 5)

	if len(suggestions) > 0 {
		errMsg += "\n\nDid you mean one of these?"
		for _, sug := range suggestions {
			marker := ""
			if sug.IsLatestInMinor {
				marker = " (latest)"
			}
			errMsg += fmt.Sprintf("\n  • %s%s", sug.Version, marker)
		}
		errMsg += "\n\nUse 'goenv install-list' to see all available versions"
	} else {
		errMsg += "\n\nUse 'goenv install-list' to see all available versions"
	}

	return stderrors.New(errMsg)
}

// versionSuggestion represents a suggested version with metadata
type versionSuggestion struct {
	Version         string
	IsLatestInMinor bool
}

// findSimilarVersions finds versions that are similar to the requested version
func (i *Installer) findSimilarVersions(requested string, releases []version.GoRelease, maxResults int) []versionSuggestion {
	var suggestions []versionSuggestion

	// Normalize requested version (remove "go" prefix if present)
	normalized := utils.NormalizeGoVersion(requested)

	// Strategy 1: Prefix matching (e.g., "1.21" matches "1.21.0", "1.21.1", etc.)
	if strings.Count(normalized, ".") < 2 {
		// Looking for major.minor - find all patches
		latestPerMinor := make(map[string]string) // track latest per minor version

		for _, release := range releases {
			ver := utils.NormalizeGoVersion(release.Version)

			// Check if this matches the prefix
			if strings.HasPrefix(ver, normalized+".") || ver == normalized {
				suggestions = append(suggestions, versionSuggestion{
					Version:         ver,
					IsLatestInMinor: false,
				})

				// Track latest for this minor version
				minorKey := utils.ExtractMajorMinor(ver)
				if latestPerMinor[minorKey] == "" || utils.CompareGoVersions(ver, latestPerMinor[minorKey]) > 0 {
					latestPerMinor[minorKey] = ver
				}

				if len(suggestions) >= maxResults {
					break
				}
			}
		}

		// Mark latest versions
		for idx := range suggestions {
			minorKey := utils.ExtractMajorMinor(suggestions[idx].Version)
			if suggestions[idx].Version == latestPerMinor[minorKey] {
				suggestions[idx].IsLatestInMinor = true
			}
		}

		if len(suggestions) > 0 {
			return suggestions
		}
	}

	// Strategy 2: Close version number matching (e.g., "1.20" suggests "1.21", "1.19")
	parts := strings.Split(normalized, ".")
	if len(parts) >= 2 {
		major := parts[0]
		minor := parts[1]

		// Try adjacent minor versions
		for _, release := range releases {
			ver := utils.NormalizeGoVersion(release.Version)
			verParts := strings.Split(ver, ".")

			if len(verParts) >= 2 && verParts[0] == major {
				// Check if minor version is close (±2)
				if absInt(parseInt(verParts[1])-parseInt(minor)) <= 2 {
					// Only add if it's the latest patch for that minor version
					if i.isLatestPatchVersion(ver, releases) {
						suggestions = append(suggestions, versionSuggestion{
							Version:         ver,
							IsLatestInMinor: true,
						})

						if len(suggestions) >= maxResults {
							break
						}
					}
				}
			}
		}

		if len(suggestions) > 0 {
			return suggestions
		}
	}

	// Strategy 3: Show latest stable versions as fallback
	for _, release := range releases {
		if release.Stable && !strings.Contains(release.Version, "beta") && !strings.Contains(release.Version, "rc") {
			ver := utils.NormalizeGoVersion(release.Version)
			suggestions = append(suggestions, versionSuggestion{
				Version:         ver,
				IsLatestInMinor: true,
			})

			if len(suggestions) >= maxResults {
				break
			}
		}
	}

	return suggestions
}

// isLatestPatchVersion checks if a version is the latest patch for its minor version
func (i *Installer) isLatestPatchVersion(ver string, releases []version.GoRelease) bool {
	minorKey := utils.ExtractMajorMinor(ver)

	for _, release := range releases {
		releaseVer := utils.NormalizeGoVersion(release.Version)
		if utils.ExtractMajorMinor(releaseVer) == minorKey {
			if utils.CompareGoVersions(releaseVer, ver) > 0 {
				return false // Found a newer patch version
			}
		}
	}

	return true
}

// parseInt parses an integer from string, returns 0 on error
func parseInt(s string) int {
	var result int
	fmt.Sscanf(s, "%d", &result)
	return result
}

// absInt returns absolute value of an integer
func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}
