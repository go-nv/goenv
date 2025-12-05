package hooks

import (
	"fmt"
	"sync"
)

// ActionExecutor defines the interface for hook actions
type ActionExecutor interface {
	// Name returns the action name
	Name() ActionName

	// Execute runs the action with the given context and parameters
	Execute(ctx *HookContext, params map[string]interface{}) error

	// Validate checks if the parameters are valid
	Validate(params map[string]interface{}) error

	// Description returns a human-readable description
	Description() string
}

// Registry manages available hook actions
type Registry struct {
	mu      sync.RWMutex
	actions map[ActionName]ActionExecutor
}

// NewRegistry creates a new action registry
func NewRegistry() *Registry {
	return &Registry{
		actions: make(map[ActionName]ActionExecutor),
	}
}

// Register adds an action executor to the registry
func (r *Registry) Register(executor ActionExecutor) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	name := executor.Name()

	// Validate that the name is not empty (zero value)
	if name == "" {
		return fmt.Errorf("action name cannot be empty")
	}

	// Validate that the name is a known action
	if !IsValidActionName(name.String()) {
		return fmt.Errorf("action %s is not a valid action name", name)
	}

	if _, exists := r.actions[name]; exists {
		return fmt.Errorf("action %s is already registered", name)
	}

	r.actions[name] = executor
	return nil
}

// Get retrieves an action executor by name
func (r *Registry) Get(name string) (ActionExecutor, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	// Return false for empty names
	if name == "" {
		return nil, false
	}

	executor, exists := r.actions[ActionName(name)]
	return executor, exists
}

// List returns all registered action names
func (r *Registry) List() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	names := make([]string, 0, len(r.actions))
	for name := range r.actions {
		names = append(names, name.String())
	}
	return names
}

// GetAll returns all registered executors
func (r *Registry) GetAll() map[string]ActionExecutor {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]ActionExecutor, len(r.actions))
	for name, executor := range r.actions {
		result[name.String()] = executor
	}
	return result
}

var (
	defaultRegistry     *Registry
	defaultRegistryOnce sync.Once
)

// DefaultRegistry returns the global action registry
func DefaultRegistry() *Registry {
	defaultRegistryOnce.Do(func() {
		defaultRegistry = NewRegistry()
		registerDefaultActions(defaultRegistry)
	})
	return defaultRegistry
}

// registerDefaultActions registers all built-in actions
func registerDefaultActions(r *Registry) {
	// Core actions (Phase 1)
	r.Register(&LogToFileAction{})
	r.Register(&HTTPWebhookAction{})
	r.Register(&NotifyDesktopAction{})
	r.Register(&CheckDiskSpaceAction{})
	r.Register(&SetEnvAction{})
	r.Register(&RunCommandAction{})
}
