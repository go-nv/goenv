package manager

import "context"

// Context key type for type-safe context storage
type managerContextKey string

const ManagerContextKey managerContextKey = "goenv.manager"

// ToContext stores the manager in the context.
func ToContext(ctx context.Context, mgr *Manager) context.Context {
	return context.WithValue(ctx, ManagerContextKey, mgr)
}

// FromContext retrieves the manager from the context.
// Returns nil if not found or if context is nil.
func FromContext(ctx context.Context) *Manager {
	if ctx == nil {
		return nil
	}
	if mgr, ok := ctx.Value(ManagerContextKey).(*Manager); ok {
		return mgr
	}
	return nil
}
