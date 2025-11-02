package utils

import (
	"encoding/json"
	"fmt"
	"os"
)

// UnmarshalJSONFile reads a JSON file and unmarshals it into the target structure.
// This consolidates the repeated pattern of os.ReadFile + json.Unmarshal (15+ occurrences).
//
// Example:
//
//	var config MyConfig
//	err := UnmarshalJSONFile("/path/to/config.json", &config)
func UnmarshalJSONFile(path string, target interface{}) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	if err := json.Unmarshal(data, target); err != nil {
		return fmt.Errorf("failed to unmarshal JSON: %w", err)
	}

	return nil
}

// MarshalJSONFile marshals a value to JSON and writes it to a file.
// The file is written with PermFileDefault permissions and pretty-printed with indentation.
//
// Example:
//
//	err := MarshalJSONFile("/path/to/config.json", myConfig)
func MarshalJSONFile(path string, value interface{}) error {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := WriteFileWithContext(path, data, PermFileDefault, "write file"); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}

// MarshalJSONCompact marshals a value to compact JSON (no indentation).
// Useful for logging or when space is important.
func MarshalJSONCompact(value interface{}) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// MarshalJSONPretty marshals a value to pretty-printed JSON.
// Useful for human-readable output.
func MarshalJSONPretty(value interface{}) (string, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return "", fmt.Errorf("failed to marshal JSON: %w", err)
	}
	return string(data), nil
}

// MarshalJSONFileCompact marshals a value to compact JSON and writes it to a file.
// The file is written with PermFileDefault permissions without indentation (space-efficient).
// Use this when file size matters or when the JSON doesn't need to be human-readable.
//
// Example:
//
//	err := MarshalJSONFileCompact("/path/to/cache.json", cacheEntry)
func MarshalJSONFileCompact(path string, value interface{}) error {
	data, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("failed to marshal JSON: %w", err)
	}

	if err := WriteFileWithContext(path, data, PermFileDefault, "write file"); err != nil {
		return fmt.Errorf("failed to write file: %w", err)
	}

	return nil
}
