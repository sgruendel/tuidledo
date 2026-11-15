package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/BurntSushi/toml"
)

const appDirName = "tuidledo"

const (
	defaultCriticalNowMaxTasks    = 5
	defaultOpportunityNowMaxTasks = 20
)

// Config contains user-configurable application settings.
type Config struct {
	MYN MYNConfig `toml:"myn"`
	UI  UIConfig  `toml:"ui"`
}

// UIConfig contains general display settings.
type UIConfig struct {
	// WeekStart is the first day of the week in date pickers. Defaults to Sunday.
	WeekStart Weekday `toml:"week_start"`
}

// Weekday is a day of the week, written in config.toml by its English name.
// The zero value is Sunday.
type Weekday time.Weekday

// UnmarshalText parses an English weekday name such as "monday", ignoring case.
func (d *Weekday) UnmarshalText(text []byte) error {
	name := strings.TrimSpace(string(text))
	for day := time.Sunday; day <= time.Saturday; day++ {
		if strings.EqualFold(name, day.String()) {
			*d = Weekday(day)
			return nil
		}
	}
	return fmt.Errorf("invalid weekday %q: use a day name such as \"sunday\" or \"monday\"", name)
}

// MYNConfig contains settings for Michael Linenberger-style task views.
type MYNConfig struct {
	CriticalNowMaxTasks    int    `toml:"critical_now_max_tasks"`
	OpportunityNowMaxTasks int    `toml:"opportunity_now_max_tasks"`
	TopLabel               string `toml:"top_label"`
	HighLabel              string `toml:"high_label"`
	MedLabel               string `toml:"med_label"`
	LowLabel               string `toml:"low_label"`
}

// Load reads the application configuration, returning defaults for missing files and values.
func Load() (Config, error) {
	cfg := Default()
	path, err := Path()
	if err != nil {
		return cfg, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			return cfg, nil
		}
		return cfg, err
	}

	if err := toml.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	return cfg.withDefaults(), nil
}

// Default returns the default application configuration.
func Default() Config {
	return Config{
		MYN: MYNConfig{
			CriticalNowMaxTasks:    defaultCriticalNowMaxTasks,
			OpportunityNowMaxTasks: defaultOpportunityNowMaxTasks,
			TopLabel:               "Top",
			HighLabel:              "High",
			MedLabel:               "Med",
			LowLabel:               "Low",
		},
	}
}

// Path returns the application configuration file path.
func Path() (string, error) {
	dir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(dir, appDirName, "config.toml"), nil
}

func (cfg Config) withDefaults() Config {
	if cfg.MYN.CriticalNowMaxTasks <= 0 {
		cfg.MYN.CriticalNowMaxTasks = defaultCriticalNowMaxTasks
	}
	if cfg.MYN.OpportunityNowMaxTasks <= 0 {
		cfg.MYN.OpportunityNowMaxTasks = defaultOpportunityNowMaxTasks
	}
	if cfg.MYN.TopLabel == "" {
		cfg.MYN.TopLabel = "Top"
	}
	if cfg.MYN.HighLabel == "" {
		cfg.MYN.HighLabel = "High"
	}
	if cfg.MYN.MedLabel == "" {
		cfg.MYN.MedLabel = "Med"
	}
	if cfg.MYN.LowLabel == "" {
		cfg.MYN.LowLabel = "Low"
	}
	return cfg
}
