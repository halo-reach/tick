package cli

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// Build-time metadata (set by main.go via SetBuildMetadata, FR-032/FR-034).
var (
	buildVersion    = "dev"
	buildCommit     = "unknown"
	buildTime       = "unknown"
	buildProdURL    = "http://localhost:8080"
	buildSITURL     = "http://localhost:8080"
	buildEnv        = "prod"
	buildSourcePath = ""
)

// SetBuildMetadata is called from main.go to inject ldflags-injected constants.
func SetBuildMetadata(version, commit, bt, prodURL, sitURL, env, sourcePath string) {
	buildVersion = version
	buildCommit = commit
	buildTime = bt
	buildProdURL = prodURL
	buildSITURL = sitURL
	buildEnv = env
	buildSourcePath = sourcePath
}

// ServerURL returns the compile-time server URL for the current binary's env.
func ServerURL() string {
	if buildEnv == "sit" {
		return buildSITURL
	}
	return buildProdURL
}

// BuiltForEnv returns the env tag compiled into the current binary.
func BuiltForEnv() string { return buildEnv }

// VersionString returns the formatted version output for `tick --version` (FR-033).
func VersionString() string {
	return fmt.Sprintf("tick version %s (commit %s, built %s, env %s)", buildVersion, buildCommit, buildTime, buildEnv)
}

// ConfigPath returns the resolved config file path, honoring --config override.
func ConfigPath() (string, error) {
	if cfgFile != "" {
		return cfgFile, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, ".tick", "config.yaml"), nil
}

// configDir returns the directory containing the config file.
func configDir() (string, error) {
	p, err := ConfigPath()
	if err != nil {
		return "", err
	}
	return filepath.Dir(p), nil
}

// writeConfigAtomic writes data to the config file via tmp + os.Rename and
// sets permissions to 0o600 (FR-013, SC-006).
func writeConfigAtomic(data []byte) error {
	finalPath, err := ConfigPath()
	if err != nil {
		return err
	}
	dir := filepath.Dir(finalPath)
	if err := os.MkdirAll(dir, 0o700); err != nil {
		return err
	}
	tmpPath := finalPath + ".tmp"
	if err := os.WriteFile(tmpPath, data, 0o600); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, finalPath); err != nil {
		_ = os.Remove(tmpPath)
		return err
	}
	return setConfigPermissions(finalPath)
}

// setConfigPermissions forces 0o600 on the config file (FR-013, SC-006).
func setConfigPermissions(path string) error {
	return os.Chmod(path, 0o600)
}

// UserConfig is the on-disk shape of ~/.tick/config.yaml (per data-model.md).
type UserConfig struct {
	APIKeys map[string]string `yaml:"api_keys"`
	Output  string            `yaml:"output,omitempty"`
}

// LoadConfig reads the user config. Migrates old fields on read (FR-014, SC-004).
// Returns an empty config if file does not exist.
func LoadConfig() (*UserConfig, error) {
	path, err := ConfigPath()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return &UserConfig{APIKeys: map[string]string{}}, nil
		}
		return nil, err
	}
	raw := map[string]any{}
	if err := yaml.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("配置文件损坏（YAML 解析失败），请删除 %s 重新 tick auth login: %w", path, err)
	}
	migrated, didMigrate := migrateLegacyFields(raw)
	cfg := &UserConfig{APIKeys: map[string]string{}}
	if v, ok := migrated["api_keys"].(map[string]any); ok {
		for k, val := range v {
			if s, ok := val.(string); ok {
				cfg.APIKeys[k] = s
			}
		}
	}
	if cfg.APIKeys == nil {
		cfg.APIKeys = map[string]string{}
	}
	if v, ok := migrated["output"].(string); ok {
		cfg.Output = v
	}
	if didMigrate {
		out, _ := yaml.Marshal(migrated)
		if err := writeConfigAtomic(out); err != nil {
			return cfg, nil // fall through with in-memory migrated values
		}
	}
	return cfg, nil
}

// SaveConfig persists the user config atomically.
func SaveConfig(cfg *UserConfig) error {
	if cfg.APIKeys == nil {
		cfg.APIKeys = map[string]string{}
	}
	out, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return writeConfigAtomic(out)
}

// migrateLegacyFields migrates old (server_url, token, current_env) fields to
// the new (api_keys) shape (FR-014). Returns the migrated map and a bool
// indicating whether any migration occurred (caller re-writes if true).
func migrateLegacyFields(raw map[string]any) (map[string]any, bool) {
	didMigrate := false
	if _, hasOld := raw["token"]; hasOld {
		didMigrate = true
		apiKeys, _ := raw["api_keys"].(map[string]any)
		if apiKeys == nil {
			apiKeys = map[string]any{}
		}
		if _, exists := apiKeys["default"]; !exists {
			if t, ok := raw["token"].(string); ok && t != "" {
				apiKeys["default"] = t
			}
		}
		raw["api_keys"] = apiKeys
		delete(raw, "token")
	}
	if _, has := raw["server_url"]; has {
		didMigrate = true
		delete(raw, "server_url")
	}
	if _, has := raw["current_env"]; has {
		didMigrate = true
		delete(raw, "current_env")
	}
	return raw, didMigrate
}

// ResolveAPIKey returns the active API key for BuiltForEnv. Resolution order:
//   1. TICK_API_KEY env var (NOT written back to config, FR-015)
//   2. api_keys[BuiltForEnv] from ~/.tick/config.yaml
func ResolveAPIKey(cfg *UserConfig) string {
	if v := strings.TrimSpace(os.Getenv("TICK_API_KEY")); v != "" {
		return v
	}
	if cfg == nil {
		return ""
	}
	return cfg.APIKeys[BuiltForEnv()]
}
