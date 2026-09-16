package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadDefaultsWhenFileMissing(t *testing.T) {
	dir := t.TempDir()

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 8080 {
		t.Errorf("Port = %d, want 8080", cfg.Port)
	}
	if cfg.DBPath != filepath.Join(dir, "johansenfoo.db") {
		t.Errorf("DBPath = %q, want %q", cfg.DBPath, filepath.Join(dir, "johansenfoo.db"))
	}
	if cfg.BaseURL != "https://johansen.foo" {
		t.Errorf("BaseURL = %q, want https://johansen.foo", cfg.BaseURL)
	}
}

func TestLoadOverridesFromFile(t *testing.T) {
	dir := t.TempDir()
	body := `{"port": 9000, "db_path": "/tmp/custom.db", "base_url": "http://localhost:9000"}`
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load(dir)
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if cfg.Port != 9000 {
		t.Errorf("Port = %d, want 9000", cfg.Port)
	}
	if cfg.DBPath != "/tmp/custom.db" {
		t.Errorf("DBPath = %q, want /tmp/custom.db", cfg.DBPath)
	}
	if cfg.BaseURL != "http://localhost:9000" {
		t.Errorf("BaseURL = %q, want http://localhost:9000", cfg.BaseURL)
	}
}

func TestLoadRejectsMalformedFile(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "config.json"), []byte("{not json"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(dir); err == nil {
		t.Fatal("Load succeeded on malformed config, want error")
	}
}
