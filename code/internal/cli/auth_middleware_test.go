package cli

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/spf13/cobra"
)

func TestIsAuthExempt_AllPaths(t *testing.T) {
	cases := []struct {
		parent string
		name   string
		want   bool
	}{
		// root-level
		{"", "help", true},
		{"", "version", true},
		{"", "completion", true},
		{"", "update", true},
		{"", "__test_ping", true}, // test prefix
		// auth subcommands
		{"auth", "login", true},
		{"auth", "logout", true},
		{"auth", "whoami", true},
		{"auth", "keys", false}, // keys/quota/status are NOT exempt
		{"auth", "quota", false},
		{"auth", "status", false},
		// task subcommands — none exempt
		{"task", "list", false},
		{"task", "create", false},
	}
	for _, c := range cases {
		t.Run(c.parent+"_"+c.name, func(t *testing.T) {
			cmd := &cobra.Command{Use: c.name}
			if c.parent != "" {
				parent := &cobra.Command{Use: c.parent}
				parent.AddCommand(cmd)
				// SetName properly so Name() returns "name" (not empty).
				cmd.Name()
			}
			got := isAuthExempt(cmd)
			if got != c.want {
				t.Errorf("isAuthExempt(%s, %s) = %v, want %v", c.parent, c.name, got, c.want)
			}
		})
	}
}

func TestRequireAuth_LoggedIn_InjectsContext(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	SetBuildMetadata("test", "c1", "now", "http://prod", "http://sit", "prod", "")

	cfgPath := filepath.Join(dir, ".tick", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte("api_keys:\n  prod: tk_cfg_x\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	cmd := &cobra.Command{Use: "list"}
	if err := RequireAuth(cmd, nil); err != nil {
		t.Fatalf("RequireAuth returned err: %v", err)
	}
	got := APIKeyFromContext(cmd.Context())
	if got != "tk_cfg_x" {
		t.Errorf("ctx key = %q, want tk_cfg_x", got)
	}
}

func TestRequireAuth_EnvOverridesConfig(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	SetBuildMetadata("test", "c1", "now", "http://prod", "http://sit", "prod", "")

	cfgPath := filepath.Join(dir, ".tick", "config.yaml")
	if err := os.MkdirAll(filepath.Dir(cfgPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(cfgPath, []byte("api_keys:\n  prod: tk_cfg_x\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("TICK_API_KEY", "tk_env_y")

	cmd := &cobra.Command{Use: "list"}
	if err := RequireAuth(cmd, nil); err != nil {
		t.Fatalf("RequireAuth returned err: %v", err)
	}
	if got := APIKeyFromContext(cmd.Context()); got != "tk_env_y" {
		t.Errorf("ctx key = %q, want tk_env_y (env should win)", got)
	}
	// verify config file is untouched
	data, _ := os.ReadFile(cfgPath)
	if !strings.Contains(string(data), "tk_cfg_x") {
		t.Errorf("config file was modified by env var path")
	}
}

func TestRequireAuth_ExemptSkipsResolve(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("HOME", dir)
	SetBuildMetadata("test", "c1", "now", "http://prod", "http://sit", "prod", "")

	// No config file → no key. login is exempt so it must not exit.
	// Build proper parent chain so isAuthExempt's parent check works.
	parent := &cobra.Command{Use: "auth"}
	cmd := &cobra.Command{Use: "login"}
	parent.AddCommand(cmd)
	if err := RequireAuth(cmd, nil); err != nil {
		t.Fatalf("exempt command should not error: %v", err)
	}
	// Context should not have a key (RequireAuth didn't resolve)
	if got := APIKeyFromContext(cmd.Context()); got != "" {
		t.Errorf("exempt cmd should leave ctx without key, got %q", got)
	}
}

func TestAPIKeyFromContext_NilOrEmpty(t *testing.T) {
	if got := APIKeyFromContext(nil); got != "" {
		t.Errorf("nil ctx → %q, want empty", got)
	}
	if got := APIKeyFromContext(context.Background()); got != "" {
		t.Errorf("empty ctx → %q, want empty", got)
	}
}

// TestRequireAuth_NotLoggedIn_Exits2 runs in a subprocess because
// RequireAuth calls os.Exit(2) on failure.
func TestRequireAuth_NotLoggedIn_Exits2(t *testing.T) {
	if os.Getenv("BE_AUTH_EXEMPT_DRIVER") == "1" {
		dir := t.TempDir()
		t.Setenv("HOME", dir)
		t.Setenv("TICK_API_KEY", "")
		SetBuildMetadata("test", "c1", "now", "http://prod", "http://sit", "prod", "")
		_ = os.Unsetenv("TICK_API_KEY")

		cmd := &cobra.Command{Use: "list"}
		_ = RequireAuth(cmd, nil)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestRequireAuth_NotLoggedIn_Exits2")
	cmd.Env = append(os.Environ(), "BE_AUTH_EXEMPT_DRIVER=1")
	err := cmd.Run()
	if err == nil {
		t.Fatalf("expected non-zero exit")
	}
	ee, ok := err.(*exec.ExitError)
	if !ok {
		t.Fatalf("not ExitError: %v", err)
	}
	if ee.ExitCode() != ExitNotLogged {
		t.Errorf("exit = %d, want %d", ee.ExitCode(), ExitNotLogged)
	}
}

func TestRequireAuth_SilenceUsageOnFailure(t *testing.T) {
	if os.Getenv("BE_AUTH_SILENCE_DRIVER") == "1" {
		dir := t.TempDir()
		t.Setenv("HOME", dir)
		_ = os.Unsetenv("TICK_API_KEY")
		SetBuildMetadata("test", "c1", "now", "http://prod", "http://sit", "prod", "")

		cmd := &cobra.Command{Use: "list"}
		_ = RequireAuth(cmd, nil)
		return
	}
	cmd := exec.Command(os.Args[0], "-test.run=TestRequireAuth_SilenceUsageOnFailure")
	cmd.Env = append(os.Environ(), "BE_AUTH_SILENCE_DRIVER=1")
	out, _ := cmd.CombinedOutput()
	if strings.Contains(string(out), "Usage:") {
		t.Errorf("stderr should not include usage banner, got: %s", out)
	}
	if !strings.Contains(string(out), "未登录") {
		t.Errorf("stderr should include 未登录, got: %s", out)
	}
}
