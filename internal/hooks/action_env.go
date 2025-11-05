package hooks

import (
	"fmt"
	"os"
	"strings"

	"github.com/go-nv/goenv/internal/utils"
)

// SetEnvAction sets environment variables during hook execution
type SetEnvAction struct{}

func (a *SetEnvAction) Name() ActionName {
	return ActionSetEnv
}

func (a *SetEnvAction) Description() string {
	return "Sets environment variables during hook execution"
}

func (a *SetEnvAction) Validate(params map[string]interface{}) error {
	// Validate variables parameter (required)
	vars, ok := params["variables"].(map[string]interface{})
	if !ok || len(vars) == 0 {
		return fmt.Errorf("variables parameter is required and must be a map")
	}

	// Validate each variable
	for name, value := range vars {
		// Validate variable name
		if err := validateEnvVarName(name); err != nil {
			return fmt.Errorf("invalid variable name %s: %w", name, err)
		}

		// Ensure value is a string
		if _, ok := value.(string); !ok {
			return fmt.Errorf("variable %s value must be a string", name)
		}

		// Validate variable value
		valueStr := value.(string)
		if err := ValidateString(valueStr); err != nil {
			return fmt.Errorf("invalid value for variable %s: %w", name, err)
		}
	}

	// Validate scope parameter (optional)
	if scope, ok := params["scope"].(string); ok {
		scope = strings.ToLower(scope)
		if scope != "hook" && scope != "process" {
			return fmt.Errorf("invalid scope: %s (must be hook or process)", scope)
		}
	}

	return nil
}

func (a *SetEnvAction) Execute(ctx *HookContext, params map[string]interface{}) error {
	// Get variables
	vars := params["variables"].(map[string]interface{})

	// Get scope (default to hook)
	scope := "hook"
	if s, ok := params["scope"].(string); ok {
		scope = strings.ToLower(s)
	}

	// Set each variable
	for name, value := range vars {
		valueStr := value.(string)
		// Interpolate variables in the value
		valueStr = interpolateString(valueStr, ctx.Variables)

		if scope == "process" {
			// Set in the actual process environment
			if err := os.Setenv(name, valueStr); err != nil {
				return fmt.Errorf("failed to set environment variable %s: %w", name, err)
			}
		} else {
			// Set in the hook context only (for subsequent actions)
			ctx.Variables[name] = valueStr
		}
	}

	return nil
}

// validateEnvVarName validates that a variable name follows environment variable naming conventions
func validateEnvVarName(name string) error {
	if name == "" {
		return fmt.Errorf("variable name cannot be empty")
	}

	// Environment variable names should:
	// - Start with a letter or underscore
	// - Contain only letters, digits, and underscores
	// - Not contain special characters or control characters

	// Check first character
	firstChar := rune(name[0])
	if !((firstChar >= 'A' && firstChar <= 'Z') ||
		(firstChar >= 'a' && firstChar <= 'z') ||
		firstChar == '_') {
		return fmt.Errorf("must start with a letter or underscore")
	}

	// Check remaining characters
	for i, char := range name {
		if !((char >= 'A' && char <= 'Z') ||
			(char >= 'a' && char <= 'z') ||
			(char >= '0' && char <= '9') ||
			char == '_') {
			return fmt.Errorf("invalid character at position %d: %c (only letters, digits, and underscores allowed)", i, char)
		}
	}

	// Check for reserved names (optional - can be expanded)
	reserved := []string{
		utils.EnvVarPath,
		utils.EnvVarHome,
		utils.EnvVarUser,
		utils.EnvVarShell,
		utils.EnvVarTerm,
	}
	for _, r := range reserved {
		if strings.EqualFold(name, r) {
			return fmt.Errorf("cannot override reserved variable: %s", r)
		}
	}

	return nil
}
