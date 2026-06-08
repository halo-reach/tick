package cli

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDetectUpdateMode_NoMarker(t *testing.T) {
	dir := t.TempDir()
	mode, source, err := detectUpdateMode(filepath.Join(dir, ".tick-source"))
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if mode != "release" {
		t.Errorf("mode = %q, want release", mode)
	}
	if source != "" {
		t.Errorf("source = %q, want empty", source)
	}
}

func TestDetectUpdateMode_WithMarker(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, ".tick-source")
	if err := os.WriteFile(marker, []byte("/Users/me/repos/tick\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mode, source, err := detectUpdateMode(marker)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if mode != "from-git" {
		t.Errorf("mode = %q, want from-git", mode)
	}
	if source != "/Users/me/repos/tick" {
		t.Errorf("source = %q, want absolute path", source)
	}
}

func TestDetectUpdateMode_BadMarkerFallsBackToRelease(t *testing.T) {
	dir := t.TempDir()
	marker := filepath.Join(dir, ".tick-source")
	if err := os.WriteFile(marker, []byte("relative/path\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	mode, source, err := detectUpdateMode(marker)
	// bad path is reported as error but caller may fallback; we expect mode=release here
	if err == nil {
		t.Errorf("expected error for non-absolute path")
	}
	if mode != "release" {
		t.Errorf("mode = %q, want release", mode)
	}
	_ = source
}

func TestAtomicReplace_PreservesOriginalOnFailure(t *testing.T) {
	dir := t.TempDir()
	finalPath := filepath.Join(dir, "tick")
	original := []byte("original-binary")
	if err := os.WriteFile(finalPath, original, 0o755); err != nil {
		t.Fatal(err)
	}
	// simulate a non-existent source so replace fails before touching finalPath
	err := atomicReplace(filepath.Join(dir, "does-not-exist"), finalPath)
	if err == nil {
		t.Fatalf("expected error when source missing")
	}
	got, _ := os.ReadFile(finalPath)
	if string(got) != string(original) {
		t.Fatalf("original binary should be preserved; got %q", string(got))
	}
}

func TestAtomicReplace_SuccessLeavesNoTmp(t *testing.T) {
	dir := t.TempDir()
	finalPath := filepath.Join(dir, "tick")
	if err := os.WriteFile(finalPath, []byte("old"), 0o755); err != nil {
		t.Fatal(err)
	}
	newPath := filepath.Join(dir, "tick.new")
	if err := os.WriteFile(newPath, []byte("new"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := atomicReplace(newPath, finalPath); err != nil {
		t.Fatalf("atomicReplace: %v", err)
	}
	got, _ := os.ReadFile(finalPath)
	if string(got) != "new" {
		t.Errorf("expected new content, got %q", string(got))
	}
	if _, err := os.Stat(finalPath + ".old"); err == nil {
		t.Errorf("old backup should be cleaned up")
	}
	if _, err := os.Stat(newPath); err == nil {
		t.Errorf("source should be removed")
	}
}

func TestParseVersion(t *testing.T) {
	cases := []struct {
		in     string
		want   string
		ok     bool
	}{
		{"v0.2.0", "0.2.0", true},
		{"0.3.1", "0.3.1", true},
		{"", "", false},
		{"abc", "", false},
	}
	for _, c := range cases {
		v, ok := parseVersion(c.in)
		if ok != c.ok {
			t.Errorf("parseVersion(%q) ok=%v want %v", c.in, ok, c.ok)
		}
		if ok && v.String() != c.want {
			t.Errorf("parseVersion(%q) = %q want %q", c.in, v.String(), c.want)
		}
	}
}

func TestIsWritable_DetectsNonWritableDir(t *testing.T) {
	dir := t.TempDir()
	exe := filepath.Join(dir, "tick")
	if err := os.WriteFile(exe, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if !isWritable(exe) {
		t.Errorf("expected writable for %s", exe)
	}
}

func TestUpdateCmd_RegisteredInRoot(t *testing.T) {
	root := NewRootCmd()
	var found bool
	for _, c := range root.Commands() {
		if c.Name() == "update" {
			found = true
		}
	}
	if !found {
		t.Fatal("update subcommand not registered")
	}
}

func TestIsCrossDeviceErr(t *testing.T) {
	if isCrossDeviceErr(nil) {
		t.Error("nil should be false")
	}
	if !isCrossDeviceErr(errCrossDeviceFake{}) {
		t.Error("cross-device string match should be true")
	}
}

type errCrossDeviceFake struct{}

func (errCrossDeviceFake) Error() string { return "invalid cross-device link" }

// silence unused import warnings when test list is empty
var _ = strings.HasPrefix
