package hooks

import (
	"strings"
	"testing"
)

func TestDefaultRegistry(t *testing.T) {
	registry := DefaultRegistry()
	if registry == nil {
		t.Fatal("DefaultRegistry() returned nil")
	}

	// Verify all 5 actions are registered
	expectedActions := []string{
		"log_to_file",
		"http_webhook",
		"notify_desktop",
		"check_disk_space",
		"set_env",
	}

	for _, actionName := range expectedActions {
		executor, exists := registry.Get(actionName)
		if !exists {
			t.Errorf("DefaultRegistry() missing action: %s", actionName)
		}
		if executor == nil {
			t.Errorf("DefaultRegistry() executor for %s is nil", actionName)
		}
	}
}

func TestRegistryRegister(t *testing.T) {
	registry := NewRegistry()

	// Create a mock executor
	mockExecutor := &LogToFileAction{}

	// Register an action
	err := registry.Register(mockExecutor)
	if err != nil {
		t.Errorf("Registry.Register() unexpected error: %v", err)
	}

	// Verify it can be retrieved
	executor, exists := registry.Get(mockExecutor.Name().String())
	if !exists {
		t.Error("Registry.Get() could not find registered action")
	}
	if executor == nil {
		t.Error("Registry.Get() returned nil executor")
	}

	// Test duplicate registration
	err = registry.Register(mockExecutor)
	if err == nil {
		t.Error("Registry.Register() expected error for duplicate registration, got nil")
	}
}

// MockEmptyNameAction is a test action that returns an empty name (zero value)
type MockEmptyNameAction struct{}

func (a *MockEmptyNameAction) Name() ActionName {
	return "" // Empty/zero value
}

func (a *MockEmptyNameAction) Description() string {
	return "Test action with empty name"
}

func (a *MockEmptyNameAction) Validate(params map[string]interface{}) error {
	return nil
}

func (a *MockEmptyNameAction) Execute(ctx *HookContext, params map[string]interface{}) error {
	return nil
}

// MockInvalidNameAction is a test action that returns an invalid name
type MockInvalidNameAction struct{}

func (a *MockInvalidNameAction) Name() ActionName {
	return ActionName("invalid_action_name")
}

func (a *MockInvalidNameAction) Description() string {
	return "Test action with invalid name"
}

func (a *MockInvalidNameAction) Validate(params map[string]interface{}) error {
	return nil
}

func (a *MockInvalidNameAction) Execute(ctx *HookContext, params map[string]interface{}) error {
	return nil
}

func TestRegistryRegisterInvalidNames(t *testing.T) {
	tests := []struct {
		name     string
		executor ActionExecutor
		wantErr  bool
		errMsg   string
	}{
		{
			name:     "Empty action name",
			executor: &MockEmptyNameAction{},
			wantErr:  true,
			errMsg:   "action name cannot be empty",
		},
		{
			name:     "Invalid action name",
			executor: &MockInvalidNameAction{},
			wantErr:  true,
			errMsg:   "not a valid action name",
		},
		{
			name:     "Valid action name",
			executor: &LogToFileAction{},
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			registry := NewRegistry()
			err := registry.Register(tt.executor)

			if tt.wantErr {
				if err == nil {
					t.Errorf("Registry.Register() expected error containing %q, got nil", tt.errMsg)
				} else if tt.errMsg != "" && !strings.Contains(err.Error(), tt.errMsg) {
					t.Errorf("Registry.Register() error = %q, want error containing %q", err.Error(), tt.errMsg)
				}
			} else {
				if err != nil {
					t.Errorf("Registry.Register() unexpected error: %v", err)
				}
			}
		})
	}
}

func TestRegistryGet(t *testing.T) {
	registry := NewRegistry()
	mockExecutor := &LogToFileAction{}
	_ = registry.Register(mockExecutor)

	tests := []struct {
		name        string
		actionName  string
		shouldExist bool
	}{
		{"Existing action", "log_to_file", true},
		{"Non-existent action", "nonexistent", false},
		{"Empty string", "", false},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			executor, exists := registry.Get(tt.actionName)
			if exists != tt.shouldExist {
				t.Errorf("Registry.Get(%q) exists = %v, want %v", tt.actionName, exists, tt.shouldExist)
			}
			if tt.shouldExist && executor == nil {
				t.Errorf("Registry.Get(%q) returned nil executor", tt.actionName)
			}
			if !tt.shouldExist && executor != nil {
				t.Errorf("Registry.Get(%q) expected nil executor, got %v", tt.actionName, executor)
			}
		})
	}
}

func TestRegistryList(t *testing.T) {
	registry := NewRegistry()

	// Register multiple actions
	_ = registry.Register(&LogToFileAction{})
	_ = registry.Register(&HTTPWebhookAction{})
	_ = registry.Register(&NotifyDesktopAction{})

	actions := registry.List()

	if len(actions) != 3 {
		t.Errorf("Registry.List() returned %d actions, want 3", len(actions))
	}

	// Verify all registered actions are in the list
	expected := map[string]bool{
		"log_to_file":    false,
		"http_webhook":   false,
		"notify_desktop": false,
	}

	for _, name := range actions {
		if _, ok := expected[name]; ok {
			expected[name] = true
		}
	}

	for name, found := range expected {
		if !found {
			t.Errorf("Registry.List() missing action: %s", name)
		}
	}
}

func TestRegistryConcurrency(t *testing.T) {
	registry := DefaultRegistry()

	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, exists := registry.Get("log_to_file")
			if !exists {
				t.Error("Concurrent Get() failed to find log_to_file")
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
