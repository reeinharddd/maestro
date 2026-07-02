package credentials

import (
	"context"
	"os"
)

var defaultStore CredentialStore

// SetDefaultStore sets a global credential store used as fallback
// when ResolveKey doesn't find the key in the environment.
// Called during initialization (e.g. from maestro init or CLI setup).
func SetDefaultStore(s CredentialStore) {
	defaultStore = s
}

// ResolveKey resolves an API key by checking the environment first,
// then falling back to the global DefaultStore (if set).
func ResolveKey(ctx context.Context, envName string) string {
	if v := os.Getenv(envName); v != "" {
		return v
	}
	if defaultStore != nil {
		v, err := defaultStore.Get(ctx, "opencode", envName)
		if err == nil && v != "" {
			return v
		}
	}
	return ""
}
