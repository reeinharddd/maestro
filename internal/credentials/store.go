package credentials

import (
	"context"
	"fmt"
	"strings"
)

type CredentialStore interface {
	Get(ctx context.Context, service, key string) (string, error)
	Set(ctx context.Context, service, key, value string) error
	Delete(ctx context.Context, service, key string) error
	List(ctx context.Context, service string) (map[string]string, error)
	Test(ctx context.Context, service string) error
	Name() string
}

type Config struct {
	Backend string
	Options map[string]string
}

func NewStore(cfg Config) (CredentialStore, error) {
	backend := strings.ToLower(strings.TrimSpace(cfg.Backend))
	if backend == "" {
		backend = "file"
	}
	switch backend {
	case "file":
		return NewFileStore(cfg.Options)
	case "bitwarden", "bw":
		return NewBitwardenStore(cfg.Options)
	case "keyring":
		return NewKeyringStore(cfg.Options)
	default:
		return nil, fmt.Errorf("unknown credential backend: %s (supported: file, bitwarden, keyring)", backend)
	}
}

type Capability struct {
	CanList    bool
	CanTest    bool
	CanDelete  bool
	Persistent bool
}