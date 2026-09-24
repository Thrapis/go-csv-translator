// Package extract defines how a localization file is read into rows and written
// back out. A concrete file layout (plain delimited text, a specific CSV schema,
// ...) is a Format registered under a string id and selected from config.
package extract

import (
	"io"

	"golang.org/x/text/encoding"
)

// DataLine is one translatable row.
//
//   - Key is everything needed to rebuild the row on Compose except the
//     translated value (for delimited files it is the text before the
//     delimiter; for the TCOAAL CSV it is "ID,Source,English").
//   - Value is the source text to translate, and after the pipeline runs, the
//     translation.
//   - Tag groups rows that belong to the same logical unit (e.g. one line of
//     dialogue split across several rows). Empty when the format has no grouping.
type DataLine struct {
	Key   string
	Value string
	Tag   string
}

// Settings carries per-file decisions an Extractor made that Compose must honour
// to round-trip the file (its text encoding and line terminator), plus an
// optional format-private payload.
type Settings struct {
	Encoding      encoding.Encoding
	LineDelimiter string

	// Extra is opaque data an Extractor may hand to its own Composer, e.g. the
	// structural skeleton of a document whose translatable cells were returned
	// as DataLines. Formats that do not need it leave it nil.
	Extra any
}

// Extractor parses a source file into rows.
type Extractor interface {
	Extract(r io.Reader, delim string) ([]DataLine, *Settings, error)
}

// Composer writes rows back to a destination file.
type Composer interface {
	Compose(w io.Writer, lines []DataLine, s *Settings, delim string) error
}

// Format is a matched Extractor/Composer pair for one file layout.
type Format interface {
	Extractor
	Composer
}
