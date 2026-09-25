// Package cyberpunk2077 supports Cyberpunk 2077 (REDengine 4) localization.
// The game's WolvenKit JSON exports are flattened to Crowdin CSV by
// tools/cp77loc before translation and rebuilt from it afterwards, so the
// pipeline itself only ever sees "crowdin-csv" files.
package cyberpunk2077

import (
	"regexp"

	_ "github.com/Thrapis/go-csv-translator/internal/extract/crowdincsv" // register "crowdin-csv"
	"github.com/Thrapis/go-csv-translator/internal/game"
	"github.com/Thrapis/go-csv-translator/internal/markup"
)

func init() { game.Register(New()) }

// Game implements game.Game. Every row is independent: no replica grouping,
// parasitizing or verbatim rows.
type Game struct{}

// New returns a ready-to-register Cyberpunk 2077 game.
func New() *Game { return &Game{} }

func (*Game) Name() string              { return "cyberpunk2077" }
func (*Game) DefaultFormat() string     { return "crowdin-csv" }
func (*Game) DefaultDelimiter() string  { return "," }
func (*Game) Analyzer() markup.Analyzer { return analyzer{} }

var voPlaceholderRe = regexp.MustCompile(`^\[[A-Za-z]{2}_[A-Za-z]{2}\]`)

// IsVOPlaceholder reports whether s is an untranslated voice-over stub such as
// "[en_us][db_db]Good evening, Night City." - English text the game keeps as a
// placeholder in every language. Such strings are never translated.
func IsVOPlaceholder(s string) bool { return voPlaceholderRe.MatchString(s) }

var c1ControlRe = regexp.MustCompile(`[\x{80}-\x{9F}]`)

// IsGlitchText reports whether s is one of the game's deliberately "corrupted"
// strings (glitched shards/e-mails): Zalgo text whose UTF-8 bytes were stored
// as Latin-1, which leaves C1 control characters (U+0080-U+009F) in it. Real
// Russian/Belarusian text never contains those. Such strings are never
// translated.
func IsGlitchText(s string) bool { return c1ControlRe.MatchString(s) }

// Untranslatable reports whether s must be kept exactly as the source.
func Untranslatable(s string) bool { return IsVOPlaceholder(s) || IsGlitchText(s) }
