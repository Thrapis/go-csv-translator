// Package tcoaalcsv implements the "The Coffin of Andy and Leyley" combined
// export CSV (dialogue.csv). It registers as "tcoaal-csv".
//
// The file is always four columns, CRLF, minimal RFC-4180 quoting. It is one
// document with a metadata header, five translatable sub-sections
// (Labels/Menus/Speakers/Items/Descriptions), then ~194 "Section,<Map>.json"
// dialogue blocks. Every row is kept verbatim on compose; only the translation
// cell of a translatable row is written.
package tcoaalcsv

import (
	"bufio"
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"

	"github.com/Thrapis/go-csv-translator/internal/extract"
)

const crlf = "\r\n"

func init() {
	extract.Register("tcoaal-csv", func() extract.Format { return format{} })
}

// section identifies which translatable block a data row belongs to. The zero
// value (metadata / unknown) is pass-through.
type section int

const (
	secMeta section = iota
	secLabels
	secMenus
	secSpeakers
	secItems
	secDescriptions
	secDialogue
)

// srcCol/dstCol: which column holds the source text and which receives the
// translation, per section.
func (s section) cols() (src, dst int, translatable bool) {
	switch s {
	case secLabels, secSpeakers, secItems:
		return 1, 2, true
	case secMenus:
		return 0, 1, true
	case secDescriptions:
		return 2, 3, true
	case secDialogue:
		return 2, 3, true
	default:
		return 0, 0, false
	}
}

// Doc is a parsed dialogue.csv: every record verbatim, plus the list of cells
// that hold translatable text.
type Doc struct {
	records [][]string
	slots   []slot
}

type slot struct {
	record int
	col    int // destination (translation) column
	sec    section
}

var subHeaders = map[string]section{
	"Labels\x00English\x00Translation\x00":           secLabels,
	"Menus\x00Translation\x00\x00":                   secMenus,
	"Speakers\x00English\x00Translation\x00":         secSpeakers,
	"Items\x00English\x00Translation\x00":            secItems,
	"Descriptions\x00Item\x00English\x00Translation": secDescriptions,
}

func joinKey(rec []string) string { return strings.Join(rec, "\x00") }

// skipBOM drops a leading UTF-8 byte-order mark if present.
func skipBOM(r io.Reader) io.Reader {
	br := bufio.NewReader(r)
	if b, err := br.Peek(3); err == nil && b[0] == 0xEF && b[1] == 0xBB && b[2] == 0xBF {
		_, _ = br.Discard(3)
	}
	return br
}

func isIDHeader(rec []string) bool {
	return len(rec) == 4 && rec[0] == "ID" && rec[1] == "Source" &&
		rec[2] == "English" && rec[3] == "Translation"
}

func allEmpty(rec []string) bool {
	for _, c := range rec {
		if c != "" {
			return false
		}
	}
	return true
}

// ParseDoc reads a combined dialogue.csv.
func ParseDoc(r io.Reader) (*Doc, error) {
	cr := csv.NewReader(skipBOM(r))
	cr.Comma = ','
	cr.FieldsPerRecord = -1
	cr.LazyQuotes = true

	records, err := cr.ReadAll()
	if err != nil {
		return nil, fmt.Errorf("parse tcoaal csv: %w", err)
	}

	d := &Doc{records: records}
	cur := secMeta
	for i, rec := range records {
		if sub, ok := subHeaderOf(rec); ok {
			cur = sub
			continue
		}
		if len(rec) == 4 && rec[0] == "Section" {
			cur = secDialogue
			continue
		}
		if isIDHeader(rec) {
			continue // repeated dialogue header row; stay in secDialogue
		}
		if allEmpty(rec) {
			cur = secMeta
			continue
		}

		src, dst, translatable := cur.cols()
		if !translatable || src >= len(rec) || dst >= len(rec) {
			continue
		}
		if strings.TrimSpace(rec[src]) == "" {
			continue
		}
		d.slots = append(d.slots, slot{record: i, col: dst, sec: cur})
	}
	return d, nil
}

func subHeaderOf(rec []string) (section, bool) {
	if len(rec) != 4 {
		return 0, false
	}
	s, ok := subHeaders[joinKey(rec)]
	return s, ok
}

// idKeyed reports whether a section's rows are addressed by the row id in
// column 0 (dialogue, speakers, items, descriptions) rather than by English key
// text (labels, menus).
func (s section) idKeyed() bool {
	switch s {
	case secDialogue, secSpeakers, secItems, secDescriptions:
		return true
	default:
		return false
	}
}

