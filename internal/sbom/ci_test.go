package sbom

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestDetectCIPlatform(t *testing.T) {
	tests := []struct {
		name     string
		envVars  map[string]string
		expected CIPlatform
	}{
		{
			name:     "GitHub Actions",
			envVars:  map[string]string{"GITHUB_ACTIONS": "true"},
			expected: PlatformGitHubActions,
		},
		{
			name:     "GitLab CI",
			envVars:  map[string]string{"GITLAB_CI": "true"},
			expected: PlatformGitLabCI,
		},
		{
			name:     "CircleCI",
			envVars:  map[string]string{"CIRCLECI": "true"},
			expected: PlatformCircleCI,
		},
		{
			name:     "Jenkins",
			envVars:  map[string]string{"JENKINS_HOME": "/var/jenkins"},
			expected: PlatformJenkins,
		},
		{
			name:     "Azure Pipelines",
			envVars:  map[string]string{"TF_BUILD": "True"},
			expected: PlatformAzurePipelines,
		},
		{
			name:     "Unknown",
			envVars:  map[string]string{},
			expected: PlatformUnknown,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clear all CI env vars first
			clearCIEnvVars()

			// Set test env vars
			for k, v := range tt.envVars {
				os.Setenv(k, v)
				defer os.Unsetenv(k)
			}

			got := DetectCIPlatform()
			if got != tt.expected {
				t.Errorf("DetectCIPlatform() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestNewCIChecker(t *testing.T) {
	tests := []struct {
		name        string
		projectRoot string
		wantNil     bool
	}{
		{
			name:        "with project root",
			projectRoot: "/tmp/project",
			wantNil:     false,
		},
		{
			name:        "empty project root uses cwd",
			projectRoot: "",
			wantNil:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewCIChecker(tt.projectRoot)

			if (checker == nil) != tt.wantNil {
				t.Errorf("NewCIChecker() nil = %v, wantNil %v", checker == nil, tt.wantNil)
			}

			if !tt.wantNil {
				if tt.projectRoot != "" && checker.ProjectRoot != tt.projectRoot {
					t.Errorf("ProjectRoot = %v, want %v", checker.ProjectRoot, tt.projectRoot)
				}
			}
		})
	}
}

func TestCIChecker_CheckSBOM(t *testing.T) {
	tests := []struct {
		name            string
		setup           func(t *testing.T) (string, string) // Returns projectRoot, sbomPath
		maxAge          time.Duration
		wantPassed      bool
		wantExists      bool
		wantStale       bool
		wantStaleReason string
	}{
		{
			name: "SBOM exists and is fresh",
			setup: func(t *testing.T) (string, string) {
				dir := t.TempDir()

				// Create go.mod (older)
				goModPath := filepath.Join(dir, "go.mod")
				os.WriteFile(goModPath, []byte("module test\n"), 0644)
				time.Sleep(10 * time.Millisecond)

				// Create SBOM (newer)
				sbomPath := filepath.Join(dir, "sbom.json")
				sbom := SimpleSBOM{Format: "CycloneDX", SpecVersion: "1.4"}
				data, _ := json.Marshal(sbom)
				os.WriteFile(sbomPath, data, 0644)

				return dir, sbomPath
			},
			maxAge:     0,
			wantPassed: true,
			wantExists: true,
			wantStale:  false,
		},
		{
			name: "SBOM does not exist",
			setup: func(t *testing.T) (string, string) {
				dir := t.TempDir()
				return dir, ""
			},
			maxAge:     0,
			wantPassed: false,
			wantExists: false,
			wantStale:  false,
		},
		{
			name: "SBOM is stale (go.mod newer)",
			setup: func(t *testing.T) (string, string) {
				dir := t.TempDir()

				// Create SBOM (older)
				sbomPath := filepath.Join(dir, "sbom.json")
				sbom := SimpleSBOM{Format: "CycloneDX", SpecVersion: "1.4"}
				data, _ := json.Marshal(sbom)
				os.WriteFile(sbomPath, data, 0644)
				time.Sleep(10 * time.Millisecond)

				// Create go.mod (newer)
				goModPath := filepath.Join(dir, "go.mod")
				os.WriteFile(goModPath, []byte("module test\n"), 0644)

				return dir, sbomPath
			},
			maxAge:          0,
			wantPassed:      false,
			wantExists:      true,
			wantStale:       true,
			wantStaleReason: "go.mod",
		},
		{
			name: "SBOM is stale (go.sum newer)",
			setup: func(t *testing.T) (string, string) {
				dir := t.TempDir()

				// Create SBOM (older)
				sbomPath := filepath.Join(dir, "sbom.json")
				sbom := SimpleSBOM{Format: "CycloneDX", SpecVersion: "1.4"}
				data, _ := json.Marshal(sbom)
				os.WriteFile(sbomPath, data, 0644)
				time.Sleep(10 * time.Millisecond)

				// Create go.sum (newer)
				goSumPath := filepath.Join(dir, "go.sum")
				os.WriteFile(goSumPath, []byte(""), 0644)

				return dir, sbomPath
			},
			maxAge:          0,
			wantPassed:      false,
			wantExists:      true,
			wantStale:       true,
			wantStaleReason: "go.sum",
		},
		{
			name: "SBOM exceeds max age",
			setup: func(t *testing.T) (string, string) {
				dir := t.TempDir()

				// Create old SBOM
				sbomPath := filepath.Join(dir, "sbom.json")
				sbom := SimpleSBOM{Format: "CycloneDX", SpecVersion: "1.4"}
				data, _ := json.Marshal(sbom)
				os.WriteFile(sbomPath, data, 0644)

				// Make it old
				oldTime := time.Now().Add(-48 * time.Hour)
				os.Chtimes(sbomPath, oldTime, oldTime)

				return dir, sbomPath
			},
			maxAge:     24 * time.Hour,
			wantPassed: false,
			wantExists: true,
			wantStale:  true,
		},
		{
			name: "Auto-detect SBOM file",
			setup: func(t *testing.T) (string, string) {
				dir := t.TempDir()

				// Create go.mod
				goModPath := filepath.Join(dir, "go.mod")
				os.WriteFile(goModPath, []byte("module test\n"), 0644)
				time.Sleep(10 * time.Millisecond)

				// Create SBOM with standard name
				sbomPath := filepath.Join(dir, "sbom.json")
				sbom := SimpleSBOM{Format: "CycloneDX", SpecVersion: "1.4"}
				data, _ := json.Marshal(sbom)
				os.WriteFile(sbomPath, data, 0644)

				return dir, "" // Empty path to trigger auto-detection
			},
			maxAge:     0,
			wantPassed: true,
			wantExists: true,
			wantStale:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			projectRoot, sbomPath := tt.setup(t)

			checker := NewCIChecker(projectRoot)
			result, err := checker.CheckSBOM(sbomPath, tt.maxAge)

			if err != nil {
				t.Fatalf("CheckSBOM() error = %v", err)
			}

			if result.Passed != tt.wantPassed {
				t.Errorf("CheckSBOM() Passed = %v, want %v", result.Passed, tt.wantPassed)
			}

			if result.SBOMExists != tt.wantExists {
				t.Errorf("CheckSBOM() SBOMExists = %v, want %v", result.SBOMExists, tt.wantExists)
			}

			if result.IsStale != tt.wantStale {
				t.Errorf("CheckSBOM() IsStale = %v, want %v", result.IsStale, tt.wantStale)
			}

			if tt.wantStaleReason != "" && !strings.Contains(result.StaleReason, tt.wantStaleReason) {
				t.Errorf("CheckSBOM() StaleReason = %q, want to contain %q",
					result.StaleReason, tt.wantStaleReason)
			}

			// Verify recommendations are provided when not passed
			if !result.Passed && len(result.Recommendations) == 0 {
				t.Error("CheckSBOM() should provide recommendations when check fails")
			}
		})
	}
}

func TestCIChecker_FormatCIOutput(t *testing.T) {
	tests := []struct {
		name         string
		platform     CIPlatform
		result       *CICheckResult
		wantContains []string
	}{
		{
			name:     "GitHub Actions - passed",
			platform: PlatformGitHubActions,
			result: &CICheckResult{
				Passed:     true,
				SBOMExists: true,
				SBOMPath:   "sbom.json",
			},
			wantContains: []string{"::notice", "SBOM Valid"},
		},
		{
			name:     "GitHub Actions - SBOM missing",
			platform: PlatformGitHubActions,
			result: &CICheckResult{
				Passed:     false,
				SBOMExists: false,
			},
			wantContains: []string{"::error", "SBOM Missing"},
		},
		{
			name:     "GitHub Actions - SBOM stale",
			platform: PlatformGitHubActions,
			result: &CICheckResult{
				Passed:      false,
				SBOMExists:  true,
				IsStale:     true,
				SBOMPath:    "sbom.json",
				StaleReason: "go.mod modified",
			},
			wantContains: []string{"::error", "SBOM Stale", "go.mod"},
		},
		{
			name:     "GitLab CI - passed",
			platform: PlatformGitLabCI,
			result: &CICheckResult{
				Passed:     true,
				SBOMExists: true,
			},
			wantContains: []string{"✅", "passed"},
		},
		{
			name:     "GitLab CI - failed",
			platform: PlatformGitLabCI,
			result: &CICheckResult{
				Passed:      false,
				SBOMExists:  true,
				IsStale:     true,
				StaleReason: "go.mod modified",
			},
			wantContains: []string{"❌", "failed", "go.mod"},
		},
		{
			name:     "Unknown platform - passed",
			platform: PlatformUnknown,
			result: &CICheckResult{
				Passed:     true,
				SBOMExists: true,
				SBOMPath:   "sbom.json",
			},
			wantContains: []string{"✅", "PASSED", "sbom.json"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &CIChecker{
				ProjectRoot: "/tmp/test",
				Platform:    tt.platform,
			}

			output := checker.FormatCIOutput(tt.result)

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("FormatCIOutput() missing %q in output:\n%s", want, output)
				}
			}
		})
	}
}

