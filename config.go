package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

type Rule struct {
	ID           string `json:"id"`
	Name         string `json:"name"`
	MatchPattern string `json:"match_pattern"`
	TargetURL    string `json:"target_url"`
	Enabled      bool   `json:"enabled"`
	CreatedAt    string `json:"created_at"`
}

type Config struct {
	ProxyPort string `json:"proxy_port"`
	WebPort   string `json:"web_port"`
	Rules     []Rule `json:"rules"`
}

func defaultConfig() *Config {
	return &Config{
		ProxyPort: "8080",
		WebPort:   "9090",
		Rules: []Rule{
			{
				ID:           generateID(),
				Name:         "example-api",
				MatchPattern: "http://api.example.com:8080/v1/*",
				TargetURL:    "http://127.0.0.1:3000/v1/*",
				Enabled:      false,
				CreatedAt:    time.Now().Format(time.RFC3339),
			},
		},
	}
}

func configPath() string {
	exe, _ := os.Executable()
	dir := filepath.Dir(exe)
	return filepath.Join(dir, "data", "rules.json")
}

func loadConfig() (*Config, error) {
	path := configPath()
	data, err := os.ReadFile(path)
	if err != nil {
		if os.IsNotExist(err) {
			cfg := defaultConfig()
			if err := saveConfig(cfg); err != nil {
				return nil, fmt.Errorf("create default config: %w", err)
			}
			return cfg, nil
		}
		return nil, fmt.Errorf("read config: %w", err)
	}
	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}
	return &cfg, nil
}

func saveConfig(cfg *Config) error {
	path := configPath()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create data dir: %w", err)
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal config: %w", err)
	}
	return os.WriteFile(path, data, 0644)
}
