package envdetect

import (
	"os"
	"runtime"
	"testing"
)

func TestDetect(t *testing.T) {
	info := Detect()

	if info == nil {
		t.Fatal("Detect() returned nil")
	}

	// Basic sanity checks
	if info.Type == "" {
		t.Error("Type should not be empty")
	}

	if info.Warnings == nil {
		t.Error("Warnings should be initialized, not nil")
	}
}

func TestDetectWSL(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("WSL detection only works on Linux")
	}

	version, distro := detectWSL()

	// We can't assume we're running in WSL, so just check the return values are valid
	if version != "" {
		t.Logf("Detected WSL version: %s, distro: %s", version, distro)
		if version != "1" && version != "2" {
			t.Errorf("Invalid WSL version: %s (expected '1' or '2')", version)
		}
	} else {
		t.Log("Not running in WSL")
	}
}

func TestDetectContainer(t *testing.T) {
	containerType := detectContainer()

	if containerType != "" {
		t.Logf("Detected container type: %s", containerType)
		// Verify it's a known container type
		validTypes := map[string]bool{
			"docker":     true,
			"podman":     true,
			"kubernetes": true,
			"lxc":        true,
			"buildkit":   true,
		}
		if !validTypes[containerType] {
			t.Errorf("Unknown container type: %s", containerType)
		}
	} else {
		t.Log("Not running in container")
	}
}

func TestDetectFilesystem(t *testing.T) {
	tmpDir := t.TempDir()

	info := DetectFilesystem(tmpDir)

	if info == nil {
		t.Fatal("DetectFilesystem() returned nil")
	}

	if info.FilesystemPath != tmpDir {
		t.Errorf("FilesystemPath = %s, want %s", info.FilesystemPath, tmpDir)
	}

	// Check that filesystem type was detected
	if info.FilesystemType == "" {
		t.Error("FilesystemType should not be empty")
	}

	t.Logf("Detected filesystem type: %s for path: %s", info.FilesystemType, tmpDir)
}

func TestDetectLinuxFilesystem(t *testing.T) {
	if runtime.GOOS != "linux" {
		t.Skip("Linux filesystem detection only works on Linux")
	}

	// Test with /tmp which should always exist
	fsType := detectLinuxFilesystem("/tmp")

	if fsType == "" {
		t.Error("detectLinuxFilesystem returned empty string")
	}

	t.Logf("Detected filesystem type for /tmp: %s", fsType)
}

func TestDetectDarwinFilesystem(t *testing.T) {
	if runtime.GOOS != "darwin" {
		t.Skip("Darwin filesystem detection only works on macOS")
	}

	// Test with /tmp which should always exist
	fsType := detectDarwinFilesystem("/tmp")

	if fsType == "" {
		t.Error("detectDarwinFilesystem returned empty string")
	}

	t.Logf("Detected filesystem type for /tmp: %s", fsType)
}

func TestDetectWindowsFilesystem(t *testing.T) {
	if runtime.GOOS != "windows" {
		t.Skip("Windows filesystem detection only works on Windows")
	}

	// Test with temp directory
	tmpDir := os.TempDir()
	fsType := detectWindowsFilesystem(tmpDir)

	if fsType == "" {
		t.Error("detectWindowsFilesystem returned empty string")
	}

	t.Logf("Detected filesystem type for %s: %s", tmpDir, fsType)
}

func TestDetectWindowsFilesystemUNC(t *testing.T) {
	// Test UNC path detection (doesn't require Windows to run)
	tests := []struct {
		name     string
		path     string
		expected FilesystemType
	}{
		{
			name:     "UNC path with backslashes",
			path:     "\\\\server\\share\\path",
			expected: FSTypeSMB,
		},
		{
			name:     "UNC path with forward slashes",
			path:     "//server/share/path",
			expected: FSTypeSMB,
		},
		{
			name:     "WSL mount",
			path:     "/mnt/c/Users/test",
			expected: FSTypeFUSE,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Skip WSL mount test on native Windows (only valid in WSL)
			if tt.name == "WSL mount" && runtime.GOOS == "windows" {
				// Check if we're in WSL by looking for /proc/version
				if _, err := os.Stat("/proc/version"); os.IsNotExist(err) {
					t.Skip("Skipping WSL mount test on native Windows")
				}
			}

			result := detectWindowsFilesystem(tt.path)
			if result != tt.expected {
				t.Errorf("detectWindowsFilesystem(%s) = %s, want %s", tt.path, result, tt.expected)
			}
		})
	}
}

