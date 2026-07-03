package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadReturnsDefaultsWhenConfigMissing(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	setConfigHome(t, home, filepath.Join(t.TempDir(), "config"))

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MYN.CriticalNowMaxTasks != 5 {
		t.Fatalf("CriticalNowMaxTasks = %d, want 5", cfg.MYN.CriticalNowMaxTasks)
	}
	if cfg.MYN.OpportunityNowMaxTasks != 20 {
		t.Fatalf("OpportunityNowMaxTasks = %d, want 20", cfg.MYN.OpportunityNowMaxTasks)
	}
}

func TestLoadReadsTOMLAndAppliesDefaultsForMissingValues(t *testing.T) {
	root := t.TempDir()
	home := filepath.Join(root, "home")
	configHome := filepath.Join(root, "config")
	setConfigHome(t, home, configHome)

	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	data := []byte("[myn]\nopportunity_now_max_tasks = 12\n")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if cfg.MYN.CriticalNowMaxTasks != 5 {
		t.Fatalf("CriticalNowMaxTasks = %d, want 5", cfg.MYN.CriticalNowMaxTasks)
	}
	if cfg.MYN.OpportunityNowMaxTasks != 12 {
		t.Fatalf("OpportunityNowMaxTasks = %d, want 12", cfg.MYN.OpportunityNowMaxTasks)
	}
}

func setConfigHome(t *testing.T, home, configHome string) {
	t.Helper()
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(configHome, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)
}