// tagFor returns the parasite / carry-over lookup key for a row: the bare id for
// id-keyed sections, "LABELS/<key>" or "MENUS/<key>" for the text-keyed ones,
// and "" for metadata.
func (s section) tagFor(col0 string) string {
	switch {
	case s.idKeyed():
		return col0
	case s == secLabels:
		return "LABELS/" + col0
	case s == secMenus:
		return "MENUS/" + col0
	default:
		return ""
	}
}

// lines returns one DataLine per translatable slot. The source column is always
// the one immediately left of the translation column.
func (d *Doc) lines() []extract.DataLine {
	out := make([]extract.DataLine, len(d.slots))
	for i, sl := range d.slots {
		rec := d.records[sl.record]
		out[i] = extract.DataLine{
			Key:   strconv.Itoa(i),
			Value: rec[sl.col-1],
			Tag:   sl.sec.tagFor(rec[0]),
		}
	}
	return out
}

// TranslationsByID maps each translatable row's lookup key (see section.tagFor)
// to its Translation column, joining consecutive same-id dialogue rows with a
// space. Rows with an empty translation cell are skipped. Used when this
// document is a parasite / carry-over source.
func (d *Doc) TranslationsByID() map[string]string {
	out := map[string]string{}
	for _, sl := range d.slots {
		rec := d.records[sl.record]
		key := sl.sec.tagFor(rec[0])
		if key == "" {
			continue
		}
		tr := strings.TrimSpace(rec[sl.col])
		if tr == "" {
			continue
		}
		if prev, ok := out[key]; ok && sl.sec.idKeyed() {
			out[key] = prev + " " + tr
		} else {
			out[key] = tr
		}
	}
	return out
}

// Write serialises the document. It matches the exporter's own quoting rule
// (quote a field only when it contains ',' '"' CR or LF) rather than
// encoding/csv's, which also quotes leading-space fields and would not
// round-trip the source byte-for-byte.
func (d *Doc) Write(w io.Writer) error {
	return WriteRecords(w, d.records)
}

// WriteRecords writes CSV records with the exporter's quoting rule and CRLF.
func WriteRecords(w io.Writer, records [][]string) error {
	bw := bufio.NewWriter(w)
	for _, rec := range records {
		for i, field := range rec {
			if i > 0 {
				bw.WriteByte(',')
			}
			writeField(bw, field)
		}
		bw.WriteString(crlf)
	}
	return bw.Flush()
}

func writeField(bw *bufio.Writer, s string) {
	if !strings.ContainsAny(s, ",\"\r\n") {
		bw.WriteString(s)
		return
	}
	bw.WriteByte('"')
	bw.WriteString(strings.ReplaceAll(s, `"`, `""`))
	bw.WriteByte('"')
}

// Sections lists the dialogue section names in order.
func (d *Doc) Sections() []string {
	var out []string
	for _, rec := range d.records {
		if len(rec) == 4 && rec[0] == "Section" {
			out = append(out, rec[1])
		}
	}
	return out
}

// SectionRecords returns the rows of one dialogue section, starting with the
// repeated ID,Source,English,Translation header.
func (d *Doc) SectionRecords(name string) [][]string {
	var out [][]string
	in := false
	for _, rec := range d.records {
		if len(rec) == 4 && rec[0] == "Section" {
			in = rec[1] == name
			continue
		}
		if !in {
			continue
		}
		if allEmpty(rec) {
			break
		}
		out = append(out, rec)
	}
	return out
}

type format struct{}

func (format) Extract(r io.Reader, _ string) ([]extract.DataLine, *extract.Settings, error) {
	d, err := ParseDoc(r)
	if err != nil {
		return nil, nil, err
	}
	return d.lines(), &extract.Settings{LineDelimiter: crlf, Extra: d}, nil
}

func (format) Compose(w io.Writer, lines []extract.DataLine, s *extract.Settings, _ string) error {
	d, ok := s.Extra.(*Doc)
	if !ok || d == nil {
		return fmt.Errorf("tcoaal csv: compose without a parsed document")
	}
	if len(lines) != len(d.slots) {
		return fmt.Errorf("tcoaal csv: got %d lines, document has %d translatable cells", len(lines), len(d.slots))
	}
	for i, l := range lines {
		sl := d.slots[i]
		d.records[sl.record][sl.col] = l.Value
	}
	return d.Write(w)
}

// Bytes is a small helper for tests.
func (d *Doc) Bytes() []byte {
	var b bytes.Buffer
	_ = d.Write(&b)
	return b.Bytes()
}
