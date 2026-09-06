package config

import (
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"time"

	"gopkg.in/yaml.v3"
)

// Load reads, parses, defaults and structurally validates the config at path.
// Checks that require the plugin registries (is "game" known? does the format
// exist?) are done later by internal/app.
func Load(path string) (*Config, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config %s: %w", path, err)
	}

	var cfg Config
	dec := yaml.NewDecoder(bytes.NewReader(raw))
	dec.KnownFields(true)
	if err := dec.Decode(&cfg); err != nil {
		return nil, fmt.Errorf("parse config %s: %w", path, err)
	}

	cfg.applyDefaults()
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config %s: %w", path, err)
	}
	return &cfg, nil
}

func (c *Config) applyDefaults() {
	if c.LogLevel == "" {
		c.LogLevel = "info"
	}
	if len(c.Translators) == 0 {
		c.Translators = []string{"lingvanex", "google"}
	}
	if c.Lingvanex.Address == "" {
		c.Lingvanex.Address = "127.0.0.1"
	}
	if c.Lingvanex.Port == 0 {
		c.Lingvanex.Port = 8000
	}
	if len(c.Lingvanex.Command) == 0 {
		c.Lingvanex.Command = []string{"py", "./server.py"}
	}
	if c.Lingvanex.Workdir == "" {
		c.Lingvanex.Workdir = "third_party/lingvanex-server"
	}
	if c.Lingvanex.HealthTimeout == 0 {
		// The Python server's ctranslate2 / sentencepiece imports take ~15-20s
		// before the port opens.
		c.Lingvanex.HealthTimeout = Duration(90 * time.Second)
	}
	if c.Lingvanex.StopTimeout == 0 {
		c.Lingvanex.StopTimeout = Duration(10 * time.Second)
	}
}

// Validate checks required fields and internal consistency.
func (c *Config) Validate() error {
	if c.Game == "" {
		return fmt.Errorf("game: is required")
	}
	if c.Source.Folder == "" {
		return fmt.Errorf("source.folder: is required")
	}
	if c.Destination.Folder == "" {
		return fmt.Errorf("destination.folder: is required")
	}
	if c.Language.Source == "" || c.Language.Target == "" {
		return fmt.Errorf("language.source and language.target are required")
	}
	if info, err := os.Stat(c.Source.Folder); err != nil {
		return fmt.Errorf("source.folder %q: %w", c.Source.Folder, err)
	} else if !info.IsDir() {
		return fmt.Errorf("source.folder %q: not a directory", c.Source.Folder)
	}
	if c.Source.Files != "" {
		if _, err := filepath.Match(c.Source.Files, "x"); err != nil {
			return fmt.Errorf("source.files: invalid glob %q: %w", c.Source.Files, err)
		}
	}
	if c.Parasitizing.Enabled && c.Parasitizing.File == "" {
		return fmt.Errorf("parasitizing.file: required when parasitizing.enabled is true")
	}
	if len(c.Translators) == 0 {
		return fmt.Errorf("translators: at least one backend is required")
	}
	return nil
}
