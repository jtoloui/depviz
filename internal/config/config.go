package config

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"gopkg.in/yaml.v3"
)

// Category represents an import classification.
type Category string

const (
	Stdlib   Category = "stdlib"
	Internal Category = "internal"
	Private  Category = "private"
	External Category = "external"
)

// ClassifyRules holds pattern lists for import classification.
type ClassifyRules struct {
	Internal []string `yaml:"internal"`
	Private  []string `yaml:"private"`
}

// Config represents a .depviz.yml configuration.
type Config struct {
	Language string        `yaml:"language"`
	Port     int           `yaml:"port,omitempty"`
	Output   string        `yaml:"output,omitempty"`
	Exclude  []string      `yaml:"exclude"`
	Classify ClassifyRules `yaml:"classify"`
}

var supportedLangs = map[string]bool{"go": true, "js": true, "multi": true}

// Load reads .depviz.yml from root. If the file doesn't exist,
// it returns DefaultFor(lang). Always returns a valid config or an error.
func Load(root, lang string) (*Config, error) {
	path := filepath.Join(root, ".depviz.yml")

	data, err := os.ReadFile(path)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return DefaultFor(lang, root)
		}
		return nil, fmt.Errorf("reading config: %w", err)
	}

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("parsing config: %w", err)
	}

	if cfg.Language == "" {
		cfg.Language = lang
	}

	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &cfg, nil
}

func (c *Config) validate() error {
	if !supportedLangs[c.Language] {
		return fmt.Errorf("unsupported language: %q", c.Language)
	}

	for _, p := range c.Classify.Internal {
		if _, err := regexp.Compile(p); err != nil {
			return fmt.Errorf("invalid internal pattern %q: %w", p, err)
		}
	}

	for _, p := range c.Classify.Private {
		if _, err := regexp.Compile(p); err != nil {
			return fmt.Errorf("invalid private pattern %q: %w", p, err)
		}
	}

	for _, e := range c.Exclude {
		if e == "" {
			return errors.New("exclude pattern must not be empty")
		}
	}

	return nil
}
