package cmd

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v2"
)

const (
	defaultPath string = ".yrs/config.yml"
)

type Config struct {
	DatabaseUrl string `yaml:"database_url"`
}

func loadConfig() (*Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return nil, err
	}

	tries := 0
OPEN:
	f, err := os.Open(filepath.Join(home, defaultPath))
	if err != nil {
		if errors.Is(err, fs.ErrNotExist) && tries == 0 {
			if err := initializeConfig(); err != nil {
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

func initializeConfig() error {
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	c := Config{DatabaseUrl: fmt.Sprintf("file:%s/.yrs/yrs.db", home)}
	f, err := os.OpenFile(
		filepath.Join(home, defaultPath),
		os.O_CREATE|os.O_RDWR,
		0644,
	)
	if err != nil {
		return err
	}

	if err := yaml.NewEncoder(f).Encode(c); err != nil {
		return err
	}

	return nil
}
