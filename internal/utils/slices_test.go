package utils

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSlicesEqual(t *testing.T) {
	tests := []struct {
		name     string
		a        []string
		b        []string
		expected bool
	}{
		{
			name:     "equal slices same order",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "c"},
			expected: true,
		},
		{
			name:     "equal slices different order",
			a:        []string{"a", "b", "c"},
			b:        []string{"c", "a", "b"},
			expected: true,
		},
		{
			name:     "different slices",
			a:        []string{"a", "b", "c"},
			b:        []string{"a", "b", "d"},
			expected: false,
		},
		{
			name:     "different lengths",
			a:        []string{"a", "b"},
			b:        []string{"a", "b", "c"},
			expected: false,
		},
		{
			name:     "empty slices",
			a:        []string{},
			b:        []string{},
			expected: true,
		},
		{
			name:     "duplicates handled correctly",
			a:        []string{"a", "a", "b"},
			b:        []string{"a", "b", "b"},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SlicesEqual(tt.a, tt.b)
			assert.Equal(t, tt.expected, result, "SlicesEqual(, ) = %v %v", tt.a, tt.b)
		})
	}
}

func TestSliceContains(t *testing.T) {
	tests := []struct {
		name     string
		slice    []string
		item     string
		expected bool
	}{
		{
			name:     "contains item",
			slice:    []string{"a", "b", "c"},
			item:     "b",
			expected: true,
		},
		{
			name:     "does not contain item",
			slice:    []string{"a", "b", "c"},
			item:     "d",
			expected: false,
		},
		{
			name:     "empty slice",
			slice:    []string{},
			item:     "a",
			expected: false,
		},
		{
			name:     "contains at start",
			slice:    []string{"a", "b", "c"},
			item:     "a",
			expected: true,
		},
		{
			name:     "contains at end",
			slice:    []string{"a", "b", "c"},
			item:     "c",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := SliceContains(tt.slice, tt.item)
			assert.Equal(t, tt.expected, result, "SliceContains(, ) = %v %v", tt.slice, tt.item)
		})
	}
}
