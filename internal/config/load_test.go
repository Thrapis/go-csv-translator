package config

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"
)

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	dir := t.TempDir()
	src := filepath.Join(dir, "src")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	para := filepath.Join(dir, "d.txt")
	if err := os.WriteFile(para, []byte("#x : y\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "config.yaml")
	body = "source:\n  folder: " + strconv.Quote(src) + "\n" +
		strings.ReplaceAll(body, "%PARA%", strconv.Quote(para))
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
	return path
}

func TestLoadValid(t *testing.T) {
	path := writeConfig(t, `
game: tcoaal
destination:
  folder: "out"
language:
  source: ru
  target: be
parasitizing:
  enabled: true
  files: [%PARA%]
lingvanex:
  healthTimeout: 45s
`)
	cfg, err := Load(path)
	if err != nil {
		t.Fatal(err)
	}
	if cfg.Game != "tcoaal" {
		t.Errorf("game = %q", cfg.Game)
	}
	if cfg.Lingvanex.HealthTimeout.Duration() != 45*time.Second {
		t.Errorf("healthTimeout = %s", cfg.Lingvanex.HealthTimeout.Duration())
	}
	if got := cfg.Translators; len(got) != 2 || got[0] != "lingvanex" {
		t.Errorf("default translators = %v", got)
	}
	if cfg.Lingvanex.Port != 8000 {
		t.Errorf("default port = %d", cfg.Lingvanex.Port)
	}
}

func TestLoadMissingGame(t *testing.T) {
	path := writeConfig(t, `
destination:
  folder: "out"
language:
  source: ru
  target: be
`)
	if _, err := Load(path); err == nil {
		t.Fatal("expected error for missing game")
	}
}

func TestLoadParasitizingWithoutFiles(t *testing.T) {
	path := writeConfig(t, `
game: tcoaal
destination:
  folder: "out"
language:
  source: ru
  target: be
parasitizing:
  enabled: true
`)
	if _, err := Load(path); err == nil {
		t.Fatal("expected error: parasitizing.enabled without files")
	}
}

func TestLoadParasitizingMissingFile(t *testing.T) {
	path := writeConfig(t, `
game: tcoaal
destination:
  folder: "out"
language:
  source: ru
  target: be
parasitizing:
  enabled: true
  files: ["nope-does-not-exist.txt"]
`)
	if _, err := Load(path); err == nil {
		t.Fatal("expected error for missing parasite file")
	}
}

func TestLoadUnknownFieldRejected(t *testing.T) {
	path := writeConfig(t, `
game: tcoaal
destination:
  folder: "out"
language:
  source: ru
  target: be
notARealKey: 1
`)
	if _, err := Load(path); err == nil {
		t.Fatal("expected error for unknown field")
	}
}

func TestLoadBadDuration(t *testing.T) {
	path := writeConfig(t, `
game: tcoaal
destination:
  folder: "out"
language:
  source: ru
  target: be
lingvanex:
  healthTimeout: "not-a-duration"
`)
	if _, err := Load(path); err == nil {
		t.Fatal("expected error for invalid duration")
	}
}
