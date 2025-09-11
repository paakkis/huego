package config

import (
	"encoding/json"
	"os"
	"path/filepath"
)

type Config struct {
	BridgeIP string `json:"bridge_ip"`
	Username string `json:"username"`
}

func getConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", err
	}
	appDir := filepath.Join(configDir, "huego")
	if err := os.MkdirAll(appDir, 0755); err != nil {
		return "", err
	}
	return filepath.Join(appDir, "config.json"), nil
}

func SaveConfig(cfg Config) error {
	path, err := getConfigPath()
	if err != nil {
		return err
	}
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	encoder := json.NewEncoder(f)
	encoder.SetIndent("", "  ")
	return encoder.Encode(cfg)
}

func LoadConfig() (Config, error) {
	var cfg Config
	path, err := getConfigPath()
	if err != nil {
		return cfg, err
	}
	f, err := os.Open(path)
	if err != nil {
		return cfg, err
	}
	defer f.Close()
	err = json.NewDecoder(f).Decode(&cfg)
	return cfg, err
}
