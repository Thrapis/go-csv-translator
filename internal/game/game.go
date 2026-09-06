// Package game defines what a supported game must provide to the pipeline and a
// registry of games selected by name from config. Each game package self-
// registers from its init and is pulled in via internal/plugins.
package game

import (
	"fmt"
	"sort"

	"github.com/Thrapis/go-csv-translator/internal/extract"
	"github.com/Thrapis/go-csv-translator/internal/markup"
)

// Game is the minimum a game plugin must implement.
type Game interface {
	// Name is the registry key and the value of the config "game" field.
	Name() string
	// Analyzer returns the markup analyzer for this game's engine.
	Analyzer() markup.Analyzer
	// DefaultFormat is the extract format id to use when config does not set one.
	DefaultFormat() string
	// DefaultDelimiter is the field delimiter to use when config does not set one.
	DefaultDelimiter() string
}

// ReplicaGrouper is an optional capability: a game implements it when several
// consecutive rows can form one logical line that must be translated as a unit
// (the pipeline's multi-row-replica mode).
type ReplicaGrouper interface {
	SameReplica(a, b extract.DataLine) bool
}

// Parasitizer is an optional capability: a game implements it when an existing
// human translation can be pulled from an external file by row tag instead of
// machine-translating the row.
type Parasitizer interface {
	Replica(file, tag string) (string, error)
}

var registry = map[string]Game{}

// Register adds g under g.Name(); call it from a game package's init.
func Register(g Game) {
	name := g.Name()
	if _, dup := registry[name]; dup {
		panic(fmt.Sprintf("game: %q registered twice", name))
	}
	registry[name] = g
}

// Get returns the game registered under name.
func Get(name string) (Game, error) {
	g, ok := registry[name]
	if !ok {
		return nil, fmt.Errorf("game: unknown game %q (known: %v)", name, Names())
	}
	return g, nil
}

// Names lists registered games, sorted.
func Names() []string {
	out := make([]string, 0, len(registry))
	for n := range registry {
		out = append(out, n)
	}
	sort.Strings(out)
	return out
}
