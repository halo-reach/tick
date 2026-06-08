package cli

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteConfigAtomic_Permissions(t *testing.T) {
	dir := t.TempDir()
	SetBuildMetadata("test", "deadbeef", "now", "http://prod", "http://sit", "prod", "")
	t.Setenv("HOME", dir)

	path, err := ConfigPath()
	if err != nil {
		t.Fatalf("ConfigPath: %v", err)
	}
	if err := writeConfigAtomic([]byte("api_keys: {prod: tk_x}\n")); err != nil {
		t.Fatalf("writeConfigAtomic: %v", err)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	perm := info.Mode().Perm()
	if perm != 0o600 {
		t.Fatalf("permission = %o, want 0o600", perm)
	}
	if _, err := os.Stat(path + ".tmp"); err == nil {
		t.Fatalf("tmp file leftover")
	}
}

func TestMigrateLegacyFields(t *testing.T) {
	t.Run("server_url token current_env all migrated", func(t *testing.T) {
		raw := map[string]any{
			"server_url":  "https://old.example",
			"token":       "tk_legacy",
			"current_env": "prod",
		}
		out, did := migrateLegacyFields(raw)
		if !did {
			t.Fatalf("expected migration to occur")
		}
		if _, ok := out["server_url"]; ok {
			t.Fatalf("server_url should be dropped")
		}
		if _, ok := out["token"]; ok {
			t.Fatalf("token should be dropped")
		}
		if _, ok := out["current_env"]; ok {
			t.Fatalf("current_env should be dropped")
		}
		keys, ok := out["api_keys"].(map[string]any)
		if !ok {
			t.Fatalf("api_keys not present")
		}
		if keys["default"] != "tk_legacy" {
			t.Fatalf("expected default=tk_legacy, got %v", keys["default"])
		}
	})
	t.Run("already migrated is no-op", func(t *testing.T) {
		raw := map[string]any{
			"api_keys": map[string]any{"prod": "tk_x"},
			"output":   "json",
		}
		out, did := migrateLegacyFields(raw)
		if did {
			t.Fatalf("did should be false when no legacy fields present")
		}
		keys := out["api_keys"].(map[string]any)
		if keys["prod"] != "tk_x" {
			t.Fatalf("api_keys.prod changed")
		}
		if out["output"] != "json" {
			t.Fatalf("output dropped")
		}
	})
}

func TestLoadConfig_BuiltForEnvMismatchReturnsClearError(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	// Simulate "dev" binary: BuiltForEnv=prod, but config has sit key only.
	SetBuildMetadata("test", "c1", "now", "http://prod", "http://sit", "prod", "")

	cfgPath := filepath.Join(dir, ".tick", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte("api_keys:\n  sit: tk_y\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig: %v", err)
	}
	if cfg.APIKeys[BuiltForEnv()] != "" {
		t.Fatalf("prod slot should be empty, got %q", cfg.APIKeys[BuiltForEnv()])
	}
	// ResolveAPIKey should not return the sit key when BuiltForEnv=prod
	if got := ResolveAPIKey(cfg); got != "" {
		t.Fatalf("expected empty api key for prod binary, got %q", got)
	}
}

func TestTICK_API_KEY_PrecedenceAndNoWriteback(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	SetBuildMetadata("test", "c1", "now", "http://prod", "http://sit", "prod", "")

	cfgPath := filepath.Join(dir, ".tick", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte("api_keys:\n  prod: tk_from_config\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TICK_API_KEY", "tk_from_env")

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if got := ResolveAPIKey(cfg); got != "tk_from_env" {
		t.Fatalf("TICK_API_KEY must take precedence; got %q", got)
	}
	// verify config unchanged on disk
	data, _ := os.ReadFile(cfgPath)
	if !strings.Contains(string(data), "tk_from_config") {
		t.Fatalf("config file was modified by env var read")
	}
	if strings.Contains(string(data), "tk_from_env") {
		t.Fatalf("env var leaked into config file")
	}
}

func TestVersionString_IncludesAllFields(t *testing.T) {
	SetBuildMetadata("v1.2.3", "abc1234", "2026-06-02T10:00:00Z", "http://p", "http://s", "prod", "")
	got := VersionString()
	for _, want := range []string{"v1.2.3", "abc1234", "2026-06-02T10:00:00Z", "prod"} {
		if !strings.Contains(got, want) {
			t.Errorf("VersionString missing %q: %s", want, got)
		}
	}
}

func TestWhoamiOutput_JSONShape(t *testing.T) {
	SetBuildMetadata("v0.1.0", "deadbeef", "2026-06-02T00:00:00Z", "https://prod", "https://sit", "prod", "")
	out := map[string]any{
		"tenant":         "acme-corp",
		"user":           "ops-team",
		"api_key_prefix": "tk_live_xxx...f9d508",
		"server_url":     ServerURL(),
		"built_for_env":  BuiltForEnv(),
		"version":        buildVersion,
		"commit":         buildCommit,
		"build_time":     buildTime,
	}
	b, err := json.Marshal(out)
	if err != nil {
		t.Fatal(err)
	}
	var m map[string]any
	if err := json.Unmarshal(b, &m); err != nil {
		t.Fatal(err)
	}
	for _, k := range []string{"tenant", "api_key_prefix", "server_url", "built_for_env", "version"} {
		if _, ok := m[k]; !ok {
			t.Errorf("whoami json missing key %q", k)
		}
	}
}
