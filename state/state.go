package state

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"time"
)

const appDirName = "tuidledo"

// State contains locally persisted authentication and UI state.
type State struct {
	AccessToken   string    `json:"access_token"`
	RefreshToken  string    `json:"refresh_token"`
	TokenExpiry   time.Time `json:"token_expiry"`
	LastContextID int64     `json:"last_context_id"`
}

// Load reads the persisted state, migrating legacy config state when needed.
func Load() (State, error) {
	path, err := Path()
	if err != nil {
		return State{}, err
	}

	st, err := loadFromPath(path)
	if err == nil {
		return st, nil
	}
	if !errors.Is(err, os.ErrNotExist) {
		return State{}, err
	}

	legacyPath, err := legacyPath()
	if err != nil {
		return State{}, err
	}

	st, err = loadFromPath(legacyPath)
	if errors.Is(err, os.ErrNotExist) {
		return State{}, nil
	}
	if err != nil {
		return State{}, err
	}

	if err := saveToPath(path, st); err != nil {
		return State{}, err
	}

	return st, nil

}

// Save writes the persisted state.
func Save(st State) error {
	path, err := Path()
	if err != nil {
		return err
	}
	return saveToPath(path, st)
}

// Path returns the platform-specific state file path.
func Path() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return pathFor(runtime.GOOS, home, os.Getenv("XDG_STATE_HOME"), os.Getenv("LOCALAPPDATA"))
}

func legacyPath() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appDirName, "config.json"), nil
}

func loadFromPath(path string) (State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return State{}, err
	}

	var st State
	if err := json.Unmarshal(data, &st); err != nil {
		return State{}, err
	}
	return st, nil
}

func saveToPath(path string, st State) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}

	data, err := json.MarshalIndent(st, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func pathFor(goos, home, xdgStateHome, localAppData string) (string, error) {
	switch goos {
	case "windows":
		if localAppData == "" {
			return "", fmt.Errorf("LOCALAPPDATA is not set")
		}
		return filepath.Join(localAppData, appDirName, "state.json"), nil
	case "darwin":
		if home == "" {
			return "", fmt.Errorf("home directory is not set")
		}
		return filepath.Join(home, "Library", "Application Support", appDirName, "state.json"), nil
	default:
		base := xdgStateHome
		if base == "" {
			if home == "" {
				return "", fmt.Errorf("home directory is not set")
			}
			base = filepath.Join(home, ".local", "state")
		}
		return filepath.Join(base, appDirName, "state.json"), nil
	}
}
