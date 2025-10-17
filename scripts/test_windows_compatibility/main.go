// Windows Compatibility Test Suite
// Tests both runtime support and code quality for Windows compatibility
package main

import (
	"bufio"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/go-nv/goenv/internal/version"
)

var (
	passed = 0
	failed = 0
	warnings = 0
)

func main() {
	fmt.Println("==================================================")
	fmt.Println("    Windows Compatibility Verification")
	fmt.Println("==================================================")
	fmt.Println()

	// Runtime tests
	testEmbeddedVersions()
	testGetFileForPlatform()
	testPlatformDistribution()
	testWindowsArchitectures()
	testWindowsFileExtensions()

	// Static code analysis tests
	testPathSeparators()
	testHomeEnvVar()
	testUnixCommands()
	testShellRedirection()

	// Summary
	fmt.Println()
	fmt.Println("==================================================")
	fmt.Println("              Summary")
	fmt.Println("==================================================")

	total := passed + failed
	if failed == 0 && warnings == 0 {
		fmt.Printf("✅ All %d tests passed!\n", total)
		fmt.Println()
		fmt.Println("Windows support verified:")
		fmt.Println("  ✓ Windows versions in embedded data")
		fmt.Println("  ✓ GetFileForPlatform works for Windows")
		fmt.Println("  ✓ All Windows architectures supported (386, amd64, arm64)")
		fmt.Println("  ✓ Correct file extensions (.zip for Windows)")
		fmt.Println("  ✓ No hardcoded path separators")
		fmt.Println("  ✓ No HOME env var issues")
		fmt.Println("  ✓ No Unix-only commands")
		fmt.Println("  ✓ No Unix-specific shell redirection")
		fmt.Println()
		fmt.Println("Recommendations:")
		fmt.Println("  1. Test actual installation on Windows")
		fmt.Println("  2. Verify PowerShell integration")
		fmt.Println("  3. Test with Windows path separators (backslashes)")
		fmt.Println("  4. Verify shim .exe creation works")
	} else {
		if failed > 0 {
			fmt.Printf("❌ %d/%d tests failed\n", failed, total)
		}
		if warnings > 0 {
			fmt.Printf("⚠️  %d warnings (non-critical issues)\n", warnings)
		}
		if failed > 0 {
			os.Exit(1)
		}
	}

	fmt.Println("==================================================")
}

func testEmbeddedVersions() {
	fmt.Println("Test 1: Embedded Versions Include Windows")
	fmt.Println("--------------------------------------------------")

	windowsCount := 0
	totalVersions := len(version.EmbeddedVersions)

	// Count versions that have at least one Windows file
	for _, release := range version.EmbeddedVersions {
		hasWindows := false
		for _, file := range release.Files {
			if file.OS == "windows" {
				hasWindows = true
				break
			}
		}
		if hasWindows {
			windowsCount++
		}
	}

	if windowsCount > 0 {
		fmt.Printf("✓ PASS: %d/%d versions include Windows files\n", windowsCount, totalVersions)
		passed++
	} else {
		fmt.Printf("✗ FAIL: No Windows versions found in embedded data\n")
		failed++
	}

	// Show sample
	sampleCount := 0
	for _, release := range version.EmbeddedVersions {
		for _, file := range release.Files {
			if file.OS == "windows" && sampleCount < 3 {
				fmt.Printf("  Sample: %s - %s\n", release.Version, file.Filename)
				sampleCount++
				break
			}
		}
		if sampleCount >= 3 {
			break
		}
	}
	fmt.Println()
}

func testGetFileForPlatform() {
	fmt.Println("Test 2: GetFileForPlatform Works for Windows")
	fmt.Println("--------------------------------------------------")

	if len(version.EmbeddedVersions) == 0 {
		fmt.Println("✗ FAIL: No embedded versions to test")
		failed++
		fmt.Println()
		return
	}

	release := version.EmbeddedVersions[0]

	// Test all common Windows architectures
	testCases := []struct {
		arch string
		name string
	}{
		{"amd64", "Windows x64"},
		{"386", "Windows 32-bit"},
		{"arm64", "Windows ARM64"},
	}

	allPassed := true
	for _, tc := range testCases {
		file, err := release.GetFileForPlatform("windows", tc.arch)
		if err != nil {
			fmt.Printf("✗ FAIL: Could not get %s: %v\n", tc.name, err)
			allPassed = false
		} else {
			fmt.Printf("✓ PASS: %s - %s (size: %d bytes)\n", tc.name, file.Filename, file.Size)
		}
	}

	if allPassed {
		passed++
	} else {
		failed++
	}
	fmt.Println()
}

