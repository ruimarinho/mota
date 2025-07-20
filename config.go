package main

import (
	"errors"
	"fmt"
	"gopkg.in/yaml.v2"
	"os"
)

type MotaUserConfig struct {
	GlobalConfig GlobalConfig `yaml:"global,omitempty"`
}

type GlobalConfig struct {
	DefaultCredentials DefaultCredentials `yaml:"credentials,omitempty"`
}

type DefaultCredentials struct {
	Username string `yaml:"username,omitempty"`
	Password string `yaml:"password,omitempty"`
}

func UserConfigPath() (string, error) {
	userHome, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	return fmt.Sprintf("%s/.mota.yml", userHome), nil
}

func LoadUserConfig(path string) (*MotaUserConfig, error) {
	if _, err := os.Stat(path); errors.Is(err, os.ErrNotExist) {
		// no config present
		return nil, nil
	}

	data, readErr := os.ReadFile(path)
	if readErr != nil {
		return nil, readErr
	}

	config := MotaUserConfig{}
	unmarshalErr := yaml.Unmarshal(data, &config)
	if unmarshalErr != nil {
		return nil, unmarshalErr
	}

	return &config, nil
}
