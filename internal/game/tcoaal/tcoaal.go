// Package tcoaal supports "The Coffin of Andy and Leyley" (RPG Maker MV/MZ
// export CSV, backslash escape markup).
package tcoaal

import (
	"github.com/Thrapis/go-csv-translator/internal/extract"
	_ "github.com/Thrapis/go-csv-translator/internal/extract/tcoaalcsv" // register "tcoaal-csv"
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

// SameReplica groups rows sharing an ID+Source tag.
func (*Game) SameReplica(a, b extract.DataLine) bool { return a.Tag == b.Tag }

// Replica returns the pre-existing human translation for tag from file.
func (g *Game) Replica(file, tag string) (string, error) {
	return g.parasite.replica(file, tag)
}