func TestEnvironmentInfoString(t *testing.T) {
	tests := []struct {
		name string
		info *EnvironmentInfo
		want string
	}{
		{
			name: "native environment",
			info: &EnvironmentInfo{
				Type: EnvTypeNative,
			},
			want: "Native",
		},
		{
			name: "docker container",
			info: &EnvironmentInfo{
				Type:          EnvTypeContainer,
				IsContainer:   true,
				ContainerType: "docker",
			},
			want: "Container (docker)",
		},
		{
			name: "WSL 2 with distro",
			info: &EnvironmentInfo{
				Type:       EnvTypeWSL,
				IsWSL:      true,
				WSLVersion: "2",
				WSLDistro:  "Ubuntu",
			},
			want: "WSL 2 (Ubuntu)",
		},
		{
			name: "WSL 1 without distro",
			info: &EnvironmentInfo{
				Type:       EnvTypeWSL,
				IsWSL:      true,
				WSLVersion: "1",
			},
			want: "WSL 1",
		},
		{
			name: "native with NFS filesystem",
			info: &EnvironmentInfo{
				Type:           EnvTypeNative,
				FilesystemType: FSTypeNFS,
			},
			want: "Native, Filesystem: nfs",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.info.String()
			if got != tt.want {
				t.Errorf("String() = %q, want %q", got, tt.want)
			}
		})
	}
}

func TestIsProblematicEnvironment(t *testing.T) {
	tests := []struct {
		name     string
		info     *EnvironmentInfo
		expected bool
	}{
		{
			name: "no warnings",
			info: &EnvironmentInfo{
				Type:     EnvTypeNative,
				Warnings: []string{},
			},
			expected: false,
		},
		{
			name: "with warnings",
			info: &EnvironmentInfo{
				Type:     EnvTypeContainer,
				Warnings: []string{"Test warning"},
			},
			expected: true,
		},
		{
			name: "multiple warnings",
			info: &EnvironmentInfo{
				Type:     EnvTypeWSL,
				Warnings: []string{"Warning 1", "Warning 2"},
			},
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.info.IsProblematicEnvironment()
			if got != tt.expected {
				t.Errorf("IsProblematicEnvironment() = %v, want %v", got, tt.expected)
			}
		})
	}
}

func TestGetWarnings(t *testing.T) {
	warnings := []string{"Warning 1", "Warning 2"}
	info := &EnvironmentInfo{
		Type:     EnvTypeNative,
		Warnings: warnings,
	}

	got := info.GetWarnings()
	if len(got) != len(warnings) {
		t.Errorf("GetWarnings() returned %d warnings, want %d", len(got), len(warnings))
	}

	for i, w := range warnings {
		if got[i] != w {
			t.Errorf("GetWarnings()[%d] = %q, want %q", i, got[i], w)
		}
	}
}

func TestFilesystemTypeWarnings(t *testing.T) {
	tests := []struct {
		name           string
		fsType         FilesystemType
		expectWarnings bool
	}{
		{
			name:           "local filesystem",
			fsType:         FSTypeLocal,
			expectWarnings: false,
		},
		{
			name:           "NFS filesystem",
			fsType:         FSTypeNFS,
			expectWarnings: true,
		},
		{
			name:           "SMB filesystem",
			fsType:         FSTypeSMB,
			expectWarnings: true,
		},
		{
			name:           "bind mount",
			fsType:         FSTypeBind,
			expectWarnings: true,
		},
		{
			name:           "FUSE filesystem",
			fsType:         FSTypeFUSE,
			expectWarnings: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create a basic environment info
			info := &EnvironmentInfo{
				Type:           EnvTypeNative,
				FilesystemType: tt.fsType,
				Warnings:       []string{},
			}

			// Manually add warnings based on filesystem type
			// (This simulates what DetectFilesystem does)
			switch tt.fsType {
			case FSTypeNFS:
				info.Warnings = append(info.Warnings, "NFS filesystem detected")
			case FSTypeSMB:
				info.Warnings = append(info.Warnings, "SMB/CIFS filesystem detected")
			case FSTypeBind:
				info.Warnings = append(info.Warnings, "Bind mount detected")
			case FSTypeFUSE:
				info.Warnings = append(info.Warnings, "FUSE filesystem detected")
			}

			hasWarnings := len(info.Warnings) > 0
			if hasWarnings != tt.expectWarnings {
				t.Errorf("Expected warnings: %v, got: %v (warnings: %v)", tt.expectWarnings, hasWarnings, info.Warnings)
			}
		})
	}
}

// TestContainerEnvVars tests container detection via environment variables
func TestContainerEnvVars(t *testing.T) {
	tests := []struct {
		name     string
		envVar   string
		envValue string
		expected string
	}{
		{
			name:     "kubernetes service host",
			envVar:   "KUBERNETES_SERVICE_HOST",
			envValue: "10.0.0.1",
			expected: "kubernetes",
		},
		{
			name:     "podman container var",
			envVar:   "container",
			envValue: "podman",
			expected: "podman",
		},
		{
			name:     "podman CONTAINER var",
			envVar:   "CONTAINER",
			envValue: "podman",
			expected: "podman",
		},
		{
			name:     "buildkit",
			envVar:   "BUILDKIT_SANDBOX_HOSTNAME",
			envValue: "buildkit",
			expected: "buildkit",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Save original value
			original := os.Getenv(tt.envVar)
			defer func() {
				if original != "" {
					os.Setenv(tt.envVar, original)
				} else {
					os.Unsetenv(tt.envVar)
				}
			}()

			// Set test value
			os.Setenv(tt.envVar, tt.envValue)

			// Detect container
			containerType := detectContainer()

			if containerType != tt.expected {
				t.Errorf("detectContainer() = %q, want %q", containerType, tt.expected)
			}
		})
	}
}
