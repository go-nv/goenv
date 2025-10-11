package build

import (
	"runtime"
	"testing"
)

func TestDetectPlatform(t *testing.T) {
	platform, err := DetectPlatform()
	if err != nil {
		t.Fatalf("DetectPlatform() failed: %v", err)
	}

	if platform.OS == "" {
		t.Error("Platform OS is empty")
	}

	if platform.Arch == "" {
		t.Error("Platform Arch is empty")
	}

	if platform.CPUCores <= 0 {
		t.Errorf("Expected positive CPU cores, got %d", platform.CPUCores)
	}

	// Verify OS matches runtime
	if platform.OS != runtime.GOOS {
		t.Errorf("Expected OS %s, got %s", runtime.GOOS, platform.OS)
	}
}

func TestPlatformIsSupported(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		want     bool
	}{
		{
			name:     "macOS amd64",
			platform: Platform{OS: "darwin", Arch: "amd64"},
			want:     true,
		},
		{
			name:     "macOS arm64",
			platform: Platform{OS: "darwin", Arch: "arm64"},
			want:     true,
		},
		{
			name:     "Linux amd64",
			platform: Platform{OS: "linux", Arch: "amd64"},
			want:     true,
		},
		{
			name:     "Linux arm64",
			platform: Platform{OS: "linux", Arch: "arm64"},
			want:     true,
		},
		{
			name:     "Windows amd64",
			platform: Platform{OS: "windows", Arch: "amd64"},
			want:     true,
		},
		{
			name:     "FreeBSD amd64",
			platform: Platform{OS: "freebsd", Arch: "amd64"},
			want:     true,
		},
		{
			name:     "Unsupported OS",
			platform: Platform{OS: "unsupported", Arch: "amd64"},
			want:     false,
		},
		{
			name:     "Unsupported arch on darwin",
			platform: Platform{OS: "darwin", Arch: "mips"},
			want:     false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.platform.IsSupported()
			if got != tt.want {
				t.Errorf("IsSupported() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestPlatformDownloadFilename(t *testing.T) {
	tests := []struct {
		name     string
		platform Platform
		version  string
		want     string
	}{
		{
			name:     "macOS amd64 with go prefix",
			platform: Platform{OS: "darwin", Arch: "amd64"},
			version:  "go1.21.0",
			want:     "go1.21.0.darwin-amd64.tar.gz",
		},
		{
			name:     "macOS amd64 without go prefix",
			platform: Platform{OS: "darwin", Arch: "amd64"},
			version:  "1.21.0",
			want:     "go1.21.0.darwin-amd64.tar.gz",
		},
		{
			name:     "Linux arm64",
			platform: Platform{OS: "linux", Arch: "arm64"},
			version:  "1.20.5",
			want:     "go1.20.5.linux-arm64.tar.gz",
		},
		{
			name:     "Windows amd64",
			platform: Platform{OS: "windows", Arch: "amd64"},
			version:  "1.21.0",
			want:     "go1.21.0.windows-amd64.zip",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := tt.platform.DownloadFilename(tt.version)
			if got != tt.want {
				t.Errorf("DownloadFilename() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestPlatformDownloadURL(t *testing.T) {
	platform := Platform{OS: "darwin", Arch: "amd64"}

	tests := []struct {
		name    string
		version string
		want    string
	}{
		{
			name:    "standard version",
			version: "1.21.0",
			want:    "https://go.dev/dl/go1.21.0.darwin-amd64.tar.gz",
		},
		{
			name:    "version with go prefix",
			version: "go1.20.5",
			want:    "https://go.dev/dl/go1.20.5.darwin-amd64.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := platform.DownloadURL(tt.version)
			if got != tt.want {
				t.Errorf("DownloadURL() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestPlatformMirrorURL(t *testing.T) {
	platform := Platform{OS: "linux", Arch: "amd64"}

	tests := []struct {
		name       string
		version    string
		mirrorBase string
		want       string
	}{
		{
			name:       "with mirror",
			version:    "1.21.0",
			mirrorBase: "https://mirror.example.com/go",
			want:       "https://mirror.example.com/go/go1.21.0.linux-amd64.tar.gz",
		},
		{
			name:       "mirror with trailing slash",
			version:    "1.21.0",
			mirrorBase: "https://mirror.example.com/go/",
			want:       "https://mirror.example.com/go/go1.21.0.linux-amd64.tar.gz",
		},
		{
			name:       "empty mirror falls back to official",
			version:    "1.21.0",
			mirrorBase: "",
			want:       "https://go.dev/dl/go1.21.0.linux-amd64.tar.gz",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := platform.MirrorURL(tt.version, tt.mirrorBase)
			if got != tt.want {
				t.Errorf("MirrorURL() = %s, want %s", got, tt.want)
			}
		})
	}
}

func TestPlatformString(t *testing.T) {
	platform := Platform{OS: "darwin", Arch: "arm64"}
	want := "darwin/arm64"
	got := platform.String()

	if got != want {
		t.Errorf("String() = %s, want %s", got, want)
	}
}
