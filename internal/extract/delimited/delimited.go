// Package delimited implements the generic "key<delim>value" line format, with
// charset auto-detection and CRLF/LF preservation. It registers as "delimited".
package delimited

import (
	"fmt"
	"io"
	"strings"

	"github.com/Thrapis/go-csv-translator/internal/extract"

	"golang.org/x/net/html/charset"
)

const (
	crlf = "\r\n"
	lf   = "\n"
)

func init() {
	extract.Register("delimited", func() extract.Format { return format{} })
}

type format struct{}

func (format) Extract(r io.Reader, delim string) ([]extract.DataLine, *extract.Settings, error) {
	raw, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}

	enc, _, _ := charset.DetermineEncoding(raw, "")
	utf8Bytes, err := enc.NewDecoder().Bytes(raw)
	if err != nil {
		return nil, nil, fmt.Errorf("decode source encoding: %w", err)
	}
	text := string(utf8Bytes)

	lineDelim := lf
	if strings.Contains(text, crlf) {
		lineDelim = crlf
	}

	settings := &extract.Settings{Encoding: enc, LineDelimiter: lineDelim}

	var lines []extract.DataLine
	for _, line := range strings.Split(text, lineDelim) {
		if before, after, ok := strings.Cut(line, delim); ok {
			lines = append(lines, extract.DataLine{Key: before, Value: after})
		}
	}
	return lines, settings, nil
}

func (format) Compose(w io.Writer, lines []extract.DataLine, s *extract.Settings, delim string) error {
	var b strings.Builder
	for _, l := range lines {
		fmt.Fprintf(&b, "%s%s%s%s", l.Key, delim, l.Value, s.LineDelimiter)
	}

	out, err := s.Encoding.NewEncoder().Bytes([]byte(b.String()))
	if err != nil {
		return fmt.Errorf("encode destination encoding: %w", err)
	}
	_, err = w.Write(out)
	return err
}
