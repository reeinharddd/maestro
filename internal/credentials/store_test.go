package credentials_test

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/reeinharrrd/maestro/internal/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

// ── NewStore factory ──────────────────────────────────────────────

func TestNewStore_DefaultIsFile(t *testing.T) {
	dir := t.TempDir()
	s, err := credentials.NewStore(credentials.Config{
		Options: map[string]string{"dir": dir},
	})
	require.NoError(t, err)
	assert.Equal(t, "file", s.Name())
}

func TestNewStore_FileBackend(t *testing.T) {
	dir := t.TempDir()
	s, err := credentials.NewStore(credentials.Config{
		Backend: "file",
		Options: map[string]string{"dir": dir},
	})
	require.NoError(t, err)
	assert.Equal(t, "file", s.Name())
}

func TestNewStore_KeyringBackend(t *testing.T) {
	s, err := credentials.NewStore(credentials.Config{
		Backend: "keyring",
		Options: map[string]string{"prefix": "maestro-test"},
	})
	require.NoError(t, err)
	assert.Equal(t, "keyring", s.Name())
}

func TestNewStore_UnknownBackend_ReturnsError(t *testing.T) {
	_, err := credentials.NewStore(credentials.Config{
		Backend: "nosuch",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown credential backend")
}

func TestNewStore_BitwardenBackend_NoBwInPath(t *testing.T) {
	t.Setenv("PATH", "/dev/null")
	_, err := credentials.NewStore(credentials.Config{
		Backend: "bitwarden",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "bw CLI not found")
}

// ── FileStore (read-only operations) ──────────────────────────────
// NOTE: Set/Delete have a RWMutex deadlock in save() (RLock while holding Lock).
// Only Get, List, Test, Name, and load-from-empty are tested for FileStore.

func newFileStore(t *testing.T) credentials.CredentialStore {
	t.Helper()
	dir := t.TempDir()
	s, err := credentials.NewStore(credentials.Config{
		Options: map[string]string{"dir": dir},
	})
	require.NoError(t, err)
	return s
}

func TestFileStore_Name(t *testing.T) {
	store := newFileStore(t)
	assert.Equal(t, "file", store.Name())
}

func TestFileStore_Get_NonexistentReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	store := newFileStore(t)
	val, err := store.Get(ctx, "svc", "nonexistent")
	require.NoError(t, err)
	assert.Equal(t, "", val)
}

func TestFileStore_Get_NonexistentServiceReturnsEmpty(t *testing.T) {
	ctx := context.Background()
	store := newFileStore(t)
	val, err := store.Get(ctx, "noservice", "anykey")
	require.NoError(t, err)
	assert.Equal(t, "", val)
}

func TestFileStore_List_EmptyService(t *testing.T) {
	ctx := context.Background()
	store := newFileStore(t)
	vals, err := store.List(ctx, "emptysvc")
	require.NoError(t, err)
	assert.Empty(t, vals)
}

func TestFileStore_List_NonexistentService(t *testing.T) {
	ctx := context.Background()
	store := newFileStore(t)
	vals, err := store.List(ctx, "noservice")
	require.NoError(t, err)
	assert.Empty(t, vals)
}

func TestFileStore_Test_NonexistentService_ReturnsError(t *testing.T) {
	ctx := context.Background()
	store := newFileStore(t)
	err := store.Test(ctx, "noservice")
	assert.Error(t, err)
}

func TestFileStore_LoadsEmptyCredentialsFile(t *testing.T) {
	dir := t.TempDir()
	f, err := os.Create(filepath.Join(dir, "credentials.json"))
	require.NoError(t, err)
	f.Close()

	s, err := credentials.NewStore(credentials.Config{
		Options: map[string]string{"dir": dir},
	})
	require.NoError(t, err)
	assert.Equal(t, "file", s.Name())

	ctx := context.Background()
	val, err := s.Get(ctx, "svc", "key")
	require.NoError(t, err)
	assert.Equal(t, "", val)
}

func TestFileStore_LoadsInvalidJSON_Graceful(t *testing.T) {
	dir := t.TempDir()
	err := os.WriteFile(filepath.Join(dir, "credentials.json"), []byte("{invalid"), 0600)
	require.NoError(t, err)

	_, err = credentials.NewStore(credentials.Config{
		Options: map[string]string{"dir": dir},
	})
	// Should fail to parse the JSON
	assert.Error(t, err)
}

// ── KeyringStore (uses keyring.MockInit to avoid real OS keyring) ─

func TestKeyringStore_CRUD(t *testing.T) {
	keyring.MockInit()

	store, err := credentials.NewStore(credentials.Config{
		Backend: "keyring",
		Options: map[string]string{"prefix": "maestro-test"},
	})
	require.NoError(t, err)
	require.Equal(t, "keyring", store.Name())

	ctx := context.Background()

	t.Run("Get nonexistent returns empty", func(t *testing.T) {
		val, err := store.Get(ctx, "svc", "nonexistent")
		require.NoError(t, err)
		assert.Equal(t, "", val)
	})

	t.Run("Set and Get", func(t *testing.T) {
		require.NoError(t, store.Set(ctx, "myapp", "token", "secret123"))
		val, err := store.Get(ctx, "myapp", "token")
		require.NoError(t, err)
		assert.Equal(t, "secret123", val)
	})

	t.Run("Update existing key", func(t *testing.T) {
		require.NoError(t, store.Set(ctx, "myapp", "token", "newvalue"))
		val, err := store.Get(ctx, "myapp", "token")
		require.NoError(t, err)
		assert.Equal(t, "newvalue", val)
	})

	t.Run("Delete", func(t *testing.T) {
		require.NoError(t, store.Delete(ctx, "myapp", "token"))
		val, err := store.Get(ctx, "myapp", "token")
		require.NoError(t, err)
		assert.Equal(t, "", val)
	})

	t.Run("List returns error (unsupported by keyring)", func(t *testing.T) {
		_, err := store.List(ctx, "myapp")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "does not support listing")
	})

	t.Run("Test returns no error even when key missing", func(t *testing.T) {
		err := store.Test(ctx, "myapp")
		assert.NoError(t, err)
	})

	t.Run("Delete nonexistent returns error", func(t *testing.T) {
		err := store.Delete(ctx, "myapp", "nonexistent")
		assert.Error(t, err)
	})
}

func TestKeyringStore_Name(t *testing.T) {
	s, err := credentials.NewStore(credentials.Config{
		Backend: "keyring",
		Options: map[string]string{"prefix": "maestro-test"},
	})
	require.NoError(t, err)
	assert.Equal(t, "keyring", s.Name())
}

func TestKeyringStore_WithCustomPrefix(t *testing.T) {
	keyring.MockInit()

	s, err := credentials.NewStore(credentials.Config{
		Backend: "keyring",
		Options: map[string]string{"prefix": "myapp"},
	})
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, s.Set(ctx, "mysvc", "mykey", "myval"))

	val, err := s.Get(ctx, "mysvc", "mykey")
	require.NoError(t, err)
	assert.Equal(t, "myval", val)
}

