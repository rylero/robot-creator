package config

import (
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const FileName = "robot-creator.yaml"

// Version marks projects generated for Commands v3. Projects without it were made by
// robot-creator v0.x, which targets Commands v2.
const Version = 3

type Config struct {
	Version      int         `yaml:"version"`
	Team         int         `yaml:"team"`
	Package      string      `yaml:"package"`
	Subsystems   []Subsystem `yaml:"subsystems,omitempty"`
	TemplatesDir string      `yaml:"templates_dir,omitempty"`
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
	if cfg.Version != Version {
		return nil, fmt.Errorf("robot-creator.yaml is for a Commands v2 project (no 'version: %d').\n"+
			"Use the v2 generator: go install github.com/rylero/robot-creator@v0.1.8\n"+
			"or, if this project already uses Commands v3, add 'version: %d' to robot-creator.yaml", Version, Version)
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

// New returns a config for a fresh Commands v3 project.
func New(team int, pkg string) *Config {
	return &Config{Version: Version, Team: team, Package: pkg}
}

func (c *Config) HasSubsystem(name string) bool {
	for _, s := range c.Subsystems {
		if s.Name == name {
			return true
		}
	}
	return false
}
