// Package tcoaalcsv implements the "The Coffin of Andy and Leyley" export CSV
// (columns ID,Source,English,Translation). It registers as "tcoaal-csv".
package tcoaalcsv

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"github.com/Thrapis/go-csv-translator/internal/extract"

	"github.com/gocarina/gocsv"
	"golang.org/x/net/html/charset"
)

const crlf = "\r\n"

func init() {
	extract.Register("tcoaal-csv", func() extract.Format { return format{} })
}

type row struct {
	ID          string `csv:"ID"`
	Source      string `csv:"Source"`
	English     string `csv:"English"`
	Translation string `csv:"Translation"`
}

type format struct{}

func (format) Extract(r io.Reader, _ string) ([]extract.DataLine, *extract.Settings, error) {
	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		cr := csv.NewReader(in)
		cr.Comma = ','
		return cr
	})

	var rows []row
	if err := gocsv.Unmarshal(r, &rows); err != nil {
		return nil, nil, fmt.Errorf("parse tcoaal csv: %w", err)
	}

	enc, _ := charset.Lookup("utf8")
	settings := &extract.Settings{Encoding: enc, LineDelimiter: crlf}

	lines := make([]extract.DataLine, 0, len(rows))
	for _, r := range rows {
		lines = append(lines, extract.DataLine{
			Key:   fmt.Sprintf("%s,%s,%s", r.ID, r.Source, r.English),
			Value: r.English,
			Tag:   fmt.Sprintf("%s,%s", r.ID, r.Source),
		})
	}
	return lines, settings, nil
}

func (format) Compose(w io.Writer, lines []extract.DataLine, _ *extract.Settings, _ string) error {
	gocsv.SetCSVWriter(func(out io.Writer) *gocsv.SafeCSVWriter {
		cw := csv.NewWriter(out)
		cw.Comma = ','
		return gocsv.NewSafeCSVWriter(cw)
	})

	rows := make([]row, 0, len(lines))
	for _, l := range lines {
		parts := strings.SplitN(l.Key, ",", 3)
		if len(parts) < 3 {
			return fmt.Errorf("tcoaal csv: malformed key %q", l.Key)
		}
		rows = append(rows, row{
			ID:          parts[0],
			Source:      parts[1],
			English:     parts[2],
			Translation: l.Value,
		})
	}

	str, err := gocsv.MarshalString(&rows)
	if err != nil {
		return fmt.Errorf("write tcoaal csv: %w", err)
	}
	_, err = io.WriteString(w, str)
	return err
}
