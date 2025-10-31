package hooks

import (
	"errors"
	"fmt"
)

// CheckDiskSpaceAction validates that sufficient disk space is available
type CheckDiskSpaceAction struct{}

func (a *CheckDiskSpaceAction) Name() ActionName {
	return ActionCheckDiskSpace
}

func (a *CheckDiskSpaceAction) Description() string {
	return "Validates that sufficient disk space is available before operations"
}

func (a *CheckDiskSpaceAction) Validate(params map[string]interface{}) error {
	// Validate path parameter (required)
	path, ok := params["path"].(string)
	if !ok || path == "" {
		return fmt.Errorf("path parameter is required")
	}
	if err := ValidatePath(path); err != nil {
		return fmt.Errorf("invalid path: %w", err)
	}

	// Validate min_free_mb parameter (required)
	minFreeMB, ok := params["min_free_mb"].(int)
	if !ok {
		// Try float conversion for YAML number parsing
		if minFreeFloat, ok := params["min_free_mb"].(float64); ok {
			minFreeMB = int(minFreeFloat)
		} else {
			return fmt.Errorf("min_free_mb parameter is required and must be a number")
		}
	}
	if minFreeMB < 0 {
		return fmt.Errorf("min_free_mb must be non-negative")
	}

	// Validate on_insufficient parameter (optional)
	if action, ok := params["on_insufficient"].(string); ok {
		if action != "warn" && action != "error" {
			return fmt.Errorf("invalid on_insufficient: %s (must be warn or error)", action)
		}
	}

	return nil
}

func (a *CheckDiskSpaceAction) Execute(ctx *HookContext, params map[string]interface{}) error {
	// Get path and expand variables
	path := params["path"].(string)
	path = interpolateString(path, ctx.Variables)
	path = expandTilde(path)

	// Get min_free_mb
	minFreeMB := 0
	if mb, ok := params["min_free_mb"].(int); ok {
		minFreeMB = mb
	} else if mbFloat, ok := params["min_free_mb"].(float64); ok {
		minFreeMB = int(mbFloat)
	}

	// Get on_insufficient (default to error)
	action := "error"
	if a, ok := params["on_insufficient"].(string); ok {
		action = a
	}

	// Check disk space
	freeMB, totalMB, err := getDiskSpace(path)
	if err != nil {
		return fmt.Errorf("failed to check disk space: %w", err)
	}

	// Check if we have enough space
	if freeMB < int64(minFreeMB) {
		msg := fmt.Sprintf("insufficient disk space: %d MB free (need %d MB) out of %d MB total at %s",
			freeMB, minFreeMB, totalMB, path)

		if action == "error" {
			return errors.New(msg)
		}
		// For "warn", we just log but don't fail
		fmt.Printf("Warning: %s\n", msg)
	}

	return nil
}
