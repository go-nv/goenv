package cmd

import (
	"strings"
	"testing"
)

// TestVSCodeTemplatesUseTilde verifies that templates use ~ instead of ${env:HOME} for Windows compatibility
func TestVSCodeTemplatesUseTilde(t *testing.T) {
	templates := []string{"basic", "advanced", "monorepo"}

	for _, template := range templates {
		t.Run(template, func(t *testing.T) {
			settings, err := generateSettings(template)
			if err != nil {
				t.Fatalf("Failed to generate %s template: %v", template, err)
			}

			// Verify toolsGopath uses tilde (cross-platform)
			toolsGopath, ok := settings["go.toolsGopath"].(string)
			if !ok {
				t.Fatal("go.toolsGopath not found in template")
			}

			if toolsGopath != "~/go/tools" {
				t.Errorf("Template '%s': Expected '~/go/tools', got '%s'", template, toolsGopath)
			}

			// Should NOT use ${env:HOME} which doesn't exist on Windows
			if strings.Contains(toolsGopath, "${env:HOME}") {
				t.Errorf("Template '%s' uses ${env:HOME} which is not Windows-compatible", template)
			}

			t.Logf("✓ Template '%s' uses cross-platform home directory: %s", template, toolsGopath)
		})
	}
}

// TestVSCodeGenerateSettingsBasic verifies the basic template structure
func TestVSCodeGenerateSettingsBasic(t *testing.T) {
	settings, err := generateSettings("basic")
	if err != nil {
		t.Fatalf("Failed to generate basic template: %v", err)
	}

	// Verify required keys exist
	requiredKeys := []string{"go.goroot", "go.gopath", "go.toolsGopath", "go.useLanguageServer"}
	for _, key := range requiredKeys {
		if _, ok := settings[key]; !ok {
			t.Errorf("Basic template missing required key: %s", key)
		}
	}

	// Verify environment variable syntax for GOROOT and GOPATH
	if settings["go.goroot"] != "${env:GOROOT}" {
		t.Errorf("Expected go.goroot to be '${env:GOROOT}', got '%v'", settings["go.goroot"])
	}

	if settings["go.gopath"] != "${env:GOPATH}" {
		t.Errorf("Expected go.gopath to be '${env:GOPATH}', got '%v'", settings["go.gopath"])
	}
}

// TestVSCodeGenerateSettingsAdvanced verifies the advanced template
func TestVSCodeGenerateSettingsAdvanced(t *testing.T) {
	settings, err := generateSettings("advanced")
	if err != nil {
		t.Fatalf("Failed to generate advanced template: %v", err)
	}

	// Verify gopls settings exist
	if _, ok := settings["gopls"]; !ok {
		t.Error("Advanced template missing gopls configuration")
	}

	// Verify autoUpdate is enabled
	if settings["go.toolsManagement.autoUpdate"] != true {
		t.Error("Advanced template should have autoUpdate enabled")
	}
}

// TestVSCodeGenerateSettingsMonorepo verifies the monorepo template
func TestVSCodeGenerateSettingsMonorepo(t *testing.T) {
	settings, err := generateSettings("monorepo")
	if err != nil {
		t.Fatalf("Failed to generate monorepo template: %v", err)
	}

	// Verify inferGopath is disabled for monorepos
	if settings["go.inferGopath"] != false {
		t.Error("Monorepo template should have inferGopath disabled")
	}

	// Verify gopls directory filters exist
	gopls, ok := settings["gopls"].(map[string]interface{})
	if !ok {
		t.Fatal("gopls settings not found in monorepo template")
	}

	if _, ok := gopls["build.directoryFilters"]; !ok {
		t.Error("Monorepo template missing build.directoryFilters")
	}
}

// TestVSCodeGenerateSettingsInvalidTemplate verifies error handling
func TestVSCodeGenerateSettingsInvalidTemplate(t *testing.T) {
	_, err := generateSettings("invalid")
	if err == nil {
		t.Error("Expected error for invalid template, got nil")
	}

	if !strings.Contains(err.Error(), "unknown template") {
		t.Errorf("Expected 'unknown template' error, got: %v", err)
	}
}

// TestVSCodeMergeSettings verifies settings merging logic
func TestVSCodeMergeSettings(t *testing.T) {
	existing := VSCodeSettings{
		"go.goroot":      "/old/path",
		"custom.setting": "value",
	}

	new := VSCodeSettings{
		"go.goroot": "${env:GOROOT}",
		"go.gopath": "${env:GOPATH}",
	}

	merged := mergeSettings(existing, new)

	// Should preserve existing go.goroot
	if merged["go.goroot"] != "/old/path" {
		t.Errorf("Expected existing go.goroot to be preserved, got '%v'", merged["go.goroot"])
	}

	// Should add new go.gopath
	if merged["go.gopath"] != "${env:GOPATH}" {
		t.Errorf("Expected new go.gopath to be added, got '%v'", merged["go.gopath"])
	}

	// Should preserve custom setting
	if merged["custom.setting"] != "value" {
		t.Error("Custom setting was not preserved")
	}
}

// TestVSCodeMergeSettingsWithOverride verifies override logic
func TestVSCodeMergeSettingsWithOverride(t *testing.T) {
	existing := VSCodeSettings{
		"go.goroot":      "/old/path",
		"go.gopath":      "/old/gopath",
		"custom.setting": "value",
	}

	new := VSCodeSettings{
		"go.goroot": "/new/path",
		"go.gopath": "/new/gopath",
	}

	overrideKeys := []string{"go.goroot", "go.gopath"}
	merged := mergeSettingsWithOverride(existing, new, overrideKeys)

	// Should override go.goroot
	if merged["go.goroot"] != "/new/path" {
		t.Errorf("Expected go.goroot to be overridden, got '%v'", merged["go.goroot"])
	}

	// Should override go.gopath
	if merged["go.gopath"] != "/new/gopath" {
		t.Errorf("Expected go.gopath to be overridden, got '%v'", merged["go.gopath"])
	}

	// Should preserve custom setting
	if merged["custom.setting"] != "value" {
		t.Error("Custom setting was not preserved")
	}
}
