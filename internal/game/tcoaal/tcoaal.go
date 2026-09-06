// Package tcoaal supports "The Coffin of Andy and Leyley" (RPG Maker MV/MZ
// combined export, backslash escape markup). Its localization ships in two
// interchangeable layouts, format ids "tcoaal-csv" and "tcoaal-txt"; pick one
// with source.format in config (default: tcoaal-csv).
package tcoaal

import (
	"strings"

	"github.com/Thrapis/go-csv-translator/internal/extract"
	_ "github.com/Thrapis/go-csv-translator/internal/extract/tcoaalcsv" // register "tcoaal-csv"
	_ "github.com/Thrapis/go-csv-translator/internal/extract/tcoaaltxt" // register "tcoaal-txt"
	"github.com/Thrapis/go-csv-translator/internal/game"
	"github.com/Thrapis/go-csv-translator/internal/markup"
)

func init() { game.Register(New()) }

// Game implements game.Game, game.ReplicaGrouper and game.Parasitizer.
type Game struct {
	parasite parasiteSource
}

// New returns a ready-to-register tcoaal game.
func New() *Game { return &Game{} }

func (*Game) Name() string              { return "tcoaal" }
func (*Game) DefaultFormat() string     { return "tcoaal-csv" }
func (*Game) DefaultDelimiter() string  { return "," }
func (*Game) Analyzer() markup.Analyzer { return analyzer{} }

// SameReplica groups rows sharing a non-empty replica id. Header-section rows
// (Labels/Menus/...) carry no tag and are each their own unit.
func (*Game) SameReplica(a, b extract.DataLine) bool {
	return a.Tag != "" && a.Tag == b.Tag
}

// Replica returns the pre-existing human translation for tag from file.
func (g *Game) Replica(file, tag string) (string, error) {
	return g.parasite.replica(file, tag)
}

// Verbatim marks localization-identity rows (the language name, font, credits)
// that must never be machine-translated - only carried over.
func (*Game) Verbatim(tag string) bool {
	return tag == "LANGUAGE" ||
		strings.HasPrefix(tag, "FONT/") ||
		strings.HasPrefix(tag, "CREDITS/")
}
