package config

import (
	"os"
	"path/filepath"

	"github.com/BurntSushi/toml"
)

const appDirName = "tuidledo"

const (
	defaultCriticalNowMaxTasks    = 5
	defaultOpportunityNowMaxTasks = 20
)

type Config struct {
	MYN MYNConfig `toml:"myn"`
}

type MYNConfig struct {
	CriticalNowMaxTasks    int    `toml:"critical_now_max_tasks"`
	OpportunityNowMaxTasks int    `toml:"opportunity_now_max_tasks"`
	TopLabel               string `toml:"top_label"`
	HighLabel              string `toml:"high_label"`
	MedLabel               string `toml:"med_label"`
	LowLabel               string `toml:"low_label"`
}

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
