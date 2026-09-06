// Package translate defines the translator abstraction and a registry of
// translation backends selected by name from config. Backends self-register from
// their init; use Build to turn a list of names into one Translator.
package translate

import (
	"context"
	"fmt"
	"log/slog"
	"sort"
)

// Translator turns text in language from into language to.
type Translator interface {
	Translate(ctx context.Context, text, from, to string) (string, error)
}

// BatchTranslator is an optional Translator capability: translating many
// strings in one call. Use the Batch helper, which falls back to a loop for
// backends that do not implement it.
type BatchTranslator interface {
	TranslateBatch(ctx context.Context, texts []string, from, to string) ([]string, error)
}

// Batch translates texts in order, using t's batch method when it has one.
func Batch(ctx context.Context, t Translator, texts []string, from, to string) ([]string, error) {
	if bt, ok := t.(BatchTranslator); ok {
		return bt.TranslateBatch(ctx, texts, from, to)
	}
	out := make([]string, len(texts))
	for i, s := range texts {
		v, err := t.Translate(ctx, s, from, to)
		if err != nil {
			return nil, err
		}
		out[i] = v
	}
	return out, nil
}

// Options are the backend-agnostic knobs a Factory may need. It deliberately
// does not depend on the config package so backends stay decoupled from it.
type Options struct {
	// LingvanexAddress and LingvanexPort locate the local Lingvanex server.
	LingvanexAddress string
	LingvanexPort    int
}

// Factory builds one backend instance.
type Factory func(opts Options, log *slog.Logger) (Translator, error)

var registry = map[string]Factory{}

// Register adds a backend under name; call it from a backend package's init.
func Register(name string, f Factory) {
	if _, dup := registry[name]; dup {
		panic(fmt.Sprintf("translate: backend %q registered twice", name))
	}
	registry[name] = f
}

// Names lists registered backends, sorted.
func Names() []string {
	out := make([]string, 0, len(registry))
	for n := range registry {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}

// Build resolves names to backends and wraps them in a fallback Chain in the
// given order. At least one name is required.
func Build(names []string, opts Options, log *slog.Logger) (Translator, error) {
	if len(names) == 0 {
		return nil, fmt.Errorf("translate: no backends configured")
	}
	ts := make([]Translator, 0, len(names))
	for _, name := range names {
		factory, ok := registry[name]
		if !ok {
			return nil, fmt.Errorf("translate: unknown backend %q (known: %v)", name, Names())
		}
		t, err := factory(opts, log)
		if err != nil {
			return nil, fmt.Errorf("translate: init backend %q: %w", name, err)
		}
		ts = append(ts, t)
	}
	if len(ts) == 1 {
		return ts[0], nil
	}
	return &Chain{backends: ts, log: log}, nil
}
