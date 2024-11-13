package config

import (
	"gopkg.in/yaml.v3"
	"os"
)

type Configuration struct {
	Storage StorageConfig `yaml:"storage"`
	Server  ServerConfig  `yaml:"server"`
}

type StorageConfig struct {
	Path string `yaml:"path"`
}

type ServerConfig struct {
	Port int `yaml:"port"`
}

func LoadConfiguration(configurationFilePath string) (*Configuration, error) {
	data, err := os.ReadFile(configurationFilePath)
	if err != nil {
		return nil, err
	}
	var config Configuration
	err = yaml.Unmarshal(data, &config)
	if err != nil {
		return nil, err
	}
	return &config, nil
}
