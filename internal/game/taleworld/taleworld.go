// Package taleworld supports Mount & Blade II: Bannerlord style localization
// (pipe-delimited, {curly} variable / gender / ternary markup).
package taleworld

import (
	_ "github.com/Thrapis/go-csv-translator/internal/extract/delimited" // register "delimited"
	"github.com/Thrapis/go-csv-translator/internal/game"
	"github.com/Thrapis/go-csv-translator/internal/markup"
)

func init() { game.Register(New()) }

// Game implements game.Game.
type Game struct{}

// New returns a ready-to-register taleworld game.
func New() *Game { return &Game{} }

func (*Game) Name() string              { return "taleworld" }
func (*Game) DefaultFormat() string     { return "delimited" }
func (*Game) DefaultDelimiter() string  { return "|" }
func (*Game) Analyzer() markup.Analyzer { return analyzer{} }
