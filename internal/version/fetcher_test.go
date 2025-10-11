package version

import (
	"testing"
)

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		v1, v2   string
		expected int
	}{
		{"go1.21.0", "go1.20.0", 1},
		{"go1.20.0", "go1.21.0", -1},
		{"go1.21.0", "go1.21.0", 0},
		{"1.21.0", "1.20.0", 1},
		{"1.21.5", "1.21.0", 1},
		{"1.21.0", "1.21.5", -1},
		{"1.21.0", "1.21.0rc1", 1},      // stable > rc
		{"1.21.0rc1", "1.21.0beta1", 1}, // rc > beta
	}

	for _, test := range tests {
		result := compareVersions(test.v1, test.v2)
		if result != test.expected {
			t.Errorf("compareVersions(%s, %s) = %d, expected %d",
				test.v1, test.v2, result, test.expected)
		}
	}
}

func TestSortVersions(t *testing.T) {
	releases := []GoRelease{
		{Version: "go1.20.0", Stable: true},
		{Version: "go1.21.0", Stable: true},
		{Version: "go1.19.0", Stable: true},
		{Version: "go1.21.5", Stable: true},
	}

	SortVersions(releases)

	expected := []string{"go1.21.5", "go1.21.0", "go1.20.0", "go1.19.0"}
	for i, release := range releases {
		if release.Version != expected[i] {
			t.Errorf("After sorting, position %d: got %s, expected %s",
				i, release.Version, expected[i])
		}
	}
}

func TestGetFileForPlatform(t *testing.T) {
	release := GoRelease{
		Version: "go1.21.0",
		Files: []GoFile{
			{Filename: "go1.21.0.linux-amd64.tar.gz", OS: "linux", Arch: "amd64", Kind: "archive"},
			{Filename: "go1.21.0.darwin-arm64.tar.gz", OS: "darwin", Arch: "arm64", Kind: "archive"},
			{Filename: "go1.21.0.windows-amd64.zip", OS: "windows", Arch: "amd64", Kind: "archive"},
		},
	}

	file, err := release.GetFileForPlatform("linux", "amd64")
	if err != nil {
		t.Fatalf("Expected to find file for linux/amd64, got error: %v", err)
	}

	if file.Filename != "go1.21.0.linux-amd64.tar.gz" {
		t.Errorf("Expected linux-amd64 file, got %s", file.Filename)
	}

	// Test non-existent platform
	_, err = release.GetFileForPlatform("nonexistent", "arch")
	if err == nil {
		t.Error("Expected error for non-existent platform, got nil")
	}
}