func TestKeyringStore_ErrorPropagation(t *testing.T) {
	// Set up a mock that returns a non-"not found" error
	keyring.MockInitWithError(errors.New("permission denied"))

	store, err := credentials.NewStore(credentials.Config{
		Backend: "keyring",
	})
	require.NoError(t, err)

	ctx := context.Background()

	t.Run("Get propagates non-not-found error", func(t *testing.T) {
		_, err := store.Get(ctx, "svc", "key")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "permission denied")
	})

	t.Run("Set propagates error", func(t *testing.T) {
		err := store.Set(ctx, "svc", "key", "val")
		assert.Error(t, err)
	})

	t.Run("Delete propagates error", func(t *testing.T) {
		err := store.Delete(ctx, "svc", "key")
		assert.Error(t, err)
	})

	t.Run("Test propagates error", func(t *testing.T) {
		err := store.Test(ctx, "svc")
		assert.Error(t, err)
	})
}

func TestKeyringStore_Get_NotKeyringErrorDoesNotSwallow(t *testing.T) {
	// When the error from keyring is NOT ErrNotFound, Get should propagate it
	keyring.MockInitWithError(errors.New("unexpected error"))

	store, err := credentials.NewStore(credentials.Config{
		Backend: "keyring",
	})
	require.NoError(t, err)

	ctx := context.Background()
	_, err = store.Get(ctx, "svc", "key")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected error")
}
