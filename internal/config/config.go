package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	Port    int    `json:"port"`
	DBPath  string `json:"db_path"`
	BaseURL string `json:"base_url"`
}

func Load(dir string) (*Config, error) {
	cfg := &Config{
		Port:    8080,
		DBPath:  filepath.Join(dir, "johansenfoo.db"),
		BaseURL: "https://johansen.foo",
	}

	path := filepath.Join(dir, "config.json")
	raw, err := os.ReadFile(path)
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return nil, err
	}

	var onDisk Config
	if err := json.Unmarshal(raw, &onDisk); err != nil {
		return nil, err
	}
	if onDisk.Port != 0 {
		cfg.Port = onDisk.Port
	}
	if onDisk.DBPath != "" {
		cfg.DBPath = onDisk.DBPath
	}
	if onDisk.BaseURL != "" {
		cfg.BaseURL = onDisk.BaseURL
	}
	return cfg, nil
}
