package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestSaveAndLoadAPIKey(t *testing.T) {
	home := t.TempDir()
	lookup := testLookup(map[string]string{"HOME": home})

	path, err := SaveAPIKey(" exa-test-key ", lookup)
	if err != nil {
		t.Fatalf("SaveAPIKey() error = %v", err)
	}
	if path != filepath.Join(home, ".exa-cli", "config.json") {
		t.Fatalf("path = %q", path)
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat config: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("config mode = %o, want 0600", got)
	}

	source, err := LoadAPIKey(lookup)
	if err != nil {
		t.Fatalf("LoadAPIKey() error = %v", err)
	}
	if source.Key != "exa-test-key" {
		t.Fatalf("key = %q", source.Key)
	}
	if source.Source != path {
		t.Fatalf("source = %q, want %q", source.Source, path)
	}
}

func TestLoadAPIKeyPrefersEnvironment(t *testing.T) {
	home := t.TempDir()
	lookupConfig := testLookup(map[string]string{"HOME": home})
	if _, err := SaveAPIKey("file-key", lookupConfig); err != nil {
		t.Fatalf("SaveAPIKey() error = %v", err)
	}

	source, err := LoadAPIKey(testLookup(map[string]string{
		"HOME":        home,
		EnvAPIKey:     "env-key",
		EnvConfigPath: filepath.Join(home, ".exa-cli", "config.json"),
	}))
	if err != nil {
		t.Fatalf("LoadAPIKey() error = %v", err)
	}
	if source.Key != "env-key" {
		t.Fatalf("key = %q, want env-key", source.Key)
	}
	if source.Source != EnvAPIKey {
		t.Fatalf("source = %q, want %q", source.Source, EnvAPIKey)
	}
}

func TestLogoutRemovesConfig(t *testing.T) {
	home := t.TempDir()
	lookup := testLookup(map[string]string{"HOME": home})
	path, err := SaveAPIKey("exa-test-key", lookup)
	if err != nil {
		t.Fatalf("SaveAPIKey() error = %v", err)
	}
	if _, err := Logout(lookup); err != nil {
		t.Fatalf("Logout() error = %v", err)
	}
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatalf("config still exists or unexpected stat error: %v", err)
	}
}

func TestLoadAPIKeyWithoutHOMEReturnsEmptySource(t *testing.T) {
	source, err := LoadAPIKey(testLookup(map[string]string{}))
	if err != nil {
		t.Fatalf("LoadAPIKey() error = %v", err)
	}
	if source.Key != "" {
		t.Fatalf("key = %q, want empty", source.Key)
	}
	if source.Source != "" {
		t.Fatalf("source = %q, want empty", source.Source)
	}
}

func testLookup(values map[string]string) LookupEnv {
	return func(key string) (string, bool) {
		value, ok := values[key]
		return value, ok
	}
}
