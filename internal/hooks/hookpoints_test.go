package hooks

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestHookPointString(t *testing.T) {
	tests := []struct {
		name      string
		hookPoint HookPoint
		expected  string
	}{
		{"PreInstall", PreInstall, "pre_install"},
		{"PostInstall", PostInstall, "post_install"},
		{"PreUninstall", PreUninstall, "pre_uninstall"},
		{"PostUninstall", PostUninstall, "post_uninstall"},
		{"PreExec", PreExec, "pre_exec"},
		{"PostExec", PostExec, "post_exec"},
		{"PreRehash", PreRehash, "pre_rehash"},
		{"PostRehash", PostRehash, "post_rehash"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tt.hookPoint.String(); got != tt.expected {
				t.Errorf("HookPoint.String() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestAllHookPoints(t *testing.T) {
	hookPoints := AllHookPoints()

	// Should have exactly 8 hook points
	assert.Len(t, hookPoints, 8, "AllHookPoints() returned hook points")

	// Verify all expected hook points are present
	expected := []HookPoint{
		PreInstall, PostInstall,
		PreUninstall, PostUninstall,
		PreExec, PostExec,
		PreRehash, PostRehash,
	}

	for _, exp := range expected {
		found := false
		for _, hp := range hookPoints {
			if hp == exp {
				found = true
				break
			}
		}
		assert.True(t, found, "AllHookPoints() missing expected hook point")
	}
}

func TestIsValidHookPoint(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		{"Valid: pre_install", "pre_install", true},
		{"Valid: post_install", "post_install", true},
		{"Valid: pre_uninstall", "pre_uninstall", true},
		{"Valid: post_uninstall", "post_uninstall", true},
		{"Valid: pre_exec", "pre_exec", true},
		{"Valid: post_exec", "post_exec", true},
		{"Valid: pre_rehash", "pre_rehash", true},
		{"Valid: post_rehash", "post_rehash", true},
		{"Invalid: empty string", "", false},
		{"Invalid: wrong case", "PRE_INSTALL", false},
		{"Invalid: typo", "pre_instal", false},
		{"Invalid: unknown", "pre_update", false},
		{"Invalid: partial match", "pre", false},
		{"Invalid: with spaces", "pre install", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsValidHookPoint(tt.input); got != tt.expected {
				t.Errorf("IsValidHookPoint(%q) = %v, want %v", tt.input, got, tt.expected)
			}
		})
	}
}

func TestHookPointConstants(t *testing.T) {
	// Verify constant values match expected strings
	tests := []struct {
		constant HookPoint
		value    string
	}{
		{PreInstall, "pre_install"},
		{PostInstall, "post_install"},
		{PreUninstall, "pre_uninstall"},
		{PostUninstall, "post_uninstall"},
		{PreExec, "pre_exec"},
		{PostExec, "post_exec"},
		{PreRehash, "pre_rehash"},
		{PostRehash, "post_rehash"},
	}

	for _, tt := range tests {
		assert.Equal(t, tt.value, string(tt.constant), "HookPoint constant has value %v", tt.constant)
	}
}
