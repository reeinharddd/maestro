package credentials_test

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/reeinharrrd/maestro/internal/credentials"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/zalando/go-keyring"
)

// ── FileStore with pre-populated data ────────────────────────────
// NOTE: FileStore.Set/Delete have a RWMutex deadlock in save()
// (RLock while holding Lock). We work around this by pre-populating
// the JSON file before opening the store for Get/List/Test tests.

func TestFileStore_Get_ExistingKey(t *testing.T) {
	t.Parallel()
	dir, store := prepopulatedFileStore(t, map[string]map[string]string{
		"openai": {"api_key": "sk-test123"},
	})
	_ = dir

	ctx := context.Background()
	val, err := store.Get(ctx, "openai", "api_key")
	require.NoError(t, err)
	assert.Equal(t, "sk-test123", val)
}

func TestFileStore_Get_NonexistentKeyInExistingService(t *testing.T) {
	t.Parallel()
	_, store := prepopulatedFileStore(t, map[string]map[string]string{
		"openai": {"api_key": "sk-test123"},
	})

	ctx := context.Background()
	val, err := store.Get(ctx, "openai", "nonexistent")
	require.NoError(t, err)
	assert.Equal(t, "", val)
}

func TestFileStore_Get_NonexistentService_FromPopulated(t *testing.T) {
	t.Parallel()
	_, store := prepopulatedFileStore(t, map[string]map[string]string{
		"openai": {"api_key": "sk-test123"},
	})

	ctx := context.Background()
	val, err := store.Get(ctx, "noservice", "anykey")
	require.NoError(t, err)
	assert.Equal(t, "", val)
}

func TestFileStore_List_PopulatedService(t *testing.T) {
	t.Parallel()
	_, store := prepopulatedFileStore(t, map[string]map[string]string{
		"svc1": {"k1": "v1", "k2": "v2"},
	})

	ctx := context.Background()
	vals, err := store.List(ctx, "svc1")
	require.NoError(t, err)
	assert.Equal(t, "v1", vals["k1"])
	assert.Equal(t, "v2", vals["k2"])
	assert.Len(t, vals, 2)
}

func TestFileStore_List_NonexistentService_FromPopulated(t *testing.T) {
	t.Parallel()
	_, store := prepopulatedFileStore(t, map[string]map[string]string{
		"svc1": {"k1": "v1"},
	})

	ctx := context.Background()
	vals, err := store.List(ctx, "noservice")
	require.NoError(t, err)
	assert.Empty(t, vals)
}

func TestFileStore_Test_ExistingService(t *testing.T) {
	t.Parallel()
	_, store := prepopulatedFileStore(t, map[string]map[string]string{
		"mysvc": {"key": "val"},
	})

	ctx := context.Background()
	err := store.Test(ctx, "mysvc")
	assert.NoError(t, err)
}

func TestFileStore_Test_NonexistentService_FromPopulated(t *testing.T) {
	t.Parallel()
	_, store := prepopulatedFileStore(t, map[string]map[string]string{
		"mysvc": {"key": "val"},
	})

	ctx := context.Background()
	err := store.Test(ctx, "noservice")
	assert.Error(t, err)
}

func TestFileStore_Delete_NonExistentService_Noop(t *testing.T) {
	t.Parallel()
	_, store := prepopulatedFileStore(t, map[string]map[string]string{
		"svc1": {"k1": "v1"},
	})

	ctx := context.Background()
	err := store.Delete(ctx, "nonexistent", "anykey")
	assert.NoError(t, err)

	// Verify existing data untouched
	val, err := store.Get(ctx, "svc1", "k1")
	require.NoError(t, err)
	assert.Equal(t, "v1", val)
}

func TestFileStore_MultipleServices(t *testing.T) {
	t.Parallel()
	_, store := prepopulatedFileStore(t, map[string]map[string]string{
		"svc1": {"key1": "val1"},
		"svc2": {"key2": "val2"},
	})

	ctx := context.Background()

	v1, err := store.Get(ctx, "svc1", "key1")
	require.NoError(t, err)
	assert.Equal(t, "val1", v1)

	v2, err := store.Get(ctx, "svc2", "key2")
	require.NoError(t, err)
	assert.Equal(t, "val2", v2)
}

