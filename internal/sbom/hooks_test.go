package sbom

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestNewHookManager(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "valid git repository",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				// Initialize git repo
				cmd := exec.Command("git", "init")
				cmd.Dir = dir
				if err := cmd.Run(); err != nil {
					t.Fatalf("failed to init git repo: %v", err)
				}
				return dir
			},
			wantErr: false,
		},
		{
			name: "not a git repository",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := tt.setup(t)

			// Use mock goenv path for testing
			manager, err := NewHookManagerWithGoenv(dir, "/usr/bin/goenv")
			if (err != nil) != tt.wantErr {
				t.Errorf("NewHookManager() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if manager == nil {
					t.Error("NewHookManager() returned nil manager")
					return
				}
				if manager.RepoRoot == "" {
					t.Error("RepoRoot is empty")
				}
				if manager.HooksDir == "" {
					t.Error("HooksDir is empty")
				}
				if manager.GoenvPath == "" {
					t.Error("GoenvPath is empty")
				}
			}
		})
	}
}

func TestHookManager_InstallHook(t *testing.T) {
	// Create test git repo
	repoDir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	manager, err := NewHookManagerWithGoenv(repoDir, "/usr/bin/goenv")
	if err != nil {
		t.Fatalf("NewHookManager() error = %v", err)
	}

	tests := []struct {
		name       string
		config     HookConfig
		preInstall func() // Install something before test
		wantErr    bool
	}{
		{
			name:    "install with default config",
			config:  DefaultHookConfig(),
			wantErr: false,
		},
		{
			name: "install with custom config",
			config: HookConfig{
				AutoGenerate: true,
				FailOnError:  false,
				OutputPath:   "custom-sbom.json",
				Format:       "spdx",
				Quiet:        true,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Clean up any existing hook
			hookPath := filepath.Join(manager.HooksDir, "pre-commit")
			os.Remove(hookPath)

			if tt.preInstall != nil {
				tt.preInstall()
			}

			err := manager.InstallHook(tt.config)
			if (err != nil) != tt.wantErr {
				t.Errorf("InstallHook() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				// Verify hook was created
				if _, err := os.Stat(hookPath); err != nil {
					t.Errorf("hook file not created: %v", err)
					return
				}

				// Verify hook is executable (Unix only - Windows uses extensions)
				if runtime.GOOS != "windows" {
					info, err := os.Stat(hookPath)
					if err != nil {
						t.Errorf("failed to stat hook: %v", err)
						return
					}
					if info.Mode()&0111 == 0 {
						t.Error("hook is not executable")
					}
				}

				// Verify hook content
				content, err := os.ReadFile(hookPath)
				if err != nil {
					t.Errorf("failed to read hook: %v", err)
					return
				}

				hookContent := string(content)

				// Check for marker
				if !strings.Contains(hookContent, "# goenv-sbom-hook") {
					t.Error("hook missing goenv marker")
				}

				// Check for shebang
				if !strings.HasPrefix(hookContent, "#!/bin/sh") {
					t.Error("hook missing shebang")
				}

				// Check for go.mod/go.sum detection
				if !strings.Contains(hookContent, "go.mod") {
					t.Error("hook missing go.mod detection")
				}

				// Check config-specific content
				if tt.config.Quiet {
					if !strings.Contains(hookContent, "GOENV_QUIET=1") {
						t.Error("quiet mode not configured")
					}
				}

				if strings.Contains(hookContent, tt.config.OutputPath) == false {
					t.Errorf("output path %q not found in hook", tt.config.OutputPath)
				}

				if strings.Contains(hookContent, tt.config.Format) == false {
					t.Errorf("format %q not found in hook", tt.config.Format)
				}
			}
		})
	}
}

func TestHookManager_UninstallHook(t *testing.T) {
	// Create test git repo
	repoDir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	manager, err := NewHookManagerWithGoenv(repoDir, "/usr/bin/goenv")
	if err != nil {
		t.Fatalf("NewHookManager() error = %v", err)
	}

	tests := []struct {
		name    string
		setup   func()
		wantErr bool
	}{
		{
			name: "uninstall existing goenv hook",
			setup: func() {
				config := DefaultHookConfig()
				if err := manager.InstallHook(config); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			wantErr: false,
		},
		{
			name: "uninstall when no hook exists",
			setup: func() {
				hookPath := filepath.Join(manager.HooksDir, "pre-commit")
				os.Remove(hookPath)
			},
			wantErr: true,
		},
		{
			name: "uninstall non-goenv hook",
			setup: func() {
				hookPath := filepath.Join(manager.HooksDir, "pre-commit")
				os.WriteFile(hookPath, []byte("#!/bin/sh\necho 'custom hook'\n"), 0755)
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			err := manager.UninstallHook()
			if (err != nil) != tt.wantErr {
				t.Errorf("UninstallHook() error = %v, wantErr %v", err, tt.wantErr)
			}

			if !tt.wantErr {
				// Verify hook was removed
				hookPath := filepath.Join(manager.HooksDir, "pre-commit")
				if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
					t.Error("hook file still exists after uninstall")
				}
			}
		})
	}
}

func TestHookManager_IsHookInstalled(t *testing.T) {
	// Create test git repo
	repoDir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	manager, err := NewHookManagerWithGoenv(repoDir, "/usr/bin/goenv")
	if err != nil {
		t.Fatalf("NewHookManager() error = %v", err)
	}

	tests := []struct {
		name    string
		setup   func()
		want    bool
		wantErr bool
	}{
		{
			name: "no hook installed",
			setup: func() {
				hookPath := filepath.Join(manager.HooksDir, "pre-commit")
				os.Remove(hookPath)
			},
			want:    false,
			wantErr: false,
		},
		{
			name: "goenv hook installed",
			setup: func() {
				config := DefaultHookConfig()
				if err := manager.InstallHook(config); err != nil {
					t.Fatalf("setup failed: %v", err)
				}
			},
			want:    true,
			wantErr: false,
		},
		{
			name: "non-goenv hook installed",
			setup: func() {
				hookPath := filepath.Join(manager.HooksDir, "pre-commit")
				os.WriteFile(hookPath, []byte("#!/bin/sh\necho 'custom'\n"), 0755)
			},
			want:    false,
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tt.setup()

			got, err := manager.IsHookInstalled()
			if (err != nil) != tt.wantErr {
				t.Errorf("IsHookInstalled() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			if got != tt.want {
				t.Errorf("IsHookInstalled() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestHookManager_GetHookStatus(t *testing.T) {
	// Create test git repo
	repoDir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	manager, err := NewHookManagerWithGoenv(repoDir, "/usr/bin/goenv")
	if err != nil {
		t.Fatalf("NewHookManager() error = %v", err)
	}

	// Install hook
	config := DefaultHookConfig()
	if err := manager.InstallHook(config); err != nil {
		t.Fatalf("InstallHook() error = %v", err)
	}

	status, err := manager.GetHookStatus()
	if err != nil {
		t.Fatalf("GetHookStatus() error = %v", err)
	}

	// Verify status fields
	if status["installed"] != true {
		t.Error("status shows not installed")
	}

	if status["repo_root"] == "" {
		t.Error("repo_root is empty")
	}

	if status["hook_path"] == "" {
		t.Error("hook_path is empty")
	}

	if status["goenv_path"] == "" {
		t.Error("goenv_path is empty")
	}

	if status["hook_content"] == "" {
		t.Error("hook_content is empty")
	}
}

func TestFindGitRoot(t *testing.T) {
	tests := []struct {
		name    string
		setup   func(t *testing.T) string
		wantErr bool
	}{
		{
			name: "find from repo root",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				cmd := exec.Command("git", "init")
				cmd.Dir = dir
				if err := cmd.Run(); err != nil {
					t.Fatalf("failed to init git repo: %v", err)
				}
				return dir
			},
			wantErr: false,
		},
		{
			name: "find from subdirectory",
			setup: func(t *testing.T) string {
				dir := t.TempDir()
				cmd := exec.Command("git", "init")
				cmd.Dir = dir
				if err := cmd.Run(); err != nil {
					t.Fatalf("failed to init git repo: %v", err)
				}
				// Create subdirectory
				subdir := filepath.Join(dir, "sub", "dir")
				os.MkdirAll(subdir, 0755)
				return subdir
			},
			wantErr: false,
		},
		{
			name: "not in git repo",
			setup: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			startPath := tt.setup(t)

			root, err := findGitRoot(startPath)
			if (err != nil) != tt.wantErr {
				t.Errorf("findGitRoot() error = %v, wantErr %v", err, tt.wantErr)
				return
			}

			if !tt.wantErr {
				if root == "" {
					t.Error("findGitRoot() returned empty root")
				}
				// Verify .git directory exists in root
				gitDir := filepath.Join(root, ".git")
				if _, err := os.Stat(gitDir); err != nil {
					t.Errorf(".git directory not found in root: %v", err)
				}
			}
		})
	}
}

func TestDefaultHookConfig(t *testing.T) {
	config := DefaultHookConfig()

	if !config.AutoGenerate {
		t.Error("AutoGenerate should be true by default")
	}

	if !config.FailOnError {
		t.Error("FailOnError should be true by default")
	}

	if config.OutputPath != "sbom.json" {
		t.Errorf("OutputPath = %q, want sbom.json", config.OutputPath)
	}

	if config.Format != "cyclonedx" {
		t.Errorf("Format = %q, want cyclonedx", config.Format)
	}

	if config.Quiet {
		t.Error("Quiet should be false by default")
	}
}

func TestGenerateHookScript(t *testing.T) {
	// Create test git repo
	repoDir := t.TempDir()
	cmd := exec.Command("git", "init")
	cmd.Dir = repoDir
	if err := cmd.Run(); err != nil {
		t.Fatalf("failed to init git repo: %v", err)
	}

	manager, err := NewHookManagerWithGoenv(repoDir, "/usr/bin/goenv")
	if err != nil {
		t.Fatalf("NewHookManager() error = %v", err)
	}

	tests := []struct {
		name           string
		config         HookConfig
		wantContains   []string
		wantNotContain []string
	}{
		{
			name:   "default config",
			config: DefaultHookConfig(),
			wantContains: []string{
				"#!/bin/sh",
				"# goenv-sbom-hook",
				"go.mod",
				"go.sum",
				"sbom.json",
				"cyclonedx",
				"exit 1", // fail on error
			},
			wantNotContain: []string{
				"GOENV_QUIET=1",
			},
		},
		{
			name: "quiet mode",
			config: HookConfig{
				AutoGenerate: true,
				FailOnError:  true,
				OutputPath:   "sbom.json",
				Format:       "cyclonedx",
				Quiet:        true,
			},
			wantContains: []string{
				"GOENV_QUIET=1",
				"--quiet",
			},
			wantNotContain: []string{
				"echo \"🔍",
			},
		},
		{
			name: "no fail on error",
			config: HookConfig{
				AutoGenerate: true,
				FailOnError:  false,
				OutputPath:   "sbom.json",
				Format:       "cyclonedx",
				Quiet:        false,
			},
			wantContains: []string{
				"continuing anyway",
			},
			wantNotContain: []string{
				"exit 1",
			},
		},
		{
			name: "custom output and format",
			config: HookConfig{
				AutoGenerate: true,
				FailOnError:  true,
				OutputPath:   "custom.spdx.json",
				Format:       "spdx",
				Quiet:        false,
			},
			wantContains: []string{
				"custom.spdx.json",
				"spdx",
			},
			wantNotContain: nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			script := manager.generateHookScript(tt.config)

			for _, want := range tt.wantContains {
				if !strings.Contains(script, want) {
					t.Errorf("generated script missing %q", want)
				}
			}

			for _, notWant := range tt.wantNotContain {
				if strings.Contains(script, notWant) {
					t.Errorf("generated script should not contain %q", notWant)
				}
			}
		})
	}
}