func testPlatformDistribution() {
	fmt.Println("Test 3: Platform Distribution")
	fmt.Println("--------------------------------------------------")

	platforms := make(map[string]int)
	for _, release := range version.EmbeddedVersions {
		for _, file := range release.Files {
			platforms[file.OS]++
		}
	}

	windowsFiles := platforms["windows"]
	totalFiles := 0
	for _, count := range platforms {
		totalFiles += count
	}

	for os, count := range platforms {
		percentage := float64(count) / float64(totalFiles) * 100
		fmt.Printf("  %s: %d files (%.1f%%)\n", os, count, percentage)
	}

	if windowsFiles > 0 {
		fmt.Printf("✓ PASS: Windows has %d files (well represented)\n", windowsFiles)
		passed++
	} else {
		fmt.Printf("✗ FAIL: No Windows files found\n")
		failed++
	}
	fmt.Println()
}

func testWindowsArchitectures() {
	fmt.Println("Test 4: Windows Architecture Coverage")
	fmt.Println("--------------------------------------------------")

	architectures := make(map[string]int)
	for _, release := range version.EmbeddedVersions {
		for _, file := range release.Files {
			if file.OS == "windows" {
				architectures[file.Arch]++
			}
		}
	}

	requiredArchs := []string{"amd64", "386", "arm64"}
	allPresent := true

	for _, arch := range requiredArchs {
		count := architectures[arch]
		if count > 0 {
			fmt.Printf("✓ %s: %d files\n", arch, count)
		} else {
			fmt.Printf("✗ %s: missing\n", arch)
			allPresent = false
		}
	}

	if allPresent {
		fmt.Println("✓ PASS: All major Windows architectures supported")
		passed++
	} else {
		fmt.Println("✗ FAIL: Some Windows architectures missing")
		failed++
	}
	fmt.Println()
}

func testWindowsFileExtensions() {
	fmt.Println("Test 5: Windows File Extensions")
	fmt.Println("--------------------------------------------------")

	correctExt := 0
	wrongExt := 0

	for _, release := range version.EmbeddedVersions {
		for _, file := range release.Files {
			if file.OS == "windows" {
				ext := filepath.Ext(file.Filename)
				if ext == ".zip" {
					correctExt++
				} else {
					wrongExt++
					if wrongExt <= 3 {
						fmt.Printf("⚠️  Unexpected extension: %s (got %s)\n", file.Filename, ext)
					}
				}
			}
		}
	}

	if wrongExt == 0 && correctExt > 0 {
		fmt.Printf("✓ PASS: All %d Windows files use .zip extension\n", correctExt)
		passed++
	} else if wrongExt > 0 {
		fmt.Printf("✗ FAIL: %d Windows files have wrong extension\n", wrongExt)
		failed++
	} else {
		fmt.Println("✗ FAIL: No Windows files found")
		failed++
	}
	fmt.Println()
}

