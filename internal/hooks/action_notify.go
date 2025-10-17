package hooks

import (
	"fmt"
	"os/exec"
	"runtime"
	"strings"
)

// NotifyDesktopAction sends desktop notifications using OS-native notification systems
type NotifyDesktopAction struct{}

func (a *NotifyDesktopAction) Name() ActionName {
	return ActionNotifyDesktop
}

func (a *NotifyDesktopAction) Description() string {
	return "Sends desktop notifications using OS-native notification systems"
}

func (a *NotifyDesktopAction) Validate(params map[string]interface{}) error {
	// Validate title parameter (required)
	title, ok := params["title"].(string)
	if !ok || title == "" {
		return fmt.Errorf("title parameter is required")
	}
	if err := ValidateString(title); err != nil {
		return fmt.Errorf("invalid title: %w", err)
	}
	if len(title) > 256 {
		return fmt.Errorf("title exceeds maximum length of 256 characters")
	}

	// Validate message parameter (required)
	message, ok := params["message"].(string)
	if !ok || message == "" {
		return fmt.Errorf("message parameter is required")
	}
	if err := ValidateString(message); err != nil {
		return fmt.Errorf("invalid message: %w", err)
	}
	if len(message) > 1024 {
		return fmt.Errorf("message exceeds maximum length of 1024 characters")
	}

	// Validate level parameter (optional)
	if level, ok := params["level"].(string); ok {
		level = strings.ToLower(level)
		if level != "info" && level != "warning" && level != "error" {
			return fmt.Errorf("invalid level: %s (must be info, warning, or error)", level)
		}
	}

	return nil
}

func (a *NotifyDesktopAction) Execute(ctx *HookContext, params map[string]interface{}) error {
	// Get title and interpolate variables
	title := params["title"].(string)
	title = interpolateString(title, ctx.Variables)

	// Get message and interpolate variables
	message := params["message"].(string)
	message = interpolateString(message, ctx.Variables)

	// Get level (default to info)
	level := "info"
	if l, ok := params["level"].(string); ok {
		level = strings.ToLower(l)
	}

	// Send notification based on OS
	return sendNotification(title, message, level)
}

// sendNotification sends a notification using OS-specific methods
func sendNotification(title, message, level string) error {
	switch runtime.GOOS {
	case "darwin":
		return sendMacNotification(title, message, level)
	case "linux":
		return sendLinuxNotification(title, message, level)
	case "windows":
		return sendWindowsNotification(title, message, level)
	default:
		return fmt.Errorf("desktop notifications not supported on %s", runtime.GOOS)
	}
}

// sendMacNotification sends a notification on macOS using osascript
func sendMacNotification(title, message, level string) error {
	// Use AppleScript to display notification
	// Note: This requires the terminal app to have notification permissions
	script := fmt.Sprintf(`display notification "%s" with title "%s"`,
		escapeAppleScript(message),
		escapeAppleScript(title))

	cmd := exec.Command("osascript", "-e", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send macOS notification: %w", err)
	}
	return nil
}

// sendLinuxNotification sends a notification on Linux using notify-send
func sendLinuxNotification(title, message, level string) error {
	// Map level to urgency
	urgency := "normal"
	if level == "warning" {
		urgency = "normal"
	} else if level == "error" {
		urgency = "critical"
	}

	// Check if notify-send is available
	if _, err := exec.LookPath("notify-send"); err != nil {
		return fmt.Errorf("notify-send not found: %w (install libnotify-bin or similar package)", err)
	}

	cmd := exec.Command("notify-send", "-u", urgency, title, message)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send Linux notification: %w", err)
	}
	return nil
}

// sendWindowsNotification sends a notification on Windows using PowerShell
func sendWindowsNotification(title, message, level string) error {
	// Use PowerShell to show a Windows notification
	// This creates a toast notification
	script := fmt.Sprintf(`
[Windows.UI.Notifications.ToastNotificationManager, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.UI.Notifications.ToastNotification, Windows.UI.Notifications, ContentType = WindowsRuntime] | Out-Null
[Windows.Data.Xml.Dom.XmlDocument, Windows.Data.Xml.Dom.XmlDocument, ContentType = WindowsRuntime] | Out-Null

$template = @"
<toast>
    <visual>
        <binding template="ToastText02">
            <text id="1">%s</text>
            <text id="2">%s</text>
        </binding>
    </visual>
</toast>
"@

$xml = New-Object Windows.Data.Xml.Dom.XmlDocument
$xml.LoadXml($template)
$toast = New-Object Windows.UI.Notifications.ToastNotification $xml
[Windows.UI.Notifications.ToastNotificationManager]::CreateToastNotifier("goenv").Show($toast)
`, escapePowerShell(title), escapePowerShell(message))

	cmd := exec.Command("powershell", "-Command", script)
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to send Windows notification: %w", err)
	}
	return nil
}

// escapeAppleScript escapes special characters for AppleScript strings
func escapeAppleScript(s string) string {
	s = strings.ReplaceAll(s, "\\", "\\\\")
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", "\\n")
	return s
}

// escapePowerShell escapes special characters for PowerShell strings
func escapePowerShell(s string) string {
	s = strings.ReplaceAll(s, "`", "``")
	s = strings.ReplaceAll(s, "\"", "`\"")
	s = strings.ReplaceAll(s, "$", "`$")
	s = strings.ReplaceAll(s, "\n", "`n")
	return s
}
