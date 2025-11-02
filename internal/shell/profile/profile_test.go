package profile

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/go-nv/goenv/internal/utils"
	"github.com/go-nv/goenv/testing/testutil"
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
			if got != tt.expected {
				t.Errorf("GetInitLine() = %q, want %q", got, tt.expected)
			}
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
			if got != tt.expected {
				t.Errorf("hasGoenvInit() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetProfile(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	home := filepath.Join(tmpDir, "home")
	if err := utils.EnsureDirWithContext(home, "create test directory"); err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}

	if !profile.Exists {
		t.Error("Expected profile to exist")
	}

	if !profile.HasGoenv {
		t.Error("Expected profile to have goenv")
	}

	if profile.Content != testContent {
		t.Errorf("Content = %q, want %q", profile.Content, testContent)
	}
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

			if tt.wantIssue && len(issues) == 0 {
				t.Error("Expected to detect PATH reset issue, but none found")
			}

			if !tt.wantIssue && len(issues) > 0 {
				t.Errorf("Expected no issues, but found: %v", issues)
			}
		})
	}
}

func TestGetAllProfiles(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	home := filepath.Join(tmpDir, "home")
	if err := utils.EnsureDirWithContext(home, "create test directory"); err != nil {
		t.Fatal(err)
	}

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
	if err != nil {
		t.Fatalf("GetAllProfiles() error = %v", err)
	}

	// Should return 3 profiles: .zshrc, .zprofile, .zshenv
	if len(profiles) != 3 {
		t.Errorf("Expected 3 profiles, got %d", len(profiles))
	}

	// Check that at least 2 exist (the ones we created)
	existCount := 0
	for _, p := range profiles {
		if p.Exists {
			existCount++
		}
	}

	if existCount != 2 {
		t.Errorf("Expected 2 existing profiles, got %d", existCount)
	}
}

func TestAddInitialization(t *testing.T) {
	// Create temporary test directory
	tmpDir := t.TempDir()
	home := filepath.Join(tmpDir, "home")
	if err := utils.EnsureDirWithContext(home, "create test directory"); err != nil {
		t.Fatal(err)
	}

	// Override home directory for testing
	oldHome := os.Getenv(utils.EnvVarHome)
	os.Setenv(utils.EnvVarHome, home)
	defer os.Setenv(utils.EnvVarHome, oldHome)

	pm := NewProfileManager(ShellTypeBash)

	// Add initialization to non-existent profile
	err := pm.AddInitialization(false)
	if err != nil {
		t.Fatalf("AddInitialization() error = %v", err)
	}

	// Verify it was added
	profile, err := pm.GetProfile()
	if err != nil {
		t.Fatalf("GetProfile() error = %v", err)
	}

	if !profile.Exists {
		t.Error("Expected profile to be created")
	}

	if !profile.HasGoenv {
		t.Error("Expected profile to have goenv")
	}

	// Try to add again - should fail
	err = pm.AddInitialization(false)
	if err == nil {
		t.Error("Expected error when adding initialization twice, got nil")
	}
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

	if len(issues) == 0 {
		t.Error("Expected to detect duplicate goenv init, but none found")
	}

	if issues[0].Type != IssueTypeDuplicate {
		t.Errorf("Expected IssueTypeDuplicate, got %v", issues[0].Type)
	}
}
