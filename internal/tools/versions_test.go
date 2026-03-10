package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestParseSemver(t *testing.T) {
	tests := []struct {
		name      string
		version   string
		wantMajor int
		wantMinor int
		wantPatch int
		wantErr   bool
	}{
		{
			name:      "standard version with v prefix",
			version:   "v1.2.3",
			wantMajor: 1,
			wantMinor: 2,
			wantPatch: 3,
			wantErr:   false,
		},
		{
			name:      "version without v prefix",
			version:   "1.2.3",
			wantMajor: 1,
			wantMinor: 2,
			wantPatch: 3,
			wantErr:   false,
		},
		{
			name:      "version with pre-release",
			version:   "v1.2.3-rc1",
			wantMajor: 1,
			wantMinor: 2,
			wantPatch: 3,
			wantErr:   false,
		},
		{
			name:      "version with build metadata",
			version:   "v1.2.3+build123",
			wantMajor: 1,
			wantMinor: 2,
			wantPatch: 3,
			wantErr:   false,
		},
		{
			name:      "version without patch",
			version:   "v1.2",
			wantMajor: 1,
			wantMinor: 2,
			wantPatch: 0,
			wantErr:   false,
		},
		{
			name:      "major version zero",
			version:   "v0.1.2",
			wantMajor: 0,
			wantMinor: 1,
			wantPatch: 2,
			wantErr:   false,
		},
		{
			name:    "invalid version",
			version: "invalid",
			wantErr: true,
		},
		{
			name:    "empty version",
			version: "",
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			major, minor, patch, err := ParseSemver(tt.version)
			if tt.wantErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
				assert.Equal(t, tt.wantMajor, major)
				assert.Equal(t, tt.wantMinor, minor)
				assert.Equal(t, tt.wantPatch, patch)
			}
		})
	}
}

func TestCompareVersions(t *testing.T) {
	tests := []struct {
		name string
		v1   string
		v2   string
		want int
	}{
		// Basic comparisons
		{
			name: "equal versions",
			v1:   "v1.2.3",
			v2:   "v1.2.3",
			want: 0,
		},
		{
			name: "v1 greater than v2 (major)",
			v1:   "v2.0.0",
			v2:   "v1.99.99",
			want: 1,
		},
		{
			name: "v1 less than v2 (major)",
			v1:   "v1.0.0",
			v2:   "v2.0.0",
			want: -1,
		},
		{
			name: "v1 greater than v2 (minor)",
			v1:   "v1.10.0",
			v2:   "v1.9.0",
			want: 1,
		},
		{
			name: "v1 less than v2 (minor)",
			v1:   "v1.9.0",
			v2:   "v1.10.0",
			want: -1,
		},
		{
			name: "v1 greater than v2 (patch)",
			v1:   "v1.2.10",
			v2:   "v1.2.9",
			want: 1,
		},
		{
			name: "v1 less than v2 (patch)",
			v1:   "v1.2.9",
			v2:   "v1.2.10",
			want: -1,
		},

		// Edge cases from the bug report
		{
			name: "BUG FIX: 1.10.0 > 1.9.0",
			v1:   "v1.10.0",
			v2:   "v1.9.0",
			want: 1,
		},
		{
			name: "BUG FIX: 0.10.0 > 0.9.0",
			v1:   "v0.10.0",
			v2:   "v0.9.0",
			want: 1,
		},
		{
			name: "BUG FIX: 1.2.10 > 1.2.9",
			v1:   "v1.2.10",
			v2:   "v1.2.9",
			want: 1,
		},

		// Without v prefix
		{
			name: "without v prefix - equal",
			v1:   "1.2.3",
			v2:   "1.2.3",
			want: 0,
		},
		{
			name: "without v prefix - greater",
			v1:   "1.10.0",
			v2:   "1.9.0",
			want: 1,
		},

		// Mixed prefixes
		{
			name: "mixed prefixes v1 > v2",
			v1:   "v1.10.0",
			v2:   "1.9.0",
			want: 1,
		},
		{
			name: "mixed prefixes v1 < v2",
			v1:   "1.9.0",
			v2:   "v1.10.0",
			want: -1,
		},

		// Pre-release versions
		{
			name: "with pre-release suffix",
			v1:   "v1.2.3-rc1",
			v2:   "v1.2.2",
			want: 1,
		},

		// Special cases
		{
			name: "unknown v1",
			v1:   "unknown",
			v2:   "v1.2.3",
			want: -1,
		},
		{
			name: "unknown v2",
			v1:   "v1.2.3",
			v2:   "unknown",
			want: 1,
		},

		// Double-digit components (comprehensive)
		{
			name: "double-digit major",
			v1:   "v12.0.0",
			v2:   "v9.0.0",
			want: 1,
		},
		{
			name: "double-digit minor",
			v1:   "v1.20.0",
			v2:   "v1.9.0",
			want: 1,
		},
		{
			name: "double-digit patch",
			v1:   "v1.2.20",
			v2:   "v1.2.9",
			want: 1,
		},
		{
			name: "all double-digit",
			v1:   "v10.20.30",
			v2:   "v9.19.29",
			want: 1,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := CompareVersions(tt.v1, tt.v2)
			assert.Equal(t, tt.want, got, "CompareVersions(%s, %s)", tt.v1, tt.v2)
		})
	}
}
