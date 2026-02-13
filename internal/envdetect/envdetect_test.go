package envdetect

import (
	"os"
	"testing"

	"github.com/go-nv/goenv/internal/osinfo"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/go-nv/goenv/internal/utils"
)

func TestDetect(t *testing.T) {
	info := Detect()

	require.NotNil(t, info, "Detect() returned nil")

	// Basic sanity checks
	assert.NotEmpty(t, info.Type, "Type should not be empty")

	assert.NotNil(t, info.Warnings, "Warnings should be initialized, not nil")
}

func TestDetectWSL(t *testing.T) {
	if !osinfo.IsLinux() {
		t.Skip("WSL detection only works on Linux")
	}

	version, distro := detectWSL()

	// We can't assume we're running in WSL, so just check the return values are valid
	if version != "" {
		t.Logf("Detected WSL version: %s, distro: %s", version, distro)
		assert.False(t, version != "1" && version != "2", "Invalid WSL version: (expected '1' or '2')")
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
		assert.True(t, validTypes[containerType], "Unknown container type")
	} else {
		t.Log("Not running in container")
	}
}

func TestDetectFilesystem(t *testing.T) {
	tmpDir := t.TempDir()

	info := DetectFilesystem(tmpDir)

	require.NotNil(t, info, "DetectFilesystem() returned nil")

	assert.Equal(t, tmpDir, info.FilesystemPath, "FilesystemPath =")

	// Check that filesystem type was detected
	assert.NotEmpty(t, info.FilesystemType, "FilesystemType should not be empty")

	t.Logf("Detected filesystem type: %s for path: %s", info.FilesystemType, tmpDir)
}

func TestDetectLinuxFilesystem(t *testing.T) {
	if !osinfo.IsLinux() {
		t.Skip("Linux filesystem detection only works on Linux")
	}

	// Test with /tmp which should always exist
	fsType := detectLinuxFilesystem("/tmp")

	assert.NotEmpty(t, fsType, "detectLinuxFilesystem returned empty string")

	t.Logf("Detected filesystem type for /tmp: %s", fsType)
}

func TestDetectDarwinFilesystem(t *testing.T) {
	if !osinfo.IsMacOS() {
		t.Skip("Darwin filesystem detection only works on macOS")
	}

	// Test with /tmp which should always exist
	fsType := detectDarwinFilesystem("/tmp")

	assert.NotEmpty(t, fsType, "detectDarwinFilesystem returned empty string")

	t.Logf("Detected filesystem type for /tmp: %s", fsType)
}

func TestDetectWindowsFilesystem(t *testing.T) {
	if !utils.IsWindows() {
		t.Skip("Windows filesystem detection only works on Windows")
	}

	// Test with temp directory
	tmpDir := os.TempDir()
	fsType := detectWindowsFilesystem(tmpDir)

	assert.NotEmpty(t, fsType, "detectWindowsFilesystem returned empty string")

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
			if tt.name == "WSL mount" && utils.IsWindows() {
				// Check if we're in WSL by looking for /proc/version
				if utils.FileNotExists("/proc/version") {
					t.Skip("Skipping WSL mount test on native Windows")
				}
			}

			result := detectWindowsFilesystem(tt.path)
			assert.Equal(t, tt.expected, result, "detectWindowsFilesystem() = %v", tt.path)
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
			assert.Equal(t, tt.want, got, "String() =")
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
			assert.Equal(t, tt.expected, got, "IsProblematicEnvironment() =")
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
	assert.Len(t, got, len(warnings), "GetWarnings() returned warnings")

	for i, w := range warnings {
		assert.Equal(t, w, got[i], "GetWarnings()[] =")
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
			assert.Equal(t, tt.expectWarnings, hasWarnings, "Expected warnings: , got: (warnings: ) %v", info.Warnings)
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

			assert.Equal(t, tt.expected, containerType, "detectContainer() =")
		})
	}
}
