package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	if got := time.Weekday(cfg.UI.WeekStart); got != time.Sunday {
		t.Fatalf("WeekStart = %v, want Sunday", got)
	}
}

func TestLoadReadsWeekStart(t *testing.T) {
	writeConfig(t, "[ui]\nweek_start = \"Monday\"\n")

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got := time.Weekday(cfg.UI.WeekStart); got != time.Monday {
		t.Fatalf("WeekStart = %v, want Monday", got)
	}
}

func TestLoadRejectsInvalidWeekStart(t *testing.T) {
	writeConfig(t, "[ui]\nweek_start = \"mon\"\n")

	_, err := Load()
	if err == nil || !strings.Contains(err.Error(), `invalid weekday "mon"`) {
		t.Fatalf("Load() error = %v, want invalid weekday error", err)
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
	data := []byte("[myn]\nopportunity_now_max_tasks = 12\nhigh_label = \"Critical Now\"\nmed_label = \"Opportunity Now\"\n")
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
	if cfg.MYN.HighLabel != "Critical Now" {
		t.Fatalf("HighLabel = %q, want Critical Now", cfg.MYN.HighLabel)
	}
	if cfg.MYN.MedLabel != "Opportunity Now" {
		t.Fatalf("MedLabel = %q, want Opportunity Now", cfg.MYN.MedLabel)
	}
	if cfg.MYN.TopLabel != "Top" {
		t.Fatalf("TopLabel = %q, want Top", cfg.MYN.TopLabel)
	}
}

func writeConfig(t *testing.T, content string) {
	t.Helper()
	root := t.TempDir()
	setConfigHome(t, filepath.Join(root, "home"), filepath.Join(root, "config"))
	path, err := Path()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
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
