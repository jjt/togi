package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConfigPathPrecedence(t *testing.T) {
	t.Setenv("TOGI_CONFIG", "/tmp/explicit.toml")
	t.Setenv("XDG_CONFIG_HOME", "/tmp/xdg")
	if got := ConfigPath(); got != "/tmp/explicit.toml" {
		t.Fatalf("TOGI_CONFIG should win, got %q", got)
	}

	t.Setenv("TOGI_CONFIG", "")
	want := filepath.Join("/tmp/xdg", "togi", "config.toml")
	if got := ConfigPath(); got != want {
		t.Fatalf("XDG path: got %q, want %q", got, want)
	}
}

func TestLoadConfigMissing(t *testing.T) {
	c, found, err := LoadConfig(filepath.Join(t.TempDir(), "nope.toml"))
	if err != nil {
		t.Fatalf("missing file should not error: %v", err)
	}
	if found {
		t.Fatalf("found should be false for missing file")
	}
	if c == nil {
		t.Fatalf("config should be non-nil")
	}
}

func TestLoadConfigParses(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "config.toml")
	body := `
[replacements]
javascript = "JavaScript"
"vs code" = "VS Code"

[surrounds.parens]
start = "parent"
end = "unparent"
open = "("
close = ")"

[surrounds.quotes]
start = "quote"
end = "end quote"
open = '"'
close = '"'
`
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	c, found, err := LoadConfig(path)
	if err != nil {
		t.Fatal(err)
	}
	if !found {
		t.Fatal("expected found=true")
	}
	if c.Replacements["javascript"] != "JavaScript" {
		t.Errorf("missing javascript replacement: %+v", c.Replacements)
	}
	if c.Replacements["vs code"] != "VS Code" {
		t.Errorf("missing vs code replacement: %+v", c.Replacements)
	}
	if len(c.Surrounds) != 2 {
		t.Fatalf("want 2 surrounds, got %d", len(c.Surrounds))
	}
	if c.Surrounds["parens"].Open != "(" || c.Surrounds["quotes"].Open != "\"" {
		t.Errorf("surrounds parsed wrong: %+v", c.Surrounds)
	}
}
