package tools

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestExtractToolName(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "golang.org/x/tools/cmd/goimports@latest",
			expected: "goimports",
		},
		{
			input:    "github.com/go-delve/delve/cmd/dlv@v1.20.1",
			expected: "dlv",
		},
		{
			input:    "golang.org/x/tools/cmd/goimports",
			expected: "goimports",
		},
		{
			input:    "goimports@latest",
			expected: "goimports",
		},
		{
			input:    "goimports",
			expected: "goimports",
		},
		{
			input:    "",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := ExtractToolName(tt.input)
			assert.Equal(t, tt.expected, result, "ExtractToolName() = %v", tt.input)
		})
	}
}

func TestExtractToolNames(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name: "multiple packages",
			input: []string{
				"golang.org/x/tools/cmd/goimports@latest",
				"github.com/go-delve/delve/cmd/dlv@v1.20.1",
				"golang.org/x/tools/gopls@latest",
			},
			expected: []string{"goimports", "dlv", "gopls"},
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name:     "single package",
			input:    []string{"goimports@latest"},
			expected: []string{"goimports"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractToolNames(tt.input)
			assert.Equal(t, tt.expected, result, "ExtractToolNames() = %v %v %v", tt.input, result, tt.expected)
		})
	}
}

func TestNormalizePackagePath(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{
			input:    "golang.org/x/tools/cmd/goimports",
			expected: "golang.org/x/tools/cmd/goimports@latest",
		},
		{
			input:    "golang.org/x/tools/cmd/goimports@latest",
			expected: "golang.org/x/tools/cmd/goimports@latest",
		},
		{
			input:    "golang.org/x/tools/cmd/goimports@v0.1.0",
			expected: "golang.org/x/tools/cmd/goimports@v0.1.0",
		},
		{
			input:    "goimports",
			expected: "golang.org/x/tools/cmd/goimports@latest",
		},
		{
			input:    "",
			expected: "@latest",
		},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			result := NormalizePackagePath(tt.input)
			assert.Equal(t, tt.expected, result, "NormalizePackagePath() = %v", tt.input)
		})
	}
}

func TestNormalizePackagePaths(t *testing.T) {
	tests := []struct {
		name     string
		input    []string
		expected []string
	}{
		{
			name: "mixed paths",
			input: []string{
				"golang.org/x/tools/cmd/goimports",
				"golang.org/x/tools/gopls@latest",
				"github.com/go-delve/delve/cmd/dlv@v1.20.1",
			},
			expected: []string{
				"golang.org/x/tools/cmd/goimports@latest",
				"golang.org/x/tools/gopls@latest",
				"github.com/go-delve/delve/cmd/dlv@v1.20.1",
			},
		},
		{
			name:     "empty slice",
			input:    []string{},
			expected: []string{},
		},
		{
			name: "all already normalized",
			input: []string{
				"goimports@latest",
				"gopls@v0.12.0",
			},
			expected: []string{
				"golang.org/x/tools/cmd/goimports@latest",
				"golang.org/x/tools/gopls@v0.12.0",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := NormalizePackagePaths(tt.input)
			assert.Equal(t, tt.expected, result, "NormalizePackagePaths() = %v %v %v", tt.input, result, tt.expected)
		})
	}
}
