package hooks

// ActionName represents a valid action type
type ActionName string

// Standard actions available in goenv hooks
const (
	// LogToFile writes log entries to files
	ActionLogToFile ActionName = "log_to_file"

	// HTTPWebhook sends HTTP requests to webhook URLs
	ActionHTTPWebhook ActionName = "http_webhook"

	// NotifyDesktop sends desktop notifications
	ActionNotifyDesktop ActionName = "notify_desktop"

	// CheckDiskSpace validates available disk space
	ActionCheckDiskSpace ActionName = "check_disk_space"

	// SetEnv sets environment variables
	ActionSetEnv ActionName = "set_env"

	// RunCommand executes a shell command
	ActionRunCommand ActionName = "run_command"
)

// String returns the string representation of the action name
func (a ActionName) String() string {
	return string(a)
}

// AllActionNames returns all valid action names
func AllActionNames() []ActionName {
	return []ActionName{
		ActionLogToFile,
		ActionHTTPWebhook,
		ActionNotifyDesktop,
		ActionCheckDiskSpace,
		ActionSetEnv,
		ActionRunCommand,
	}
}

// IsValidActionName checks if a string represents a valid action name
func IsValidActionName(s string) bool {
	for _, action := range AllActionNames() {
		if string(action) == s {
			return true
		}
	}
	return false
}