func TestCIChecker_EvaluateScanResult(t *testing.T) {
	tests := []struct {
		name       string
		scanResult *ScanResult
		failOn     string
		wantPass   bool
	}{
		{
			name: "no vulnerabilities",
			scanResult: &ScanResult{
				Summary: VulnerabilitySummary{
					Total: 0,
				},
			},
			failOn:   "high",
			wantPass: true,
		},
		{
			name: "critical found, threshold critical",
			scanResult: &ScanResult{
				Summary: VulnerabilitySummary{
					Total:    1,
					Critical: 1,
				},
			},
			failOn:   "critical",
			wantPass: false,
		},
		{
			name: "critical found, threshold high",
			scanResult: &ScanResult{
				Summary: VulnerabilitySummary{
					Total:    1,
					Critical: 1,
				},
			},
			failOn:   "high",
			wantPass: false,
		},
		{
			name: "high found, threshold critical",
			scanResult: &ScanResult{
				Summary: VulnerabilitySummary{
					Total: 1,
					High:  1,
				},
			},
			failOn:   "critical",
			wantPass: true,
		},
		{
			name: "medium found, threshold high",
			scanResult: &ScanResult{
				Summary: VulnerabilitySummary{
					Total:  1,
					Medium: 1,
				},
			},
			failOn:   "high",
			wantPass: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := NewCIChecker("")

			gotPass := checker.evaluateScanResult(tt.scanResult, tt.failOn)
			if gotPass != tt.wantPass {
				t.Errorf("evaluateScanResult() = %v, want %v", gotPass, tt.wantPass)
			}
		})
	}
}

