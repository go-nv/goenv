package cache

import (
	"testing"
	"time"
)

func TestParseByteSize(t *testing.T) {
	tests := []struct {
		input    string
		expected int64
		wantErr  bool
	}{
		{"1GB", 1024 * 1024 * 1024, false},
		{"500MB", 500 * 1024 * 1024, false},
		{"1.5GB", int64(1.5 * 1024 * 1024 * 1024), false},
		{"2TB", 2 * 1024 * 1024 * 1024 * 1024, false},
		{"100KB", 100 * 1024, false},
		{"1K", 1024, false},
		{"1M", 1024 * 1024, false},
		{"1G", 1024 * 1024 * 1024, false},
		{"100B", 100, false},
		{"100", 100, false}, // No unit defaults to bytes
		{"", 0, true},       // Empty should error
		{"invalid", 0, true},
		{"100XB", 0, true}, // Invalid unit
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseByteSize(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseByteSize(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseByteSize(%q) unexpected error: %v", tt.input, err)
				return
			}
			if result != tt.expected {
				t.Errorf("ParseByteSize(%q) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseDuration(t *testing.T) {
	tests := []struct {
		input    string
		expected time.Duration
		wantErr  bool
	}{
		{"30d", 30 * 24 * time.Hour, false},
		{"1w", 7 * 24 * time.Hour, false},
		{"24h", 24 * time.Hour, false},
		{"1.5d", time.Duration(1.5 * float64(24*time.Hour)), false},
		{"2weeks", 14 * 24 * time.Hour, false},
		{"3days", 3 * 24 * time.Hour, false},
		{"60m", 60 * time.Minute, false},
		{"3600s", 3600 * time.Second, false},
		{"", 0, true}, // Empty should error
		{"invalid", 0, true},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result, err := ParseDuration(tt.input)
			if tt.wantErr {
				if err == nil {
					t.Errorf("ParseDuration(%q) expected error, got nil", tt.input)
				}
				return
			}
			if err != nil {
				t.Errorf("ParseDuration(%q) unexpected error: %v", tt.input, err)
				return
			}
			if result != tt.expected {
				t.Errorf("ParseDuration(%q) = %v, want %v", tt.input, result, tt.expected)
			}
		})
	}
}

func TestParseABIFromCacheName(t *testing.T) {
	tests := []struct {
		name           string
		cacheName      string
		expectedGOOS   string
		expectedGOARCH string
		expectedABI    map[string]string
	}{
		{
			name:           "old format",
			cacheName:      "go-build",
			expectedGOOS:   "",
			expectedGOARCH: "",
			expectedABI:    nil,
		},
		{
			name:           "basic darwin arm64",
			cacheName:      "go-build-darwin-arm64",
			expectedGOOS:   "darwin",
			expectedGOARCH: "arm64",
			expectedABI:    nil,
		},
		{
			name:           "linux amd64 with v3",
			cacheName:      "go-build-linux-amd64-v3",
			expectedGOOS:   "linux",
			expectedGOARCH: "amd64",
			expectedABI:    map[string]string{"GOAMD64": "v3"},
		},
		{
			name:           "linux arm with v7",
			cacheName:      "go-build-linux-arm-v7",
			expectedGOOS:   "linux",
			expectedGOARCH: "arm",
			expectedABI:    map[string]string{"GOARM": "7"},
		},
		{
			name:           "windows 386 sse2",
			cacheName:      "go-build-windows-386-sse2",
			expectedGOOS:   "windows",
			expectedGOARCH: "386",
			expectedABI:    map[string]string{"GO386": "sse2"},
		},
		{
			name:           "with CGO hash",
			cacheName:      "go-build-linux-amd64-cgo-abc123",
			expectedGOOS:   "linux",
			expectedGOARCH: "amd64",
			expectedABI:    map[string]string{"CGO_HASH": "abc123"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			goos, goarch, abi := ParseABIFromCacheName(tt.cacheName)

			if goos != tt.expectedGOOS {
				t.Errorf("GOOS = %q, want %q", goos, tt.expectedGOOS)
			}
			if goarch != tt.expectedGOARCH {
				t.Errorf("GOARCH = %q, want %q", goarch, tt.expectedGOARCH)
			}

			if tt.expectedABI == nil {
				if abi != nil {
					t.Errorf("ABI = %v, want nil", abi)
				}
			} else {
				if abi == nil {
					t.Errorf("ABI = nil, want %v", tt.expectedABI)
					return
				}
				for k, v := range tt.expectedABI {
					if abi[k] != v {
						t.Errorf("ABI[%q] = %q, want %q", k, abi[k], v)
					}
				}
			}
		})
	}
}

func TestFormatNumber(t *testing.T) {
	tests := []struct {
		input    int
		expected string
	}{
		{0, "0"},
		{123, "123"},
		{1234, "1,234"},
		{1234567, "1,234,567"},
		{1234567890, "1,234,567,890"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatNumber(tt.input)
			if result != tt.expected {
				t.Errorf("FormatNumber(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}

func TestFormatFileCount(t *testing.T) {
	tests := []struct {
		n           int
		approximate bool
		expected    string
	}{
		{1234, false, "1,234"},
		{1234, true, "~1,234"},
		{-1, false, "~"},
		{-1, true, "~"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatFileCount(tt.n, tt.approximate)
			if result != tt.expected {
				t.Errorf("FormatFileCount(%d, %v) = %q, want %q", tt.n, tt.approximate, result, tt.expected)
			}
		})
	}
}

func TestFormatBytes(t *testing.T) {
	tests := []struct {
		input    int64
		expected string
	}{
		{100, "100 B"},
		{1024, "1.0 KB"},
		{1536, "1.5 KB"},
		{1048576, "1.0 MB"},
		{1073741824, "1.0 GB"},
		{1099511627776, "1.0 TB"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := FormatBytes(tt.input)
			if result != tt.expected {
				t.Errorf("FormatBytes(%d) = %q, want %q", tt.input, result, tt.expected)
			}
		})
	}
}
