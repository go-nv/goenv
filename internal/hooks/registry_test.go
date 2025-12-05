package hooks

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultRegistry(t *testing.T) {
	registry := DefaultRegistry()
	require.NotNil(t, registry, "DefaultRegistry() returned nil")

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
		assert.True(t, exists, "DefaultRegistry() missing action")
		assert.NotNil(t, executor, "DefaultRegistry() executor for is nil")
	}
}

func TestRegistryRegister(t *testing.T) {
	registry := NewRegistry()

	// Create a mock executor
	mockExecutor := &LogToFileAction{}

	// Register an action
	err := registry.Register(mockExecutor)
	assert.NoError(t, err, "Registry.Register() unexpected error")

	// Verify it can be retrieved
	executor, exists := registry.Get(mockExecutor.Name().String())
	assert.True(t, exists, "Registry.Get() could not find registered action")
	assert.NotNil(t, executor, "Registry.Get() returned nil executor")

	// Test duplicate registration
	err = registry.Register(mockExecutor)
	assert.Error(t, err, "Registry.Register() expected error for duplicate registration, got nil")
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
				assert.NoError(t, err, "Registry.Register() unexpected error")
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
			assert.Equal(t, tt.shouldExist, exists, "Registry.Get() exists = %v", tt.actionName)
			assert.False(t, tt.shouldExist && executor == nil, "Registry.Get() returned nil executor")
			assert.False(t, !tt.shouldExist && executor != nil, "Registry.Get() expected nil executor")
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

	assert.Len(t, actions, 3, "Registry.List() returned actions")

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

	for _, found := range expected {
		assert.True(t, found, "Registry.List() missing action")
	}
}

func TestRegistryConcurrency(t *testing.T) {
	registry := DefaultRegistry()

	// Test concurrent access
	done := make(chan bool, 10)
	for i := 0; i < 10; i++ {
		go func() {
			_, exists := registry.Get("log_to_file")
			assert.True(t, exists, "Concurrent Get() failed to find log_to_file")
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}
