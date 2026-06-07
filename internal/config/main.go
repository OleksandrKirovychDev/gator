package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

const configFileName = ".gatorconfig.json"

type Config struct {
	DBURL 		string `json:"db_url"`
	CurrentUser string `json:"current_user_name"`
}

func (c *Config) SetUser(userName string) error {
	c.CurrentUser = userName
	return write(*c)
}

func Read() (Config, error) {
	path, err := getConfigFilePath()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(path)
	if err != nil {
		return Config{}, fmt.Errorf("failed reading config file: %w", err)
	}

	var config Config

	err = json.Unmarshal(data, &config)
	if err != nil {
		return Config{}, fmt.Errorf("failed parsing config file: %w", err)
	}

	return config, nil
}

func getConfigFilePath() (string, error) {
	homeDir, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("locating user home directory: %w", err)
	}

	path := filepath.Join(homeDir, configFileName)

	return path, nil
}

func write(cfg Config) error {
	path, err := getConfigFilePath()
	if err != nil {
		return err
	}

	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed serializing config: %w", err)
	}

	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".gatorconfig-*.tmp")
	if err != nil {
		return fmt.Errorf("creating temp file: %w", err)
	}
	tmpName := tmp.Name()

	defer os.Remove(tmpName)


	if _, err := tmp.Write(data); err != nil {
		tmp.Close()
		return fmt.Errorf("writing temp file: %w", err)
	}

	if err := tmp.Close(); err != nil {
		return fmt.Errorf("closing temp file: %w", err)
	}

	if err := os.Rename(tmpName, path); err != nil {
		return fmt.Errorf("replacing config file: %w", err)
	}

	return nil
}