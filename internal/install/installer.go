package install

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"crypto/sha256"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/go-nv/goenv/internal/config"
	"github.com/go-nv/goenv/internal/version"
	"github.com/schollz/progressbar/v3"
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

// NewInstaller creates a new installer
func NewInstaller(cfg *config.Config) *Installer {
	return &Installer{
		config: cfg,
		client: &http.Client{
			Timeout: 10 * time.Minute, // Long timeout for downloads
		},
		MirrorURL: os.Getenv("GO_BUILD_MIRROR_URL"),
	}
}

// Install installs a specific Go version
func (i *Installer) Install(goVersion string, force bool) error {
	// Fetch version information with fallback
	fetcher := version.NewFetcherWithOptions(version.FetcherOptions{Debug: false})
	releases, err := fetcher.FetchWithFallback(i.config.Root)
	if err != nil {
		return fmt.Errorf("failed to fetch available versions: %w", err)
	}

	var targetRelease *version.GoRelease
	for _, release := range releases {
		if release.Version == goVersion || strings.TrimPrefix(release.Version, "go") == goVersion {
			targetRelease = &release
			break
		}
	}

	if targetRelease == nil {
		return fmt.Errorf("version %s not found", goVersion)
	}

	// Get download file for current platform
	file, err := targetRelease.GetFileForPlatform("", "")
	if err != nil {
		return fmt.Errorf("no download available for current platform: %w", err)
	}

	versionDir := filepath.Join(i.config.VersionsDir(), strings.TrimPrefix(targetRelease.Version, "go"))

	// Check if already installed
	if !force {
		if _, err := os.Stat(versionDir); err == nil {
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
				return fmt.Errorf("failed to download: %w", err)
			}
		} else {
			return fmt.Errorf("failed to download: %w", err)
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
			return fmt.Errorf("failed to extract: %w", err)
		}
	} else {
		if err := i.extractTarGz(tempFile, versionDir); err != nil {
			return fmt.Errorf("failed to extract: %w", err)
		}
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
		return "", fmt.Errorf("failed to create temp file: %w", err)
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
		return "", fmt.Errorf("failed to write download: %w", err)
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
		return fmt.Errorf("failed to open archive: %w", err)
	}
	defer file.Close()

	gzr, err := gzip.NewReader(file)
	if err != nil {
		return fmt.Errorf("failed to create gzip reader: %w", err)
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
		if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
			return fmt.Errorf("failed to create directory: %w", err)
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
		return fmt.Errorf("failed to open ZIP archive: %w", err)
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
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("failed to create directory %s: %w", target, err)
			}
		} else {
			// Ensure parent directory exists
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("failed to create parent directory: %w", err)
			}

			// Extract file
			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, f.Mode())
			if err != nil {
				return fmt.Errorf("failed to create file %s: %w", target, err)
			}

			rc, err := f.Open()
			if err != nil {
				outFile.Close()
				return fmt.Errorf("failed to open file in ZIP: %w", err)
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

	if _, err := os.Stat(versionDir); os.IsNotExist(err) {
		return fmt.Errorf("version %s is not installed", goVersion)
	}

	fmt.Printf("Uninstalling Go %s...\n", goVersion)

	if err := os.RemoveAll(versionDir); err != nil {
		return fmt.Errorf("failed to remove version directory: %w", err)
	}

	fmt.Printf("Successfully uninstalled Go %s\n", goVersion)
	return nil
}
