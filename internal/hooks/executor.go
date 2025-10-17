package hooks

import (
	"context"
	"fmt"
	"regexp"
	"strings"
	"time"
)

// Executor manages hook execution
type Executor struct {
	config   *Config
	registry *Registry
}

// NewExecutor creates a new hook executor
func NewExecutor(config *Config) *Executor {
	return &Executor{
		config:   config,
		registry: DefaultRegistry(),
	}
}

// Execute runs all hooks for the given hook point
func (e *Executor) Execute(hookPoint HookPoint, variables map[string]string) error {
	if !e.config.IsEnabled() {
		return nil // Hooks disabled
	}

	actions := e.config.GetHooks(hookPoint.String())
	if len(actions) == 0 {
		return nil // No hooks configured for this point
	}

	ctx := &HookContext{
		Command:   hookPoint.String(),
		Variables: variables,
		Settings:  e.config.Settings,
		StartTime: time.Now(),
	}

	for i, action := range actions {
		if err := e.executeAction(ctx, action); err != nil {
			if e.config.Settings.ContinueOnError {
				// Log error but continue
				logError(fmt.Sprintf("hook %s[%d] failed: %v", hookPoint.String(), i, err))
				continue
			}
			return fmt.Errorf("hook %s[%d] failed: %w", hookPoint.String(), i, err)
		}
	}

	return nil
}

// executeAction executes a single action
func (e *Executor) executeAction(ctx *HookContext, action Action) error {
	// Get executor for this action type
	executor, exists := e.registry.Get(action.Action)
	if !exists {
		return fmt.Errorf("unknown action: %s", action.Action)
	}

	// Validate parameters
	if err := executor.Validate(action.Params); err != nil {
		return fmt.Errorf("validation failed: %w", err)
	}

	// Create context with timeout
	timeout := e.config.GetTimeout()
	execCtx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	// Execute with timeout
	done := make(chan error, 1)
	go func() {
		done <- executor.Execute(ctx, action.Params)
	}()

	select {
	case err := <-done:
		return err
	case <-execCtx.Done():
		return fmt.Errorf("action timed out after %s", timeout)
	}
}

// TestExecute performs a dry-run of hooks without actually executing them
func (e *Executor) TestExecute(hookPoint HookPoint, variables map[string]string) ([]string, error) {
	if !e.config.IsEnabled() {
		return []string{"Hooks are disabled"}, nil
	}

	actions := e.config.GetHooks(hookPoint.String())
	if len(actions) == 0 {
		return []string{fmt.Sprintf("No hooks configured for %s", hookPoint.String())}, nil
	}

	results := make([]string, 0, len(actions))

	for i, action := range actions {
		executor, exists := e.registry.Get(action.Action)
		if !exists {
			results = append(results, fmt.Sprintf("✗ %s: unknown action", action.Action))
			continue
		}

		// Validate parameters
		if err := executor.Validate(action.Params); err != nil {
			results = append(results, fmt.Sprintf("✗ %s: validation failed: %v", action.Action, err))
			continue
		}

		// Describe what would happen
		description := fmt.Sprintf("✓ %s: %s", action.Action, executor.Description())
		results = append(results, description)

		// Show interpolated values
		if variables != nil {
			params := interpolateParams(action.Params, variables)
			for key, value := range params {
				results = append(results, fmt.Sprintf("  %s: %v", key, value))
			}
		}

		_ = i // avoid unused warning
	}

	return results, nil
}

// interpolateParams replaces template variables in parameters
func interpolateParams(params map[string]interface{}, variables map[string]string) map[string]interface{} {
	result := make(map[string]interface{})
	for key, value := range params {
		switch v := value.(type) {
		case string:
			result[key] = interpolateString(v, variables)
		default:
			result[key] = value
		}
	}
	return result
}

// interpolateString replaces {variable} placeholders with values
func interpolateString(s string, variables map[string]string) string {
	result := s
	for key, value := range variables {
		placeholder := fmt.Sprintf("{%s}", key)
		result = strings.ReplaceAll(result, placeholder, value)
	}
	return result
}

// logError logs an error message (placeholder for future logging)
func logError(message string) {
	// TODO: Implement proper logging to hooks.log
	fmt.Fprintf(stderr(), "goenv hooks: %s\n", message)
}

// stderr returns a writer for error output (allows testing)
var stderr = func() interface{ Write([]byte) (int, error) } {
	return &stderrWriter{}
}

type stderrWriter struct{}

func (w *stderrWriter) Write(p []byte) (int, error) {
	// In production, this would write to os.Stderr or log file
	// For now, just return success
	return len(p), nil
}

// Common validation patterns
var (
	pathTraversalPattern = regexp.MustCompile(`\.\.`)
	controlCharsPattern  = regexp.MustCompile(`[\x00-\x1f\x7f]`)
)

// ValidatePath checks for path traversal attempts
func ValidatePath(path string) error {
	if pathTraversalPattern.MatchString(path) {
		return fmt.Errorf("path contains path traversal sequence (..): %s", path)
	}
	return nil
}

// ValidateString checks for control characters
func ValidateString(s string) error {
	if controlCharsPattern.MatchString(s) {
		return fmt.Errorf("string contains control characters")
	}
	return nil
}

// ValidateURL checks if a URL is allowed
func ValidateURL(urlStr string, allowHTTP, allowInternalIPs bool) error {
	if urlStr == "" {
		return fmt.Errorf("URL is empty")
	}

	// Basic validation
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		return fmt.Errorf("URL must start with http:// or https://")
	}

	// HTTPS enforcement
	if !allowHTTP && strings.HasPrefix(urlStr, "http://") {
		return fmt.Errorf("HTTP URLs are not allowed (use HTTPS or set allow_http: true)")
	}

	// SSRF protection (check for internal IPs)
	if !allowInternalIPs {
		// Extract hostname and check if it's internal
		// This is a simplified check - production would use proper URL parsing
		lowerURL := strings.ToLower(urlStr)
		internalPatterns := []string{
			"localhost", "127.0.0.1", "0.0.0.0",
			"10.", "192.168.", "172.16.", "172.31.",
			"169.254.", // Link-local
			"::1",      // IPv6 localhost
		}
		for _, pattern := range internalPatterns {
			if strings.Contains(lowerURL, pattern) {
				return fmt.Errorf("internal/private IP addresses are not allowed (set allow_internal_ips: true)")
			}
		}
	}

	return nil
}
