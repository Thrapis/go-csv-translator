// Package titanquest supports Titan Quest style localization (=-delimited,
// {curly}/[square] special tokens and %x variables).
package titanquest

import (
	_ "github.com/Thrapis/go-game-translator/internal/extract/delimited" // register "delimited"
	"github.com/Thrapis/go-game-translator/internal/game"
	"github.com/Thrapis/go-game-translator/internal/markup"
)

func init() { game.Register(New()) }

// Game implements game.Game.
type Game struct{}

// New returns a ready-to-register titanquest game.
func New() *Game { return &Game{} }

func (*Game) Name() string              { return "titanquest" }
func (*Game) DefaultFormat() string     { return "delimited" }
func (*Game) DefaultDelimiter() string  { return "=" }
func (*Game) Analyzer() markup.Analyzer { return analyzer{} }
