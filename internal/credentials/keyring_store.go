package credentials

import (
	"context"
	"fmt"
	"strings"

	"github.com/zalando/go-keyring"
)

type KeyringStore struct {
	servicePrefix string
}

func NewKeyringStore(opts map[string]string) (*KeyringStore, error) {
	prefix := "maestro"
	if customPrefix, ok := opts["prefix"]; ok && customPrefix != "" {
		prefix = customPrefix
	}
	return &KeyringStore{servicePrefix: prefix}, nil
}

func (ks *KeyringStore) keyName(service, key string) string {
	return fmt.Sprintf("%s/%s/%s", ks.servicePrefix, service, key)
}

func (ks *KeyringStore) serviceName(service string) string {
	return fmt.Sprintf("%s/%s", ks.servicePrefix, service)
}

func (ks *KeyringStore) Get(ctx context.Context, service, key string) (string, error) {
	val, err := keyring.Get(ks.serviceName(service), ks.keyName(service, key))
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return "", nil
		}
		return "", err
	}
	return val, nil
}

func (ks *KeyringStore) Set(ctx context.Context, service, key, value string) error {
	return keyring.Set(ks.serviceName(service), ks.keyName(service, key), value)
}

func (ks *KeyringStore) Delete(ctx context.Context, service, key string) error {
	return keyring.Delete(ks.serviceName(service), ks.keyName(service, key))
}

func (ks *KeyringStore) List(ctx context.Context, service string) (map[string]string, error) {
	return nil, fmt.Errorf("keyring backend does not support listing all keys for a service")
}

func (ks *KeyringStore) Test(ctx context.Context, service string) error {
	val, err := keyring.Get(ks.serviceName(service), "test")
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return nil
		}
		return err
	}
	_ = val
	return nil
}

func (ks *KeyringStore) Name() string {
	return "keyring"
}