func TestCIChecker_FormatScanOutput(t *testing.T) {
	scanResult := &CIScanResult{
		Scanner:  "grype",
		ScanTime: time.Now(),
		Passed:   false,
		Summary: &VulnerabilitySummary{
			Total:    10,
			Critical: 2,
			High:     3,
			Medium:   4,
			Low:      1,
		},
		Vulnerabilities: []Vulnerability{
			{
				ID:             "CVE-2024-1234",
				PackageName:    "golang.org/x/crypto",
				PackageVersion: "0.1.0",
				Severity:       "critical",
				Description:    "Test vulnerability",
			},
		},
	}

	tests := []struct {
		name         string
		platform     CIPlatform
		wantContains []string
	}{
		{
			name:     "GitHub Actions",
			platform: PlatformGitHubActions,
			wantContains: []string{
				"::error",
				"CVE-2024-1234",
				"golang.org/x/crypto",
			},
		},
		{
			name:     "GitLab CI",
			platform: PlatformGitLabCI,
			wantContains: []string{
				"Vulnerability Scan Results",
				"grype",
				"Critical: 2",
				"❌",
			},
		},
		{
			name:     "Unknown platform",
			platform: PlatformUnknown,
			wantContains: []string{
				"Vulnerability Scan Results",
				"grype",
				"🔴 Critical: 2",
				"❌ Result: FAILED",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			checker := &CIChecker{
				Platform: tt.platform,
			}

			output := checker.FormatScanOutput(scanResult)

			for _, want := range tt.wantContains {
				if !strings.Contains(output, want) {
					t.Errorf("FormatScanOutput() missing %q in output:\n%s", want, output)
				}
			}
		})
	}
}

