package cli

import (
	"os"
	"path/filepath"
	"testing"
)

func TestAuthLogout_ClearsOnlyBuiltForEnv(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)

	// Pretend binary is BuiltForEnv=prod
	SetBuildMetadata("test", "c1", "now", "http://prod", "http://sit", "prod", "")

	cfgPath := filepath.Join(dir, ".tick", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte("api_keys:\n  prod: tk_prod_x\n  sit: tk_sit_y\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	delete(cfg.APIKeys, BuiltForEnv())
	if err := SaveConfig(cfg); err != nil {
		t.Fatal(err)
	}

	cfg2, err := LoadConfig()
	if err != nil {
		t.Fatal(err)
	}
	if cfg2.APIKeys["prod"] != "" {
		t.Errorf("prod key should be cleared, got %q", cfg2.APIKeys["prod"])
	}
	if cfg2.APIKeys["sit"] != "tk_sit_y" {
		t.Errorf("sit key should be preserved, got %q", cfg2.APIKeys["sit"])
	}
	// permissions preserved
	info, _ := os.Stat(cfgPath)
	if info.Mode().Perm() != 0o600 {
		t.Errorf("permissions drift: got %o", info.Mode().Perm())
	}
}
