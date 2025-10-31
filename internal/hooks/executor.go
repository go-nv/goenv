package hooks

import (
	"context"
	"fmt"
	"net"
	"net/url"
	"os"
	"regexp"
	"strings"
	"time"

	"github.com/go-nv/goenv/internal/utils"
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
			results = append(results, fmt.Sprintf("%s%s: unknown action", utils.Emoji("✗ "), action.Action))
			continue
		}

		// Validate parameters
		if err := executor.Validate(action.Params); err != nil {
			results = append(results, fmt.Sprintf("%s%s: validation failed: %v", utils.Emoji("✗ "), action.Action, err))
			continue
		}

		// Describe what would happen
		description := fmt.Sprintf("%s%s: %s", utils.Emoji("✓ "), action.Action, executor.Description())
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

// logError logs an error message to both stderr and optional log file
func logError(message string) {
	timestamped := fmt.Sprintf("[%s] %s", time.Now().Format("2006-01-02 15:04:05"), message)

	// Always write to stderr
	fmt.Fprintf(stderr(), "goenv hooks: %s\n", message)

	// Optionally write to log file if GOENV_HOOKS_LOG is set
	if logPath := utils.GoenvEnvVarHooksLog.UnsafeValue(); logPath != "" {
		// Open or create log file with append mode
		f, err := os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			// Can't log the error without recursing, so just skip
			return
		}
		defer f.Close()

		// Write timestamped message
		fmt.Fprintf(f, "%s\n", timestamped)
	}
}

// stderr returns a writer for error output (allows testing)
var stderr = func() interface{ Write([]byte) (int, error) } {
	return &stderrWriter{}
}

type stderrWriter struct{}

func (w *stderrWriter) Write(p []byte) (int, error) {
	// In production, write to os.Stderr
	return os.Stderr.Write(p)
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
	return ValidateURLWithStrictDNS(urlStr, allowHTTP, allowInternalIPs, false)
}

// ValidateURLWithStrictDNS checks if a URL is allowed with optional strict DNS mode
func ValidateURLWithStrictDNS(urlStr string, allowHTTP, allowInternalIPs, strictDNS bool) error {
	if urlStr == "" {
		return fmt.Errorf("URL is empty")
	}

	// Parse URL properly
	parsedURL, err := url.Parse(urlStr)
	if err != nil {
		return fmt.Errorf("invalid URL: %w", err)
	}

	// Validate scheme
	if parsedURL.Scheme != "http" && parsedURL.Scheme != "https" {
		return fmt.Errorf("URL must use http or https scheme")
	}

	// HTTPS enforcement
	if !allowHTTP && parsedURL.Scheme == "http" {
		return fmt.Errorf("HTTP URLs are not allowed (use HTTPS or set allow_http: true)")
	}

	// SSRF protection - check if target resolves to internal/private IPs
	if !allowInternalIPs {
		hostname := parsedURL.Hostname()
		if hostname == "" {
			return fmt.Errorf("URL must have a hostname")
		}

		// First check: literal IP addresses in hostname
		if ip := net.ParseIP(hostname); ip != nil {
			if isPrivateIP(ip) {
				return fmt.Errorf("internal/private IP addresses are not allowed (set allow_internal_ips: true)")
			}
			return nil
		}

		// Second check: resolve hostname to IP addresses and validate each
		ips, err := net.LookupIP(hostname)
		if err != nil {
			// DNS resolution failed
			if strictDNS {
				// Strict mode: reject when DNS fails and internal IPs are not allowed
				return fmt.Errorf("DNS resolution failed and strict_dns is enabled: %w", err)
			}

			// Non-strict mode (default): fall back to substring checks as defense in depth
			// This catches some obvious cases even if DNS is unavailable
			lowerHost := strings.ToLower(hostname)
			suspiciousPatterns := []string{
				"localhost", "127.0.0.1", "0.0.0.0",
				".local", ".internal", ".lan",
			}
			for _, pattern := range suspiciousPatterns {
				if strings.Contains(lowerHost, pattern) {
					return fmt.Errorf("hostname appears to target internal resources (set allow_internal_ips: true)")
				}
			}
			// DNS failure but no obvious internal patterns - allow but could log warning
			return nil
		}

		// Check all resolved IPs
		for _, ip := range ips {
			if isPrivateIP(ip) {
				return fmt.Errorf("hostname resolves to internal/private IP %s (set allow_internal_ips: true)", ip.String())
			}
		}
	}

	return nil
}

// isPrivateIP checks if an IP address is private/internal according to RFCs
func isPrivateIP(ip net.IP) bool {
	// Define private/internal IP ranges
	privateIPBlocks := []string{
		// IPv4 private ranges (RFC1918)
		"10.0.0.0/8",
		"172.16.0.0/12",
		"192.168.0.0/16",

		// IPv4 carrier-grade NAT (RFC6598)
		"100.64.0.0/10",

		// IPv4 loopback (RFC1122)
		"127.0.0.0/8",

		// IPv4 link-local (RFC3927)
		"169.254.0.0/16",

		// IPv4 broadcast
		"255.255.255.255/32",

		// IPv6 loopback (RFC4291)
		"::1/128",

		// IPv6 link-local (RFC4291)
		"fe80::/10",

		// IPv6 unique local (RFC4193)
		"fc00::/7",

		// IPv6 documentation (RFC3849)
		"2001:db8::/32",
	}

	for _, block := range privateIPBlocks {
		_, ipNet, err := net.ParseCIDR(block)
		if err != nil {
			continue
		}
		if ipNet.Contains(ip) {
			return true
		}
	}

	return false
}
