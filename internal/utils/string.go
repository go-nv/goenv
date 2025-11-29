package utils

import (
	"strings"
)

// SplitLines splits a string into non-empty, trimmed lines.
// This is a common pattern in the codebase for processing command output
// and file contents.
//
// Example:
//
//	lines := SplitLines("line1\nline2\n  \nline3  ")
//	// Returns: ["line1", "line2", "line3"]
func SplitLines(s string) []string {
	if s == "" {
		return nil
	}

	lines := strings.Split(s, "\n")
	result := make([]string, 0, len(lines))

	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line != "" {
			result = append(result, line)
		}
	}

	return result
}

// SplitFields splits a string into fields using strings.Fields after trimming.
// This is more robust than strings.Fields alone as it handles leading/trailing
// whitespace consistently.
//
// Example:
//
//	fields := SplitFields("  field1   field2  field3  ")
//	// Returns: ["field1", "field2", "field3"]
func SplitFields(s string) []string {
	return strings.Fields(strings.TrimSpace(s))
}

// GetField returns the field at the given index from a whitespace-split string.
// Returns the field and true if the index is valid, or empty string and false otherwise.
//
// Example:
//
//	field, ok := GetField("one two three", 1)
//	// Returns: "two", true
//
//	field, ok := GetField("one two", 5)
//	// Returns: "", false
func GetField(s string, index int) (string, bool) {
	fields := SplitFields(s)
	if index < 0 || index >= len(fields) {
		return "", false
	}
	return fields[index], true
}

// ProcessLines calls a processor function for each non-empty, trimmed line in
// the input string. This is useful for processing multi-line content like
// command output or file contents.
//
// Example:
//
//	err := ProcessLines(content, func(line string) error {
//	    if strings.HasPrefix(line, "#") {
//	        return nil // skip comments
//	    }
//	    return processLine(line)
//	})
func ProcessLines(content string, processor func(line string) error) error {
	lines := SplitLines(content)
	for _, line := range lines {
		if err := processor(line); err != nil {
			return err
		}
	}
	return nil
}

// TrimSuffix removes a suffix from a string if present.
// Unlike strings.TrimSuffix, this is case-insensitive and trims whitespace.
func TrimSuffixIgnoreCase(s, suffix string) string {
	s = strings.TrimSpace(s)
	lowerS := strings.ToLower(s)
	lowerSuffix := strings.ToLower(suffix)

	if strings.HasSuffix(lowerS, lowerSuffix) {
		return s[:len(s)-len(suffix)]
	}
	return s
}
