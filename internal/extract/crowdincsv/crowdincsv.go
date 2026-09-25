// Package crowdincsv implements a four-column CSV that Crowdin imports and
// exports as-is (scheme "identifier,source_phrase,translation,context", first
// line is a header). It registers as "crowdin-csv".
//
// The pipeline translates the source column and writes the result into the
// translation column; id, source and context are kept. A Crowdin export of the
// same file has the same shape, so both are interchangeable downstream.
package crowdincsv

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/Thrapis/go-game-translator/internal/extract"
)

// Header is the first line of every file.
var Header = []string{"id", "source", "translation", "context"}

const (
	colID = iota
	colSource
	colTranslation
	colContext
)

func init() {
	extract.Register("crowdin-csv", func() extract.Format { return format{} })
}

// Row is one translatable string.
type Row struct {
	ID, Source, Translation, Context string
}

// Read parses a file, header included.
func Read(r io.Reader) ([]Row, error) {
	cr := csv.NewReader(r)
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true
	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("crowdin csv: %w", err)
	}
	if len(records) == 0 {
		return nil, fmt.Errorf("crowdin csv: empty file, want header %v", Header)
	}
	head := records[0]
	if len(head) > 0 {
		head[0] = strings.TrimPrefix(head[0], "\xEF\xBB\xBF")
	}
	if strings.Join(head, ",") != strings.Join(Header, ",") {
		return nil, fmt.Errorf("crowdin csv: header %v, want %v", head, Header)
	}
	rows := make([]Row, 0, len(records)-1)
	for i, rec := range records[1:] {
		if len(rec) != len(Header) {
			return nil, fmt.Errorf("crowdin csv: line %d: %d fields, want %d", i+2, len(rec), len(Header))
		}
		rows = append(rows, Row{rec[colID], rec[colSource], rec[colTranslation], rec[colContext]})
	}
	return rows, nil
}

// Write serialises rows with the header, LF line ends and RFC-4180 quoting.
func Write(w io.Writer, rows []Row) error {
	cw := csv.NewWriter(w)
	if err := cw.Write(Header); err != nil {
		return err
	}
	for _, r := range rows {
		if err := cw.Write([]string{r.ID, r.Source, r.Translation, r.Context}); err != nil {
			return err
		}
	}
	cw.Flush()
	return cw.Error()
}

// doc is the Settings.Extra skeleton: every row, plus which of them were
// handed out as DataLines.
type doc struct {
	rows  []Row
	slots []int
}

type format struct{}

func (format) Extract(r io.Reader, _ string) ([]extract.DataLine, *extract.Settings, error) {
	rows, err := Read(r)
	if err != nil {
		return nil, nil, err
	}
	d := &doc{rows: rows}
	var lines []extract.DataLine
	for i, row := range rows {
		if strings.TrimSpace(row.Source) == "" {
			continue
		}
		d.slots = append(d.slots, i)
		lines = append(lines, extract.DataLine{Key: row.ID, Value: row.Source})
	}
	return lines, &extract.Settings{LineDelimiter: "\n", Extra: d}, nil
}

func (format) Compose(w io.Writer, lines []extract.DataLine, s *extract.Settings, _ string) error {
	d, ok := s.Extra.(*doc)
	if !ok || d == nil {
		return fmt.Errorf("crowdin csv: compose without a parsed document")
	}
	if len(lines) != len(d.slots) {
		return fmt.Errorf("crowdin csv: got %d lines, document has %d source rows", len(lines), len(d.slots))
	}
	for i, l := range lines {
		d.rows[d.slots[i]].Translation = l.Value
	}
	return Write(w, d.rows)
}
