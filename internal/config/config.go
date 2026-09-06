// Package config is the typed schema for the YAML file that drives a run, plus
// its loader. It holds data only; semantic checks that need the plugin
// registries live in internal/app.
package config

import (
	"fmt"
	"time"
)

// Config is the whole config file.
type Config struct {
	Game string `yaml:"game"`

	Source      Source      `yaml:"source"`
	Destination Destination `yaml:"destination"`
	Language    Language    `yaml:"language"`

	Delimiter        string `yaml:"delimiter"`
	SkipFirstLine    bool   `yaml:"skipFirstLine"`
	MultiRowReplicas bool   `yaml:"multiRowReplicas"`
	// Concurrency is how many translation batches to run at once (default 1).
	// Keep it <= the server's INTER_THREADS or requests queue past the timeout.
	// It only affects speed, never the output.
	Concurrency int `yaml:"concurrency"`
	// BatchSize is how many strings go to the translator per request
	// (default 64). Smaller = more requests but each finishes sooner.
	BatchSize int `yaml:"batchSize"`

	Parasitizing Parasitizing `yaml:"parasitizing"`
	Translators  []string     `yaml:"translators"`
	Lingvanex    Lingvanex    `yaml:"lingvanex"`

	LogLevel string `yaml:"logLevel"`
}

// Source describes the input tree.
type Source struct {
	Folder string `yaml:"folder"`
	Format string `yaml:"format"` // extract format id; empty => game default
	// Files is an optional shell glob matched against each file's base name
	// (e.g. "dialogue.*"). Empty means every file in the tree is translated.
	Files string `yaml:"files"`
}

// Destination describes the output tree.
type Destination struct {
	Folder string `yaml:"folder"`
	// FolderNameMap renames sub-folders on the way out, e.g. {ru: be}.
	FolderNameMap map[string]string `yaml:"folderNameMap"`
}

// Language is the translation direction (backend-specific codes).
type Language struct {
	Source string `yaml:"source"`
	Target string `yaml:"target"`
}

// Parasitizing seeds translation from existing translation files instead of
// machine-translating from scratch. Requires a game that implements
// game.Parasitizer.
//
//   - CarryOver files are already in the target language. A replica whose id is
//     found there is copied verbatim and never sent to the translator. Use a
//     previous run's output to re-translate only what changed in a new version.
//   - Files are in the source language and are still translated; they supply
//     better source text than the game's own (e.g. a natural human phrasing).
//
// Both lists are consulted in order (CarryOver first). Anything unmatched is
// machine-translated from the game's own source text.
type Parasitizing struct {
	Enabled   bool     `yaml:"enabled"`
	CarryOver []string `yaml:"carryOver"`
	Files     []string `yaml:"files"`
}

// Lingvanex configures both the translator backend and its process supervisor.
type Lingvanex struct {
	Manage         bool     `yaml:"manage"`
	Address        string   `yaml:"address"`
	Port           int      `yaml:"port"`
	Workdir        string   `yaml:"workdir"`
	Command        []string `yaml:"command"`
	HealthTimeout  Duration `yaml:"healthTimeout"`
	StopTimeout    Duration `yaml:"stopTimeout"`
	RequestTimeout Duration `yaml:"requestTimeout"` // per HTTP request; default 5m
}

// Duration unmarshals a Go duration string ("30s", "2m") from YAML.
type Duration time.Duration

func (d *Duration) UnmarshalYAML(unmarshal func(any) error) error {
	var s string
	if err := unmarshal(&s); err != nil {
		return err
	}
	if s == "" {
		return nil
	}
	parsed, err := time.ParseDuration(s)
	if err != nil {
		return fmt.Errorf("invalid duration %q: %w", s, err)
	}
	*d = Duration(parsed)
	return nil
}

// Duration returns the value as time.Duration.
func (d Duration) Duration() time.Duration { return time.Duration(d) }
