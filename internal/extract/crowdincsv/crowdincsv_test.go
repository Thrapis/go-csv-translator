package crowdincsv

import (
	"bytes"
	"strings"
	"testing"

	"github.com/Thrapis/go-csv-translator/internal/extract"
)

const sample = "id,source,translation,context\n" +
	"40,Новости,,\"Gameplay-News, main\"\n" +
	"41,\"Первая\nвторая\",,\n" +
	"42,,,empty source is skipped\n" +
	"43@male,\"Он сказал \"\"да\"\"\",,male V variant\n"

func TestRoundTrip(t *testing.T) {
	f, _ := extract.Get("crowdin-csv")
	lines, s, err := f.Extract(strings.NewReader(sample), ",")
	if err != nil {
		t.Fatal(err)
	}
	if len(lines) != 3 {
		t.Fatalf("got %d lines, want 3: %+v", len(lines), lines)
	}
	if lines[1].Key != "41" || lines[1].Value != "Первая\nвторая" {
		t.Errorf("line 1 = %+v", lines[1])
	}

	var untouched bytes.Buffer
	if err := f.Compose(&untouched, lines, s, ","); err != nil {
		t.Fatal(err)
	}
	// Compose copies the source into translation when nothing was translated.
	rows, err := Read(&untouched)
	if err != nil {
		t.Fatal(err)
	}
	if rows[0].Translation != "Новости" || rows[2].Translation != "" {
		t.Errorf("rows = %+v", rows)
	}

	lines[0].Value = "Навіны"
	lines[2].Value = `Ён сказаў "так"`
	var out bytes.Buffer
	if err := f.Compose(&out, lines, s, ","); err != nil {
		t.Fatal(err)
	}
	rows, err = Read(&out)
	if err != nil {
		t.Fatal(err)
	}
	want := []Row{
		{"40", "Новости", "Навіны", "Gameplay-News, main"},
		{"41", "Первая\nвторая", "Первая\nвторая", ""},
		{"42", "", "", "empty source is skipped"},
		{"43@male", `Он сказал "да"`, `Ён сказаў "так"`, "male V variant"},
	}
	for i := range want {
		if rows[i] != want[i] {
			t.Errorf("row %d = %+v, want %+v", i, rows[i], want[i])
		}
	}
}

func TestBadHeader(t *testing.T) {
	if _, err := Read(strings.NewReader("key,value\na,b\n")); err == nil {
		t.Fatal("expected header error")
	}
	if _, err := Read(strings.NewReader("")); err == nil {
		t.Fatal("expected empty-file error")
	}
}
