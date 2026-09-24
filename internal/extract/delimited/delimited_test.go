package delimited

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Thrapis/go-csv-translator/internal/extract"
)

func TestExtractComposeRoundTrip(t *testing.T) {
	f, err := extract.Get("delimited")
	if err != nil {
		t.Fatal(err)
	}

	src := "greeting=Hello world\r\nfarewell=Goodbye\r\nnodelimiterline\r\n"
	lines, settings, err := f.Extract(strings.NewReader(src), "=")
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 2 {
		t.Fatalf("got %d lines, want 2 (the line without a delimiter is dropped)", len(lines))
	}
	if lines[0].Key != "greeting" || lines[0].Value != "Hello world" {
		t.Errorf("line 0 = %+v", lines[0])
	}
	if settings.LineDelimiter != "\r\n" {
		t.Errorf("LineDelimiter = %q, want CRLF", settings.LineDelimiter)
	}

	var out bytes.Buffer
	if err := f.Compose(&out, lines, settings, "="); err != nil {
		t.Fatal(err)
	}
	want := "greeting=Hello world\r\nfarewell=Goodbye\r\n"
	if out.String() != want {
		t.Errorf("compose = %q, want %q", out.String(), want)
	}
}
