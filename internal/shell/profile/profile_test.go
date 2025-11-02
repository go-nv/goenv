package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetInitLine(t *testing.T) {
	tests := []struct {
		shell    ShellType
		expected string
	}{
		{ShellTypeBash, "eval \"$(goenv init -)\""},
		{ShellTypeZsh, "eval \"$(goenv init -)\""},
		{ShellTypeFish, "status --is-interactive; and source (goenv init -|psub)"},
		{ShellTypePowerShell, "Invoke-Expression (goenv init - | Out-String)"},
		{ShellTypeCmd, "FOR /f \"tokens=*\" %%i IN ('goenv init -') DO @%%i"},
	}

	for _, tt := range tests {
		t.Run(string(tt.shell), func(t *testing.T) {
			pm := NewProfileManager(tt.shell)
			got := pm.GetInitLine()
			assert.Equal(t, tt.expected, got, "GetInitLine() =")
		})
	}
}

func TestHasGoenvInit(t *testing.T) {
	tests := []struct {
		name     string
		content  string
		expected bool
	}{
		{"empty content", "", false},
		{"has goenv init", "eval \"$(goenv init -)\"", true},
		{"has GOENV_ROOT", "export GOENV_ROOT=$HOME/.goenv", true},
		{"has goenv/shims", "export PATH=$HOME/.goenv/shims:$PATH", true},
		{"no goenv", "export PATH=$HOME/bin:$PATH", false},
	}

	pm := NewProfileManager(ShellTypeBash)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := pm.hasGoenvInit(tt.content)
			assert.Equal(t, tt.expected, got, "hasGoenvInit() =")
		})
	}
}

func TestGetProfile(t *testing.T) {
	var err error
	// This test is Unix-specific (tests .bashrc)
	if utils.IsWindows() {
		t.Skip("Skipping Unix shell profile test on Windows")
	}

	// Create temporary test directory
	tmpDir := t.TempDir()
	home := filepath.Join(tmpDir, "home")
	err = utils.EnsureDirWithContext(home, "create test directory")
	require.NoError(t, err)

	// Override home directory for testing
	oldHome := os.Getenv(utils.EnvVarHome)
	os.Setenv(utils.EnvVarHome, home)
	defer os.Setenv(utils.EnvVarHome, oldHome)

	// Create a test profile
	testContent := "eval \"$(goenv init -)\"\n"
	bashrc := filepath.Join(home, ".bashrc")
	testutil.WriteTestFile(t, bashrc, []byte(testContent), utils.PermFileDefault)

	pm := NewProfileManager(ShellTypeBash)
	profile, err := pm.GetProfile()
	require.NoError(t, err, "GetProfile() error =")

	assert.True(t, profile.Exists, "Expected profile to exist")

	assert.True(t, profile.HasGoenv, "Expected profile to have goenv")

	assert.Equal(t, testContent, profile.Content, "Content =")
}

func TestDetectPathResets(t *testing.T) {
	tests := []struct {
		name      string
		content   string
		wantIssue bool
	}{
		{
			name:      "no PATH modification",
			content:   "eval \"$(goenv init -)\"",
			wantIssue: false,
		},
		{
			name:      "PATH append (safe)",
			content:   "export PATH=/usr/bin:$PATH\neval \"$(goenv init -)\"",
			wantIssue: false,
		},
		{
			name:      "PATH reset (unsafe)",
			content:   "export PATH=/usr/bin\neval \"$(goenv init -)\"",
			wantIssue: true,
		},
		{
			name:      "PATH reset without $PATH",
			content:   "PATH=\"/usr/local/bin:/usr/bin\"",
			wantIssue: true,
		},
	}

	pm := NewProfileManager(ShellTypeBash)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			profile := &Profile{
				Path:     "/test/.bashrc",
				Content:  tt.content,
				Exists:   true,
				HasGoenv: true,
			}

			issues := pm.detectPathResets([]*Profile{profile})

			assert.False(t, tt.wantIssue && len(issues) == 0, "Expected to detect PATH reset issue, but none found")

			assert.False(t, !tt.wantIssue && len(issues) > 0, "Expected no issues, but found")
		})
	}
}

func TestGetAllProfiles(t *testing.T) {
	var err error
	// This test is Unix-specific (tests .zshrc, .zprofile)
	if utils.IsWindows() {
		t.Skip("Skipping Unix shell profile test on Windows")
	}

	// Create temporary test directory
	tmpDir := t.TempDir()
	home := filepath.Join(tmpDir, "home")
	err = utils.EnsureDirWithContext(home, "create test directory")
	require.NoError(t, err)

	// Override home directory for testing
	oldHome := os.Getenv(utils.EnvVarHome)
	os.Setenv(utils.EnvVarHome, home)
	defer os.Setenv(utils.EnvVarHome, oldHome)

	// Create multiple zsh profiles
	zshrc := filepath.Join(home, ".zshrc")
	zprofile := filepath.Join(home, ".zprofile")

	testutil.WriteTestFile(t, zshrc, []byte("# zshrc\n"), utils.PermFileDefault)
	testutil.WriteTestFile(t, zprofile, []byte("# zprofile\n"), utils.PermFileDefault)

	pm := NewProfileManager(ShellTypeZsh)
	profiles, err := pm.GetAllProfiles()
	require.NoError(t, err, "GetAllProfiles() error =")

	// Should return 3 profiles: .zshrc, .zprofile, .zshenv
	assert.Len(t, profiles, 3, "Expected 3 profiles")

	// Check that at least 2 exist (the ones we created)
	existCount := 0
	for _, p := range profiles {
		if p.Exists {
			existCount++
		}
	}

	assert.Equal(t, 2, existCount, "Expected 2 existing profiles")
}

func TestAddInitialization(t *testing.T) {
	var err error
	// Create temporary test directory
	tmpDir := t.TempDir()
	home := filepath.Join(tmpDir, "home")
	err = utils.EnsureDirWithContext(home, "create test directory")
	require.NoError(t, err)

	// Override home directory for testing
	oldHome := os.Getenv(utils.EnvVarHome)
	os.Setenv(utils.EnvVarHome, home)
	defer os.Setenv(utils.EnvVarHome, oldHome)

	pm := NewProfileManager(ShellTypeBash)

	// Add initialization to non-existent profile
	err = pm.AddInitialization(false)
	require.NoError(t, err, "AddInitialization() error =")

	// Verify it was added
	profile, err := pm.GetProfile()
	require.NoError(t, err, "GetProfile() error =")

	assert.True(t, profile.Exists, "Expected profile to be created")

	assert.True(t, profile.HasGoenv, "Expected profile to have goenv")

	// Try to add again - should fail
	err = pm.AddInitialization(false)
	assert.Error(t, err, "Expected error when adding initialization twice, got nil")
}

func TestDetectDuplicates(t *testing.T) {
	content := `
export PATH=$HOME/bin:$PATH
eval "$(goenv init -)"
# some other content
eval "$(goenv init -)"
`

	profile := &Profile{
		Path:     "/test/.bashrc",
		Content:  content,
		Exists:   true,
		HasGoenv: true,
	}

	pm := NewProfileManager(ShellTypeBash)
	issues := pm.detectDuplicates([]*Profile{profile})

	assert.NotEqual(t, 0, len(issues), "Expected to detect duplicate goenv init, but none found")

	assert.Equal(t, IssueTypeDuplicate, issues[0].Type, "Expected IssueTypeDuplicate")
}