func TestCIChecker_WriteScanResultToFile(t *testing.T) {
	checker := NewCIChecker("")

	result := &CIScanResult{
		Scanner:  "grype",
		ScanTime: time.Now(),
		Passed:   true,
		Summary: &VulnerabilitySummary{
			Total: 0,
		},
	}

	outputPath := filepath.Join(t.TempDir(), "result.json")

	err := checker.WriteScanResultToFile(result, outputPath)
	if err != nil {
		t.Fatalf("WriteScanResultToFile() error = %v", err)
	}

	// Verify file exists and is valid JSON
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read output file: %v", err)
	}

	var loaded CIScanResult
	if err := json.Unmarshal(data, &loaded); err != nil {
		t.Fatalf("output is not valid JSON: %v", err)
	}

	if loaded.Scanner != result.Scanner {
		t.Errorf("loaded scanner = %q, want %q", loaded.Scanner, result.Scanner)
	}
}

func TestCIChecker_ExportToGitHubSARIF(t *testing.T) {
	checker := NewCIChecker("")

	result := &CIScanResult{
		Scanner:  "grype",
		ScanTime: time.Now(),
		Vulnerabilities: []Vulnerability{
			{
				ID:             "CVE-2024-1234",
				PackageName:    "golang.org/x/crypto",
				PackageVersion: "0.1.0",
				Severity:       "critical",
				Description:    "Test vulnerability",
				FixedInVersion: "0.2.0",
			},
		},
	}

	outputPath := filepath.Join(t.TempDir(), "results.sarif")

	err := checker.ExportToGitHubSARIF(result, outputPath)
	if err != nil {
		t.Fatalf("ExportToGitHubSARIF() error = %v", err)
	}

	// Verify file exists and is valid JSON
	data, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("failed to read SARIF file: %v", err)
	}

	var sarif map[string]interface{}
	if err := json.Unmarshal(data, &sarif); err != nil {
		t.Fatalf("SARIF is not valid JSON: %v", err)
	}

	// Verify SARIF structure
	if sarif["version"] != "2.1.0" {
		t.Errorf("SARIF version = %v, want 2.1.0", sarif["version"])
	}

	runs, ok := sarif["runs"].([]interface{})
	if !ok || len(runs) == 0 {
		t.Fatal("SARIF missing runs array")
	}

	run := runs[0].(map[string]interface{})
	results, ok := run["results"].([]interface{})
	if !ok || len(results) == 0 {
		t.Fatal("SARIF run missing results")
	}

	// Verify vulnerability was converted
	firstResult := results[0].(map[string]interface{})
	if firstResult["ruleId"] != "CVE-2024-1234" {
		t.Errorf("SARIF result ruleId = %v, want CVE-2024-1234", firstResult["ruleId"])
	}
}

// Helper function to clear CI environment variables
func clearCIEnvVars() {
	ciVars := []string{
		"GITHUB_ACTIONS",
		"GITLAB_CI",
		"CIRCLECI",
		"JENKINS_HOME",
		"TF_BUILD",
	}
	for _, v := range ciVars {
		os.Unsetenv(v)
	}
}
