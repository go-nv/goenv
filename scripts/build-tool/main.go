package main

import (
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"time"

	"github.com/go-nv/goenv/internal/utils"
)

var (
	version   = getVersion()
	commitSHA = getCommitSHA()
	buildTime = time.Now().UTC().Format("2006-01-02T15:04:05Z")
)

func main() {
	var (
		task   = flag.String("task", "build", "Task to run")
		prefix = flag.String("prefix", getDefaultPrefix(), "Installation prefix")
	)
	flag.Parse()

	switch *task {
	case "build":
		build()
	case "test":
		test()
	case "test-windows":
		testWindows()
	case "clean":
		clean()
	case "install":
		install(*prefix)
	case "uninstall":
		uninstall(*prefix)
	case "cross-build":
		crossBuild()
	case "generate-embedded":
		generateEmbedded()
	case "dev-deps":
		devDeps()
	case "migrate-test":
		migrateTest()
	case "bats-test":
		batsTest()
	case "release":
		release()
	case "snapshot":
		snapshot()
	case "version":
		showVersion()
	case "help":
		showHelp()
	default:
		fmt.Printf("Unknown task: %s\n", *task)
		showHelp()
		os.Exit(1)
	}
}

func build() {
	fmt.Println("Building goenv...")

	binaryName := "goenv"
	if runtime.GOOS == "windows" {
		binaryName += ".exe"
	}

	ldflags := fmt.Sprintf("-X main.version=%s -X main.commit=%s -X main.buildTime=%s", version, commitSHA, buildTime)

	cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", binaryName, ".")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Build failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Printf("✓ Built %s successfully\n", binaryName)
}

func test() {
	fmt.Println("Running tests...")

	cmd := exec.Command("go", "test", "-v", "./...")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Tests failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ All tests passed")
}

func clean() {
	fmt.Println("Cleaning build artifacts...")

	patterns := []string{"goenv", "goenv.exe", "bin/", "dist/"}

	for _, pattern := range patterns {
		if err := os.RemoveAll(pattern); err != nil && !os.IsNotExist(err) {
			fmt.Printf("Warning: failed to remove %s: %v\n", pattern, err)
		}
	}

	// Run go clean
	cmd := exec.Command("go", "clean")
	cmd.Run() // Ignore errors

	fmt.Println("✓ Clean complete")
}

func crossBuild() {
	fmt.Println("Cross-compiling for all platforms...")

	// Generate embedded versions first
	generateEmbedded()

	if err := os.MkdirAll("dist", 0755); err != nil {
		fmt.Printf("Failed to create dist directory: %v\n", err)
		os.Exit(1)
	}

	platforms := []struct {
		GOOS, GOARCH string
	}{
		{"linux", "amd64"},
		{"linux", "arm64"},
		{"darwin", "amd64"},
		{"darwin", "arm64"},
		{"windows", "amd64"},
		{"freebsd", "amd64"},
	}

	for _, platform := range platforms {
		buildForPlatform(platform.GOOS, platform.GOARCH)
	}

	fmt.Println("✓ Cross-compilation complete")
}

func buildForPlatform(goos, goarch string) {
	binaryName := fmt.Sprintf("dist/goenv-%s-%s", goos, goarch)
	if goos == "windows" {
		binaryName += ".exe"
	}

	fmt.Printf("Building for %s/%s...\n", goos, goarch)

	ldflags := fmt.Sprintf("-X main.version=%s -X main.commit=%s -X main.buildTime=%s", version, commitSHA, buildTime)

	cmd := exec.Command("go", "build", "-ldflags", ldflags, "-o", binaryName, ".")
	cmd.Env = append(os.Environ(),
		"GOOS="+goos,
		"GOARCH="+goarch,
	)

	if err := cmd.Run(); err != nil {
		fmt.Printf("✗ Failed to build for %s/%s: %v\n", goos, goarch, err)
	} else {
		fmt.Printf("✓ %s\n", binaryName)
	}
}

func install(prefix string) {
	fmt.Printf("Installing goenv to %s...\n", prefix)

	// Build first
	build()

	binDir := filepath.Join(prefix, "bin")
	if err := os.MkdirAll(binDir, 0755); err != nil {
		fmt.Printf("Failed to create bin directory: %v\n", err)
		os.Exit(1)
	}

	// Copy binary
	srcBinary := "goenv"
	if runtime.GOOS == "windows" {
		srcBinary += ".exe"
	}

	dstBinary := filepath.Join(binDir, filepath.Base(srcBinary))
	if err := utils.CopyFile(srcBinary, dstBinary); err != nil {
		fmt.Printf("Failed to copy binary: %v\n", err)
		os.Exit(1)
	}

	// Ensure executable on Unix (CopyFile preserves permissions, but set explicitly for safety)
	if runtime.GOOS != "windows" {
		if err := os.Chmod(dstBinary, 0755); err != nil {
			fmt.Printf("Failed to make binary executable: %v\n", err)
			os.Exit(1)
		}
	}

	fmt.Printf("✓ Installed %s\n", dstBinary)
}

