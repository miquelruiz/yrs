package lib

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

const (
	defaultPath   string = ".yrs"
	defaultName   string = "config.yml"
	defaultDbName string = "yrs.db"
)

type Config struct {
	DatabaseDriver string `yaml:"database_driver"`
	DatabaseUrl    string `yaml:"database_url"`
}

func LoadConfig(configPath string) (*Config, error) {
	mayInitConfig := false
	if configPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return nil, err
		}
		configPath = filepath.Join(home, defaultPath, defaultName)
		mayInitConfig = true
	}

	tries := 0

OPEN:
	f, err := os.Open(configPath)
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) && tries == 0 && mayInitConfig {
			if err := initializeConfig(configPath); err != nil {
				return nil, err
			}
			tries++
			goto OPEN
		} else {
			return nil, err
		}
	}

	var config Config
	if err = yaml.NewDecoder(f).Decode(&config); err != nil {
		return nil, err
	}

	return &config, nil
}

func initializeConfig(configPath string) error {
	fmt.Printf("Initializing config in %s\n", configPath)
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}
	dbDir := filepath.Join(home, defaultPath)
	_, err = os.Stat(dbDir)
	if err != nil {
		if err := os.Mkdir(dbDir, 0750); err != nil {
			return err
		}
	}
	fullDbPath := filepath.Join(dbDir, defaultDbName)
	c := Config{
		DatabaseDriver: "sqlite3",
		DatabaseUrl:    fmt.Sprintf("file:%s", fullDbPath),
	}
	f, err := os.OpenFile(configPath, os.O_CREATE|os.O_RDWR, 0644)
	if err != nil {
		return err
	}

	if err := yaml.NewEncoder(f).Encode(c); err != nil {
		return err
	}

	return err
}