// ── KeyringStore edge cases ─────────────────────────────────────

func TestKeyringStore_SpecialCharacters(t *testing.T) {
	keyring.MockInit()

	store, err := credentials.NewStore(credentials.Config{
		Backend: "keyring",
		Options: map[string]string{"prefix": "maestro-edge"},
	})
	require.NoError(t, err)

	ctx := context.Background()
	tests := []struct {
		name  string
		value string
	}{
		{name: "unicode", value: "héllo 🚀 world"},
		{name: "json", value: `{"key": "value", "nested": [1,2,3]}`},
		{name: "symbols", value: "!@#$%^&*()_+-=[]{}|;':\",./<>?~"},
		{name: "spaces", value: "   spaced   value   "},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			require.NoError(t, store.Set(ctx, "special", tc.name, tc.value))
			got, err := store.Get(ctx, "special", tc.name)
			require.NoError(t, err)
			assert.Equal(t, tc.value, got)
		})
	}
}

func TestKeyringStore_EmptyValue(t *testing.T) {
	keyring.MockInit()

	store, err := credentials.NewStore(credentials.Config{
		Backend: "keyring",
		Options: map[string]string{"prefix": "maestro-edge"},
	})
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, store.Set(ctx, "svc", "emptykey", ""))

	val, err := store.Get(ctx, "svc", "emptykey")
	require.NoError(t, err)
	assert.Equal(t, "", val)
}

func TestKeyringStore_Overwrite(t *testing.T) {
	keyring.MockInit()

	store, err := credentials.NewStore(credentials.Config{
		Backend: "keyring",
		Options: map[string]string{"prefix": "maestro-edge"},
	})
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, store.Set(ctx, "svc", "key", "original"))
	require.NoError(t, store.Set(ctx, "svc", "key", "updated"))

	val, err := store.Get(ctx, "svc", "key")
	require.NoError(t, err)
	assert.Equal(t, "updated", val)
}

func TestKeyringStore_ListReturnsError(t *testing.T) {
	keyring.MockInit()

	store, err := credentials.NewStore(credentials.Config{
		Backend: "keyring",
	})
	require.NoError(t, err)

	ctx := context.Background()
	_, err = store.List(ctx, "svc")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not support listing")
}

func TestKeyringStore_MultipleKeys(t *testing.T) {
	keyring.MockInit()

	store, err := credentials.NewStore(credentials.Config{
		Backend: "keyring",
		Options: map[string]string{"prefix": "maestro-edge"},
	})
	require.NoError(t, err)

	ctx := context.Background()
	require.NoError(t, store.Set(ctx, "multi", "k1", "v1"))
	require.NoError(t, store.Set(ctx, "multi", "k2", "v2"))
	require.NoError(t, store.Set(ctx, "multi", "k3", "v3"))

	for _, tc := range []struct{ key, expect string }{
		{"k1", "v1"}, {"k2", "v2"}, {"k3", "v3"},
	} {
		val, err := store.Get(ctx, "multi", tc.key)
		require.NoError(t, err)
		assert.Equal(t, tc.expect, val)
	}
}

// ── Helpers ──────────────────────────────────────────────────────

func prepopulatedFileStore(t *testing.T, data map[string]map[string]string) (string, credentials.CredentialStore) {
	t.Helper()
	dir := t.TempDir()
	b, err := json.MarshalIndent(data, "", "  ")
	require.NoError(t, err)
	err = os.WriteFile(filepath.Join(dir, "credentials.json"), b, 0600)
	require.NoError(t, err)
	s, err := credentials.NewStore(credentials.Config{
		Options: map[string]string{"dir": dir},
	})
	require.NoError(t, err)
	return dir, s
}
