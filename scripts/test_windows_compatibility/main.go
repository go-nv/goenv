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
	passed   = 0
	failed   = 0
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
	testHardcodedUnixPaths()
	testFilePermissions()
	testPowerShellProfileDetection()

	// Windows binary detection tests
	testHardcodedExeChecks()
	testBinaryCreationInTests()
	testPathComparisonInTests()
	testPlatformSpecificErrorMessages()
	testBinaryExtensionStripping()
	testPermissionTestsSkipWindows()
	testHardcodedShebangInTests()
	testShimCreationPatterns()
	testExtensionStrippingForDisplay()
	testExeFilesWithScriptContent()
	testBatFilesProperSyntax()
	testBatchEchoQuotes()
	testContainsPathComparisons()

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
		fmt.Println("  ✓ No hardcoded Unix paths")
		fmt.Println("  ✓ File permissions handled correctly")
		fmt.Println("  ✓ PowerShell profile detection implemented")
		fmt.Println("  ✓ All .exe checks include .bat support")
		fmt.Println("  ✓ Test binaries use platform-specific extensions")
		fmt.Println("  ✓ Path comparisons use ToSlash() normalization")
		fmt.Println("  ✓ Error messages are platform-agnostic")
		fmt.Println("  ✓ Binary extension stripping handles both .exe and .bat")
		fmt.Println("  ✓ Permission tests skip on Windows")
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
		hasShellBranching := false
		for scanner.Scan() {
			lineNum++
			line := scanner.Text()

			// Track if file has Windows checks
			if strings.Contains(line, "runtime.GOOS") && strings.Contains(line, "windows") {
				hasWindowsCheck = true
			}

			// Track if file has shell-specific branching (fish/powershell/cmd/else)
			if strings.Contains(line, "shell ==") &&
				(strings.Contains(line, "powershell") || strings.Contains(line, "cmd") || strings.Contains(line, "fish")) {
				hasShellBranching = true
			}

			// Check for ps command
			if strings.Contains(line, `exec.Command`) && strings.Contains(line, `"ps"`) &&
				!strings.Contains(line, "//") {
				if !hasWindowsCheck {
					issues = append(issues, fmt.Sprintf("%s:%d: ps command", path, lineNum))
				}
			}

			// Check for hash -r (but allow if in shell-specific branch)
			if strings.Contains(line, "hash -r") && !strings.Contains(line, "//") {
				// Only flag if no platform/shell checks exist
				if !hasWindowsCheck && !hasShellBranching {
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

func testHardcodedUnixPaths() {
	fmt.Println("Test 10: No Hardcoded Unix Paths")
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
		// Skip this test file itself to avoid self-reference
		if strings.Contains(path, "test_windows_compatibility") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		lineNum := 0
		allLines := []string{} // Store all lines for function-level analysis
		for scanner.Scan() {
			allLines = append(allLines, scanner.Text())
		}

		// Analyze lines with full file context
		for i, line := range allLines {
			lineNum = i + 1

			// Check for hardcoded Unix paths like /usr/local
			if regexp.MustCompile(`"/usr/|"/etc/|"/var/|"/tmp/`).MatchString(line) &&
				!strings.Contains(line, "//") &&
				!strings.Contains(line, "runtime.GOOS") {

				// Look back up to 15 lines for function-level guards
				hasGuard := false
				lookback := 15
				if i < lookback {
					lookback = i
				}
				for j := 1; j <= lookback; j++ {
					prevLine := allLines[i-j]
					// Check for Windows guards or function boundaries
					if strings.Contains(prevLine, "runtime.GOOS") &&
						(strings.Contains(prevLine, "windows") || strings.Contains(prevLine, "!= \"windows\"")) {
						hasGuard = true
						break
					}
					// Stop at function boundary
					if strings.HasPrefix(strings.TrimSpace(prevLine), "func ") {
						break
					}
				}

				if !hasGuard {
					issues = append(issues, fmt.Sprintf("%s:%d: %s", path, lineNum, strings.TrimSpace(line)))
				}
			}

			// Check for hardcoded .bashrc, .zshrc paths without os.UserHomeDir
			if (strings.Contains(line, ".bashrc") || strings.Contains(line, ".zshrc") || strings.Contains(line, ".config")) &&
				strings.Contains(line, `"HOME"`) &&
				!strings.Contains(line, "//") {
				issues = append(issues, fmt.Sprintf("%s:%d: HOME env var in path", path, lineNum))
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if len(issues) == 0 {
		fmt.Println("✓ PASS: No hardcoded Unix paths found")
		passed++
	} else {
		fmt.Printf("⚠️  WARNING: Found %d hardcoded Unix path issues\n", len(issues))
		for i, issue := range issues {
			if i < 5 {
				fmt.Printf("  %s\n", issue)
			}
		}
		if len(issues) > 5 {
			fmt.Printf("  ... and %d more\n", len(issues)-5)
		}
		warnings += len(issues)
	}
	fmt.Println()
}

func testFilePermissions() {
	fmt.Println("Test 11: File Permission Handling")
	fmt.Println("--------------------------------------------------")

	issues := []string{}
	shimFiles := make(map[string]bool)

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

		// Track shim-related files
		if strings.Contains(path, "shim") {
			shimFiles[path] = true
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

			// Check for inconsistent permissions between .bat and Unix shims
			if strings.Contains(line, "WriteFile") && regexp.MustCompile(`0[67]\d{2}`).MatchString(line) {
				// Check if this file handles both Windows and Unix
				if shimFiles[path] && !strings.Contains(line, "runtime.GOOS") {
					issues = append(issues, fmt.Sprintf("%s:%d: Permission may need platform check", path, lineNum))
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if len(issues) == 0 {
		fmt.Println("✓ PASS: File permissions look correct")
		passed++
	} else {
		fmt.Printf("⚠️  INFO: Found %d file permission patterns (informational)\n", len(issues))
		for i, issue := range issues {
			if i < 3 {
				fmt.Printf("  %s\n", issue)
			}
		}
		warnings++
	}
	fmt.Println()
}

func testPowerShellProfileDetection() {
	fmt.Println("Test 12: PowerShell Profile Auto-Detection Support")
	fmt.Println("--------------------------------------------------")

	foundPowerShellHandling := false
	foundProfilePath := false

	err := filepath.Walk("../../cmd", func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}
		if !strings.HasSuffix(path, ".go") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			line := scanner.Text()
			if strings.Contains(line, "powershell") && strings.Contains(line, "case") {
				foundPowerShellHandling = true
			}
			if strings.Contains(line, "$PROFILE") || strings.Contains(line, "Documents\\PowerShell") || strings.Contains(line, "Documents/PowerShell") {
				foundProfilePath = true
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if foundPowerShellHandling && foundProfilePath {
		fmt.Println("✓ PASS: PowerShell profile detection implemented")
		passed++
	} else if foundPowerShellHandling {
		fmt.Println("⚠️  INFO: PowerShell handling exists but profile auto-detection may need improvement")
		fmt.Println("  Consider adding $PROFILE path detection for Windows users")
		warnings++
	} else {
		fmt.Println("ℹ️  INFO: PowerShell integration may need profile auto-detection")
		warnings++
	}
	fmt.Println()
}

func testHardcodedExeChecks() {
	fmt.Println("Test 13: No Hardcoded .exe Checks Without .bat Support")
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
		allLines := []string{}
		for scanner.Scan() {
			allLines = append(allLines, scanner.Text())
		}

		for i, line := range allLines {
			lineNum := i + 1

			// Check for hardcoded .exe without .bat support
			if (strings.Contains(line, `".exe"`) || strings.Contains(line, "go.exe")) &&
				strings.Contains(line, "runtime.GOOS") &&
				strings.Contains(line, "windows") &&
				!strings.Contains(line, "//") {

				// Look ahead/behind for .bat support or pathutil.FindExecutable
				hasBatSupport := false
				lookRange := 5
				startIdx := i - lookRange
				if startIdx < 0 {
					startIdx = 0
				}
				endIdx := i + lookRange
				if endIdx >= len(allLines) {
					endIdx = len(allLines) - 1
				}

				for j := startIdx; j <= endIdx; j++ {
					if strings.Contains(allLines[j], ".bat") || strings.Contains(allLines[j], "pathutil.FindExecutable") {
						hasBatSupport = true
						break
					}
				}

				if !hasBatSupport && !strings.Contains(path, "shim") {
					issues = append(issues, fmt.Sprintf("%s:%d: .exe without .bat support", path, lineNum))
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if len(issues) == 0 {
		fmt.Println("✓ PASS: All .exe checks include .bat support")
		passed++
	} else {
		fmt.Printf("⚠️  WARNING: Found %d .exe checks without .bat support\n", len(issues))
		for i, issue := range issues {
			if i < 3 {
				fmt.Printf("  %s\n", issue)
			}
		}
		warnings += len(issues)
	}
	fmt.Println()
}

func testBinaryCreationInTests() {
	fmt.Println("Test 14: Test Files Create Binaries With Correct Extensions")
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
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		allLines := []string{}
		for scanner.Scan() {
			allLines = append(allLines, scanner.Text())
		}

		for i, line := range allLines {
			lineNum := i + 1

			// Check for WriteFile creating binaries without Windows handling
			if strings.Contains(line, "WriteFile") &&
				(strings.Contains(line, `"go"`) || strings.Contains(line, "goExe") ||
					strings.Contains(line, "goBinary") || strings.Contains(line, "execPath")) &&
				!strings.Contains(line, "//") {

				// Check if runtime.GOOS or .bat exists nearby
				hasRuntimeCheck := false
				lookRange := 10
				startIdx := i - lookRange
				if startIdx < 0 {
					startIdx = 0
				}
				endIdx := i + lookRange
				if endIdx >= len(allLines) {
					endIdx = len(allLines) - 1
				}

				for j := startIdx; j <= endIdx; j++ {
					if (strings.Contains(allLines[j], "runtime.GOOS") && strings.Contains(allLines[j], "windows")) ||
						strings.Contains(allLines[j], ".bat") {
						hasRuntimeCheck = true
						break
					}
				}

				if !hasRuntimeCheck {
					issues = append(issues, fmt.Sprintf("%s:%d: Binary created without Windows extension handling", path, lineNum))
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if len(issues) == 0 {
		fmt.Println("✓ PASS: All test binaries use platform-specific extensions")
		passed++
	} else {
		fmt.Printf("⚠️  WARNING: Found %d test binary creations without extension handling\n", len(issues))
		for i, issue := range issues {
			if i < 3 {
				fmt.Printf("  %s\n", issue)
			}
		}
		warnings += len(issues)
	}
	fmt.Println()
}

func testPathComparisonInTests() {
	fmt.Println("Test 15: Test Path Comparisons Use filepath.ToSlash()")
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
		if !strings.HasSuffix(path, "_test.go") {
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

			// Check for path comparisons with Unix-style paths without ToSlash
			if (strings.Contains(line, "Expected") || strings.Contains(line, "expected")) &&
				strings.Contains(line, `"/`) &&
				(strings.Contains(line, "!=") || strings.Contains(line, "==")) &&
				!strings.Contains(line, "ToSlash") &&
				!strings.Contains(line, "//") {
				issues = append(issues, fmt.Sprintf("%s:%d", path, lineNum))
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if len(issues) == 0 {
		fmt.Println("✓ PASS: Path comparisons use ToSlash() normalization")
		passed++
	} else {
		fmt.Printf("⚠️  WARNING: Found %d path comparisons without ToSlash()\n", len(issues))
		for i, issue := range issues {
			if i < 3 {
				fmt.Printf("  %s\n", issue)
			}
		}
		warnings += len(issues)
	}
	fmt.Println()
}

func testPlatformSpecificErrorMessages() {
	fmt.Println("Test 16: Error Message Checks Are Platform-Agnostic")
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
		if !strings.HasSuffix(path, "_test.go") {
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

			// Check for Unix-specific error messages
			if (strings.Contains(line, `"no such file or directory"`) ||
				strings.Contains(line, `"permission denied"`) ||
				strings.Contains(line, `"Permission denied"`)) &&
				!strings.Contains(line, "//") {
				issues = append(issues, fmt.Sprintf("%s:%d", path, lineNum))
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if len(issues) == 0 {
		fmt.Println("✓ PASS: Error message checks are platform-agnostic")
		passed++
	} else {
		fmt.Printf("⚠️  WARNING: Found %d platform-specific error message checks\n", len(issues))
		for i, issue := range issues {
			if i < 3 {
				fmt.Printf("  %s\n", issue)
			}
		}
		warnings += len(issues)
	}
	fmt.Println()
}

func testBinaryExtensionStripping() {
	fmt.Println("Test 17: Binary Name Stripping Handles Both .exe and .bat")
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
		allLines := []string{}
		for scanner.Scan() {
			allLines = append(allLines, scanner.Text())
		}

		for i, line := range allLines {
			lineNum := i + 1

			// Check for TrimSuffix(".exe") without .bat handling
			if strings.Contains(line, "TrimSuffix") && strings.Contains(line, `".exe"`) &&
				!strings.Contains(line, "//") {

				// Look nearby for .bat handling
				hasBatHandling := false
				lookRange := 3
				startIdx := i - lookRange
				if startIdx < 0 {
					startIdx = 0
				}
				endIdx := i + lookRange
				if endIdx >= len(allLines) {
					endIdx = len(allLines) - 1
				}

				for j := startIdx; j <= endIdx; j++ {
					if strings.Contains(allLines[j], `".bat"`) {
						hasBatHandling = true
						break
					}
				}

				if !hasBatHandling {
					issues = append(issues, fmt.Sprintf("%s:%d", path, lineNum))
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if len(issues) == 0 {
		fmt.Println("✓ PASS: Binary extension stripping handles both .exe and .bat")
		passed++
	} else {
		fmt.Printf("⚠️  WARNING: Found %d extension stripping issues\n", len(issues))
		for i, issue := range issues {
			if i < 3 {
				fmt.Printf("  %s\n", issue)
			}
		}
		warnings += len(issues)
	}
	fmt.Println()
}

func testPermissionTestsSkipWindows() {
	fmt.Println("Test 18: Permission Tests Skip On Windows")
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
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		allLines := []string{}
		for scanner.Scan() {
			allLines = append(allLines, scanner.Text())
		}

		for i, line := range allLines {
			lineNum := i + 1

			// Check for permission-related test setup
			if (strings.Contains(line, "Chmod") || strings.Contains(line, "0444") || strings.Contains(line, "0000")) &&
				!strings.Contains(line, "//") {

				// Look for Windows skip nearby
				hasWindowsSkip := false
				lookRange := 15
				startIdx := i - lookRange
				if startIdx < 0 {
					startIdx = 0
				}
				endIdx := i + lookRange
				if endIdx >= len(allLines) {
					endIdx = len(allLines) - 1
				}

				for j := startIdx; j <= endIdx; j++ {
					testLine := allLines[j]
					if (strings.Contains(testLine, "runtime.GOOS") && strings.Contains(testLine, "windows") && strings.Contains(testLine, "Skip")) ||
						strings.Contains(testLine, "skipOnWindows") {
						hasWindowsSkip = true
						break
					}
				}

				if !hasWindowsSkip {
					issues = append(issues, fmt.Sprintf("%s:%d", path, lineNum))
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if len(issues) == 0 {
		fmt.Println("✓ PASS: Permission tests skip on Windows")
		passed++
	} else {
		fmt.Printf("⚠️  WARNING: Found %d permission tests without Windows skip\n", len(issues))
		for i, issue := range issues {
			if i < 3 {
				fmt.Printf("  %s\n", issue)
			}
		}
		warnings += len(issues)
	}
	fmt.Println()
}
func testHardcodedShebangInTests() {
	fmt.Println("Test 19: No Hardcoded Unix Shebangs Without Windows Checks")
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
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		allLines := []string{}
		for scanner.Scan() {
			allLines = append(allLines, scanner.Text())
		}

		for i, line := range allLines {
			lineNum := i + 1

			// Check for hardcoded shebangs without runtime checks
			if (strings.Contains(line, `"#!/bin/bash`) ||
				strings.Contains(line, `"#!/bin/sh`) ||
				strings.Contains(line, `"#!/usr/bin/env bash`) ||
				strings.Contains(line, `"#!/usr/bin/env sh`)) &&
				!strings.Contains(line, "//") {

				// Check if runtime.GOOS exists nearby
				hasRuntimeCheck := false
				lookRange := 10
				startIdx := i - lookRange
				if startIdx < 0 {
					startIdx = 0
				}
				endIdx := i + lookRange
				if endIdx >= len(allLines) {
					endIdx = len(allLines) - 1
				}

				for j := startIdx; j <= endIdx; j++ {
					if strings.Contains(allLines[j], "runtime.GOOS") && strings.Contains(allLines[j], "windows") {
						hasRuntimeCheck = true
						break
					}
				}

				if !hasRuntimeCheck {
					issues = append(issues, fmt.Sprintf("%s:%d: Hardcoded Unix shebang without Windows check", path, lineNum))
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if len(issues) == 0 {
		fmt.Println("✓ PASS: No hardcoded Unix shebangs found")
		passed++
	} else {
		fmt.Printf("⚠️  WARNING: Found %d hardcoded Unix shebangs without Windows checks\n", len(issues))
		for i, issue := range issues {
			if i < 3 {
				fmt.Printf("  %s\n", issue)
			}
		}
		warnings += len(issues)
	}
	fmt.Println()
}

func testShimCreationPatterns() {
	fmt.Println("Test 20: Shim Creation Uses Platform-Specific Patterns")
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
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		allLines := []string{}
		for scanner.Scan() {
			allLines = append(allLines, scanner.Text())
		}

		for i, line := range allLines {
			lineNum := i + 1

			// Check for shim-related WriteFile without Windows handling
			if strings.Contains(line, "WriteFile") &&
				(strings.Contains(line, "shim") || strings.Contains(line, "Shim")) &&
				!strings.Contains(line, "//") {

				// Check if runtime.GOOS or .bat exists nearby
				hasRuntimeCheck := false
				lookRange := 10
				startIdx := i - lookRange
				if startIdx < 0 {
					startIdx = 0
				}
				endIdx := i + lookRange
				if endIdx >= len(allLines) {
					endIdx = len(allLines) - 1
				}

				for j := startIdx; j <= endIdx; j++ {
					if (strings.Contains(allLines[j], "runtime.GOOS") && strings.Contains(allLines[j], "windows")) ||
						strings.Contains(allLines[j], ".bat") ||
						strings.Contains(allLines[j], "@echo off") {
						hasRuntimeCheck = true
						break
					}
				}

				if !hasRuntimeCheck {
					issues = append(issues, fmt.Sprintf("%s:%d: Shim creation without Windows handling", path, lineNum))
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if len(issues) == 0 {
		fmt.Println("✓ PASS: All shim creation uses platform-specific patterns")
		passed++
	} else {
		fmt.Printf("⚠️  WARNING: Found %d shim creations without Windows handling\n", len(issues))
		for i, issue := range issues {
			if i < 3 {
				fmt.Printf("  %s\n", issue)
			}
		}
		warnings += len(issues)
	}
	fmt.Println()
}
func testExtensionStrippingForDisplay() {
	fmt.Println("Test 21: Extension Stripping For Display on Windows")
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
		allLines := []string{}
		for scanner.Scan() {
			allLines = append(allLines, scanner.Text())
		}

		for i, line := range allLines {
			lineNum := i + 1

			// Check for functions that return shim/binary names without stripping extensions
			if (strings.Contains(line, "return") && strings.Contains(line, "entry.Name()")) ||
				(strings.Contains(line, "append") && strings.Contains(line, "entry.Name()")) ||
				(strings.Contains(line, "Fprintln") && strings.Contains(line, "foundPath")) {

				// Look for Windows extension stripping nearby
				hasExtensionStripping := false
				lookRange := 10
				startIdx := i - lookRange
				if startIdx < 0 {
					startIdx = 0
				}
				endIdx := i + lookRange
				if endIdx >= len(allLines) {
					endIdx = len(allLines) - 1
				}

				for j := startIdx; j <= endIdx; j++ {
					if (strings.Contains(allLines[j], "TrimSuffix") &&
						(strings.Contains(allLines[j], ".exe") || strings.Contains(allLines[j], ".bat"))) ||
						(strings.Contains(allLines[j], "runtime.GOOS") &&
							strings.Contains(allLines[j], "windows") &&
							strings.Contains(allLines[j], "TrimSuffix")) {
						hasExtensionStripping = true
						break
					}
				}

				// Check if this is in a function that returns binary/shim names
				inRelevantFunction := false
				for j := i; j >= 0 && j > i-50; j-- {
					if strings.Contains(allLines[j], "func") &&
						(strings.Contains(allLines[j], "ListShims") ||
							strings.Contains(allLines[j], "Whence") ||
							strings.Contains(allLines[j], "Which") ||
							strings.Contains(allLines[j], "FindBinary")) {
						inRelevantFunction = true
						break
					}
				}

				if inRelevantFunction && !hasExtensionStripping {
					issues = append(issues, fmt.Sprintf("%s:%d: Binary/shim name may need extension stripping on Windows", path, lineNum))
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if len(issues) == 0 {
		fmt.Println("✓ PASS: All binary/shim names strip extensions appropriately")
		passed++
	} else {
		fmt.Printf("⚠️  INFO: Found %d potential extension stripping issues\n", len(issues))
		for i, issue := range issues {
			if i < 3 {
				fmt.Printf("  %s\n", issue)
			}
		}
		// Don't count as warnings, just informational
	}
	fmt.Println()
}

func testExeFilesWithScriptContent() {
	fmt.Println("Test 22: .exe Files Should Not Have Script Content")
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
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		allLines := []string{}
		for scanner.Scan() {
			allLines = append(allLines, scanner.Text())
		}

		for i, line := range allLines {
			lineNum := i + 1

			// Look for += ".exe" assignments
			if strings.Contains(line, `+= ".exe"`) {
				// Check if nearby WriteFile has script content (shebang or @echo off)
				lookRange := 15
				startIdx := i - 5
				if startIdx < 0 {
					startIdx = 0
				}
				endIdx := i + lookRange
				if endIdx >= len(allLines) {
					endIdx = len(allLines) - 1
				}

				hasScriptContent := false
				for j := startIdx; j <= endIdx; j++ {
					if strings.Contains(allLines[j], "WriteFile") {
						// Look at content being written
						for k := j - 5; k <= j+5 && k < len(allLines); k++ {
							if k < 0 {
								continue
							}
							if strings.Contains(allLines[k], "#!/") ||
								strings.Contains(allLines[k], "@echo off") ||
								strings.Contains(allLines[k], "exit 0") ||
								strings.Contains(allLines[k], "exit 1") {
								hasScriptContent = true
								break
							}
						}
					}
					if hasScriptContent {
						break
					}
				}

				if hasScriptContent {
					issues = append(issues, fmt.Sprintf("%s:%d: .exe file created with script content (use .bat for text scripts)", path, lineNum))
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if len(issues) == 0 {
		fmt.Println("✓ PASS: No .exe files with script content found")
		passed++
	} else {
		fmt.Printf("⚠️  WARNING: Found %d .exe files with script content\n", len(issues))
		for i, issue := range issues {
			if i < 3 {
				fmt.Printf("  %s\n", issue)
			}
		}
		warnings += len(issues)
	}
	fmt.Println()
}

func testBatFilesProperSyntax() {
	fmt.Println("Test 23: .bat Files Should Have Proper Syntax")
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
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		allLines := []string{}
		for scanner.Scan() {
			allLines = append(allLines, scanner.Text())
		}

		for i, line := range allLines {
			lineNum := i + 1

			// Look for += ".bat" assignments
			if strings.Contains(line, `+= ".bat"`) || strings.Contains(line, `.bat"`) {
				// Check if nearby WriteFile has proper batch syntax
				lookRange := 10
				startIdx := i - 5
				if startIdx < 0 {
					startIdx = 0
				}
				endIdx := i + lookRange
				if endIdx >= len(allLines) {
					endIdx = len(allLines) - 1
				}

				hasEchoOff := false
				hasWriteFile := false
				for j := startIdx; j <= endIdx; j++ {
					if strings.Contains(allLines[j], "WriteFile") {
						hasWriteFile = true
						// Look at content being written
						for k := j - 10; k <= j+5 && k < len(allLines); k++ {
							if k < 0 {
								continue
							}
							if strings.Contains(allLines[k], "@echo off") {
								hasEchoOff = true
								break
							}
						}
					}
				}

				if hasWriteFile && !hasEchoOff {
					issues = append(issues, fmt.Sprintf("%s:%d: .bat file may be missing '@echo off' at start", path, lineNum))
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if len(issues) == 0 {
		fmt.Println("✓ PASS: All .bat files have proper syntax")
		passed++
	} else {
		fmt.Printf("⚠️  INFO: Found %d potential .bat syntax issues\n", len(issues))
		for i, issue := range issues {
			if i < 3 {
				fmt.Printf("  %s\n", issue)
			}
		}
		// Informational only
	}
	fmt.Println()
}

func testBatchEchoQuotes() {
	fmt.Println("Test 24: Batch Echo Should Not Use Quotes")
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
		if !strings.HasSuffix(path, "_test.go") {
			return nil
		}

		file, err := os.Open(path)
		if err != nil {
			return nil
		}
		defer file.Close()

		scanner := bufio.NewScanner(file)
		allLines := []string{}
		for scanner.Scan() {
			allLines = append(allLines, scanner.Text())
		}

		for i, line := range allLines {
			lineNum := i + 1

			// Look for echo " in content assigned to batch files
			if strings.Contains(line, `echo "`) {
				// Check if we're in a Windows batch context
				isInBatchContext := false
				lookRange := 10
				startIdx := i - lookRange
				if startIdx < 0 {
					startIdx = 0
				}
				endIdx := i + lookRange
				if endIdx >= len(allLines) {
					endIdx = len(allLines) - 1
				}

				for j := startIdx; j <= endIdx; j++ {
					if strings.Contains(allLines[j], "@echo off") ||
						(strings.Contains(allLines[j], `".bat"`) && strings.Contains(allLines[j], "WriteFile")) {
						isInBatchContext = true
						break
					}
				}

				if isInBatchContext && !strings.Contains(line, "//") {
					issues = append(issues, fmt.Sprintf("%s:%d: Batch echo includes quotes in output (remove quotes)", path, lineNum))
				}
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if len(issues) == 0 {
		fmt.Println("✓ PASS: No batch echo quote issues found")
		passed++
	} else {
		fmt.Printf("⚠️  WARNING: Found %d batch echo quote issues\n", len(issues))
		for i, issue := range issues {
			if i < 3 {
				fmt.Printf("  %s\n", issue)
			}
		}
		warnings += len(issues)
	}
	fmt.Println()
}

func testContainsPathComparisons() {
	fmt.Println("Test 25: Contains/HasPrefix/HasSuffix Path Comparisons")
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
		if !strings.HasSuffix(path, "_test.go") {
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

			// Check for path comparisons with forward slashes without ToSlash
			if (strings.Contains(line, ".Contains(") ||
				strings.Contains(line, ".HasPrefix(") ||
				strings.Contains(line, ".HasSuffix(")) &&
				(strings.Contains(line, `"/`) || strings.Contains(line, `"\`)) &&
				!strings.Contains(line, "ToSlash") &&
				!strings.Contains(line, "//") {
				issues = append(issues, fmt.Sprintf("%s:%d", path, lineNum))
			}
		}
		return nil
	})

	if err != nil {
		fmt.Printf("⚠️  WARNING: Error scanning files: %v\n", err)
		warnings++
	} else if len(issues) == 0 {
		fmt.Println("✓ PASS: All string path comparisons look good")
		passed++
	} else {
		fmt.Printf("⚠️  INFO: Found %d path comparisons that may need ToSlash()\n", len(issues))
		for i, issue := range issues {
			if i < 3 {
				fmt.Printf("  %s\n", issue)
			}
		}
		// Informational only
	}
	fmt.Println()
}