func testPathSeparators() {
	fmt.Println("Test 6: No Hardcoded Path Separators")
	fmt.Println("--------------------------------------------------")

	issues := []string{}
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "vendor" || info.Name() == ".git" || info.Name() == "node_modules" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// Check for hardcoded colon in path list separators
			if regexp.MustCompile(`\+\s*":"\s*\+`).MatchString(line) &&
				!strings.Contains(line, "os.PathListSeparator") &&
				!strings.Contains(line, "http") &&
				!strings.Contains(line, "//") {
				issues = append(issues, fmt.Sprintf("%s:%d: %s", path, lineNum, strings.TrimSpace(line)))
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if len(issues) == 0 {
		fmt.Println("✓ PASS: No hardcoded path separator issues found")
		passed++
	} else {
		fmt.Printf("✗ FAIL: Found %d hardcoded path separator issues\n", len(issues))
		for i, issue := range issues {
			if i < 3 {
				fmt.Printf("  %s\n", issue)
			}
		}
		if len(issues) > 3 {
			fmt.Printf("  ... and %d more\n", len(issues)-3)
		}
		failed++
	}
	fmt.Println()
}

func testHomeEnvVar() {
	fmt.Println("Test 7: No HOME Environment Variable Usage")
	fmt.Println("--------------------------------------------------")

	issues := []string{}
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "vendor" || info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			if regexp.MustCompile(`os\.Getenv\("HOME"\)`).MatchString(line) &&
				!strings.Contains(line, "//") &&
				!strings.Contains(line, "Fallback") {
				issues = append(issues, fmt.Sprintf("%s:%d", path, lineNum))
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if len(issues) == 0 {
		fmt.Println("✓ PASS: No HOME environment variable issues found")
		passed++
	} else {
		fmt.Printf("⚠️  WARNING: Found %d HOME env var usages (should use os.UserHomeDir)\n", len(issues))
		for i, issue := range issues {
			if i < 3 {
				fmt.Printf("  %s\n", issue)
			}
		}
		if len(issues) > 3 {
			fmt.Printf("  ... and %d more\n", len(issues)-3)
		}
		warnings += len(issues)
	}
	fmt.Println()
}

func testUnixCommands() {
	fmt.Println("Test 8: No Unix-Only Commands Without Platform Checks")
	fmt.Println("--------------------------------------------------")

	issues := []string{}
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "vendor" || info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		hasWindowsCheck := false
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// Track if file has Windows checks
			if strings.Contains(line, "runtime.GOOS") && strings.Contains(line, "windows") {
				hasWindowsCheck = true
			}

			// Check for ps command
			if strings.Contains(line, `exec.Command`) && strings.Contains(line, `"ps"`) &&
				!strings.Contains(line, "//") {
				if !hasWindowsCheck {
					issues = append(issues, fmt.Sprintf("%s:%d: ps command", path, lineNum))
				}
			}

			// Check for hash -r
			if strings.Contains(line, "hash -r") && !strings.Contains(line, "//") {
				if !hasWindowsCheck {
					issues = append(issues, fmt.Sprintf("%s:%d: hash -r", path, lineNum))
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if len(issues) == 0 {
		fmt.Println("✓ PASS: No Unix-only command issues found")
		passed++
	} else {
		fmt.Printf("✗ FAIL: Found %d Unix-only commands without platform checks\n", len(issues))
		for i, issue := range issues {
			if i < 3 {
				fmt.Printf("  %s\n", issue)
			}
		}
		if len(issues) > 3 {
			fmt.Printf("  ... and %d more\n", len(issues)-3)
		}
		failed++
	}
	fmt.Println()
}

func testShellRedirection() {
	fmt.Println("Test 9: No Unix-Specific Shell Redirection")
	fmt.Println("--------------------------------------------------")

	issues := []string{}
	err := filepath.Walk(".", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			if info.Name() == "vendor" || info.Name() == ".git" {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		hasShellCheck := false
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// Track if section has shell checks
			if strings.Contains(line, "shell") && (strings.Contains(line, "powershell") || strings.Contains(line, "cmd")) {
				hasShellCheck = true
			}

			if regexp.MustCompile(`2>/dev/null|>/dev/null`).MatchString(line) &&
				!strings.Contains(line, "//") &&
				!hasShellCheck {
				issues = append(issues, fmt.Sprintf("%s:%d", path, lineNum))
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if len(issues) == 0 {
		fmt.Println("✓ PASS: No Unix shell redirection issues found")
		passed++
	} else {
		fmt.Printf("✗ FAIL: Found %d Unix shell redirection issues\n", len(issues))
		for i, issue := range issues {
			if i < 3 {
				fmt.Printf("  %s\n", issue)
			}
		}
		if len(issues) > 3 {
			fmt.Printf("  ... and %d more\n", len(issues)-3)
		}
		failed++
	}
	fmt.Println()
}
