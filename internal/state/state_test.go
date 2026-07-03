package state

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestPathForLinuxUsesXDGStateHome(t *testing.T) {
	got, err := pathFor("linux", "/home/alice", "/tmp/state-home", "")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/tmp/state-home", appDirName, "state.json")
	if got != want {
		t.Fatalf("pathFor() = %q, want %q", got, want)
	}
}

func TestPathForLinuxFallsBackToLocalState(t *testing.T) {
	got, err := pathFor("linux", "/home/alice", "", "")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/home/alice", ".local", "state", appDirName, "state.json")
	if got != want {
		t.Fatalf("pathFor() = %q, want %q", got, want)
	}
}

func TestPathForDarwinUsesApplicationSupport(t *testing.T) {
	got, err := pathFor("darwin", "/Users/alice", "", "")
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join("/Users/alice", "Library", "Application Support", appDirName, "state.json")
	if got != want {
		t.Fatalf("pathFor() = %q, want %q", got, want)
	}
}

func TestPathForWindowsUsesLocalAppData(t *testing.T) {
	got, err := pathFor("windows", "", "", `C:\Users\alice\AppData\Local`)
	if err != nil {
		t.Fatal(err)
	}
	want := filepath.Join(`C:\Users\alice\AppData\Local`, appDirName, "state.json")
	if got != want {
		t.Fatalf("pathFor() = %q, want %q", got, want)
	}
}

func TestLoadMigratesLegacyConfigToState(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	root := t.TempDir()
	home := filepath.Join(root, "home")
	configDir := filepath.Join(root, "config")
	legacyPath := filepath.Join(configDir, appDirName, "config.json")
	newPath := filepath.Join(home, ".local", "state", appDirName, "state.json")

	setUserHomeDir(t, home)
	t.Setenv("XDG_CONFIG_HOME", configDir)

	legacy := State{
		AccessToken:   "access",
		RefreshToken:  "refresh",
		TokenExpiry:   time.Date(2026, 7, 3, 12, 0, 0, 0, time.UTC),
		LastContextID: 42,
	}
	if err := saveLegacyConfig(legacyPath, legacy); err != nil {
		t.Fatal(err)
	}

	got, err := Load()
	if err != nil {
		t.Fatal(err)
	}
	if got != legacy {
		t.Fatalf("Load() = %#v, want %#v", got, legacy)
	}

	migrated, err := loadFromPath(newPath)
	if err != nil {
		t.Fatal(err)
	}
	if migrated != legacy {
		t.Fatalf("migrated state = %#v, want %#v", migrated, legacy)
	}
}

func TestLoadReturnsErrorForInvalidLegacyConfig(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	root := t.TempDir()
	home := filepath.Join(root, "home")
	configDir := filepath.Join(root, "config")
	legacyPath := filepath.Join(configDir, appDirName, "config.json")

	setUserHomeDir(t, home)
	t.Setenv("XDG_CONFIG_HOME", configDir)

	if err := os.MkdirAll(filepath.Dir(legacyPath), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(legacyPath, []byte("not-json"), 0o600); err != nil {
		t.Fatal(err)
	}

	if _, err := Load(); err == nil {
		t.Fatal("Load() error = nil, want invalid legacy JSON error")
	}
}

func saveLegacyConfig(path string, st State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data := []byte("{\n  \"access_token\": \"" + st.AccessToken + "\",\n  \"refresh_token\": \"" + st.RefreshToken + "\",\n  \"token_expiry\": \"" + st.TokenExpiry.Format(time.RFC3339Nano) + "\",\n  \"last_context_id\": 42\n}\n")
	return os.WriteFile(path, data, 0o600)
}

func setUserHomeDir(t *testing.T, home string) {
	t.Helper()
	if err := os.MkdirAll(home, 0o700); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	if err := os.Unsetenv("USERPROFILE"); err != nil {
		t.Fatal(err)
	}
}