func generateEmbedded() {
	fmt.Println("Generating embedded versions...")

	cmd := exec.Command("go", "run", "scripts/generate_embedded_versions/main.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to generate embedded versions: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Embedded versions generated")
}

func uninstall(prefix string) {
	binPath := filepath.Join(prefix, "bin", "goenv")
	if runtime.GOOS == "windows" {
		binPath += ".exe"
	}

	if err := os.Remove(binPath); err != nil && !os.IsNotExist(err) {
		fmt.Printf("Failed to remove binary: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Uninstalled goenv")
}

func showVersion() {
	fmt.Printf("goenv Build Information\n")
	fmt.Printf("  Version:    %s\n", version)
	fmt.Printf("  Commit:     %s\n", commitSHA)
	fmt.Printf("  Build Time: %s\n", buildTime)
}

func testWindows() {
	fmt.Println("Testing Windows compatibility...")

	cmd := exec.Command("go", "run", "scripts/test_windows_compatibility/main.go")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("Windows compatibility test failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Windows compatibility verified")
}

func devDeps() {
	fmt.Println("Managing Go module dependencies...")

	fmt.Println("Downloading dependencies...")
	cmd := exec.Command("go", "mod", "download")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to download dependencies: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("Tidying modules...")
	cmd = exec.Command("go", "mod", "tidy")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		fmt.Printf("Failed to tidy modules: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Dependencies updated")
}

func migrateTest() {
	fmt.Println("Running migration tests (Go tests + bats tests)...")

	// Run Go tests first
	test()

	// Then run bats tests
	batsTest()
}

func batsTest() {
	fmt.Println("Running legacy bats tests (if available)...")

	// Check if bats is available and test directory exists
	batsCmd := exec.Command("bats", "--version")
	if err := batsCmd.Run(); err != nil {
		fmt.Println("Bats not installed - skipping legacy tests")
		return
	}

	if _, err := os.Stat("test"); os.IsNotExist(err) {
		fmt.Println("Test directory not found - skipping legacy tests")
		return
	}

	cmd := exec.Command("bats", "test/")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Println("Bats tests failed or not available - continuing")
	} else {
		fmt.Println("✓ Bats tests passed")
	}
}

func release() {
	fmt.Println("Creating release with GoReleaser...")

	// Check if goreleaser is available
	checkCmd := exec.Command("goreleaser", "version")
	if err := checkCmd.Run(); err != nil {
		fmt.Printf("Error: goreleaser not found. Install with: go install github.com/goreleaser/goreleaser@latest\n")
		os.Exit(1)
	}

	cmd := exec.Command("goreleaser", "release", "--clean")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("GoReleaser release failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Release created successfully")
}

func snapshot() {
	fmt.Println("Creating snapshot build with GoReleaser...")

	// Check if goreleaser is available
	checkCmd := exec.Command("goreleaser", "version")
	if err := checkCmd.Run(); err != nil {
		fmt.Printf("Error: goreleaser not found. Install with: go install github.com/goreleaser/goreleaser@latest\n")
		os.Exit(1)
	}

	cmd := exec.Command("goreleaser", "build", "--snapshot", "--clean")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		fmt.Printf("GoReleaser snapshot failed: %v\n", err)
		os.Exit(1)
	}

	fmt.Println("✓ Snapshot build created successfully")
}

func showHelp() {
	fmt.Println(`goenv Build Tool

Usage: go run scripts/build-tool/main.go -task=<task> [options]

Tasks:
  build              Build the goenv binary (default)
  test               Run all tests
  test-windows       Test Windows compatibility
  clean              Clean build artifacts
  install            Install goenv (use -prefix to set location)
  uninstall          Uninstall goenv
  cross-build        Build for all platforms
  generate-embedded  Generate embedded versions
  dev-deps           Download and tidy Go module dependencies
  migrate-test       Run both Go tests and bats tests
  bats-test          Run legacy bats tests (if available)
  release            Create release with GoReleaser
  snapshot           Create snapshot build with GoReleaser
  version            Show version information
  help               Show this help

Options:
  -prefix string     Installation prefix (default: platform-specific)

Examples:
  go run scripts/build-tool/main.go -task=build
  go run scripts/build-tool/main.go -task=install -prefix=/usr/local
  go run scripts/build-tool/main.go -task=cross-build
  go run scripts/build-tool/main.go -task=release`)
}

// Helper functions
func getVersion() string {
	if content, err := os.ReadFile("APP_VERSION"); err == nil {
		return strings.TrimSpace(string(content))
	}
	return "dev"
}

func getCommitSHA() string {
	cmd := exec.Command("git", "rev-parse", "--short", "HEAD")
	if output, err := cmd.Output(); err == nil {
		return strings.TrimSpace(string(output))
	}
	return "unknown"
}

func getDefaultPrefix() string {
	if runtime.GOOS == "windows" {
		if appdata := os.Getenv("LOCALAPPDATA"); appdata != "" {
			return filepath.Join(appdata, "goenv")
		}
		return "C:\\goenv"
	}
	return "/usr/local"
}
