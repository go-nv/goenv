package config

import "context"

// Context key type for type-safe context storage
type configContextKey string

const ConfigContextKey configContextKey = "goenv.config"

// ToContext stores the config in the context.
func ToContext(ctx context.Context, cfg *Config) context.Context {
	return context.WithValue(ctx, ConfigContextKey, cfg)
}

// FromContext retrieves the config from the context.
// Returns nil if not found or if context is nil.
func FromContext(ctx context.Context) *Config {
	if ctx == nil {
		return nil
	}
	if cfg, ok := ctx.Value(ConfigContextKey).(*Config); ok {
		return cfg
	}
	return nil
}
