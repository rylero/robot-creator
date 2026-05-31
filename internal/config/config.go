package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const FileName = "robot-creator.yaml"

type Config struct {
	Team           int         `yaml:"team"`
	Package        string      `yaml:"package"`
	Subsystems     []Subsystem `yaml:"subsystems,omitempty"`
	Superstructure bool        `yaml:"superstructure,omitempty"`
}

type Subsystem struct {
	Name    string `yaml:"name"`
	Type    string `yaml:"type"`
	Motors  int    `yaml:"motors,omitempty"`
	Aligned bool   `yaml:"aligned,omitempty"`
}

func Load(root string) (*Config, error) {
	data, err := os.ReadFile(filepath.Join(root, FileName))
	if err != nil {
		return nil, fmt.Errorf("not a robot-creator project (robot-creator.yaml not found): run 'robot-creator init' first")
	}
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("malformed robot-creator.yaml: %w", err)
	}
	return &cfg, nil
}

func Save(root string, cfg *Config) error {
	data, err := yaml.Marshal(cfg)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(root, FileName), data, 0644)
}

func FindRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, FileName)); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("not in a robot-creator project. Run 'robot-creator init' first")
		}
		dir = parent
	}
}

func (c *Config) HasSubsystem(name string) bool {
	for _, s := range c.Subsystems {
		if s.Name == name {
			return true
		}
	}
	return false
}
