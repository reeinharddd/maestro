package credentials

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"

	"github.com/reeinharrrd/maestro/internal/config"
)

type FileStore struct {
	path string
	mu   sync.RWMutex
	data map[string]map[string]string
}

func NewFileStore(opts map[string]string) (*FileStore, error) {
	dir := config.CredentialsDir()
	if customDir, ok := opts["dir"]; ok && customDir != "" {
		dir = customDir
	}
	if err := os.MkdirAll(dir, 0700); err != nil {
		return nil, fmt.Errorf("create credentials dir: %w", err)
	}
	path := filepath.Join(dir, "credentials.json")
	fs := &FileStore{path: path, data: make(map[string]map[string]string)}
	if err := fs.load(); err != nil {
		return nil, err
	}
	return fs, nil
}

func (fs *FileStore) load() error {
	fs.mu.Lock()
	defer fs.mu.Unlock()

	data, err := os.ReadFile(fs.path)
	if err != nil {
		if os.IsNotExist(err) {
			fs.data = make(map[string]map[string]string)
			return nil
		}
		return fmt.Errorf("read credentials: %w", err)
	}
	if len(data) == 0 {
		fs.data = make(map[string]map[string]string)
		return nil
	}
	return json.Unmarshal(data, &fs.data)
}

func (fs *FileStore) save() error {
	fs.mu.RLock()
	defer fs.mu.RUnlock()

	data, err := json.MarshalIndent(fs.data, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal credentials: %w", err)
	}
	tmpPath := fs.path + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0600); err != nil {
		return fmt.Errorf("write credentials temp: %w", err)
	}
	return os.Rename(tmpPath, fs.path)
}

func (fs *FileStore) Get(ctx context.Context, service, key string) (string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	if svc, ok := fs.data[service]; ok {
		return svc[key], nil
	}
	return "", nil
}

func (fs *FileStore) Set(ctx context.Context, service, key, value string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if fs.data[service] == nil {
		fs.data[service] = make(map[string]string)
	}
	fs.data[service][key] = value
	return fs.save()
}

func (fs *FileStore) Delete(ctx context.Context, service, key string) error {
	fs.mu.Lock()
	defer fs.mu.Unlock()
	if svc, ok := fs.data[service]; ok {
		delete(svc, key)
		if len(svc) == 0 {
			delete(fs.data, service)
		}
		return fs.save()
	}
	return nil
}

func (fs *FileStore) List(ctx context.Context, service string) (map[string]string, error) {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	if svc, ok := fs.data[service]; ok {
		result := make(map[string]string, len(svc))
		for k, v := range svc {
			result[k] = v
		}
		return result, nil
	}
	return make(map[string]string), nil
}

func (fs *FileStore) Test(ctx context.Context, service string) error {
	fs.mu.RLock()
	defer fs.mu.RUnlock()
	if _, ok := fs.data[service]; ok {
		return nil
	}
	return fmt.Errorf("service %q not found", service)
}

func (fs *FileStore) Name() string {
	return "file"
}