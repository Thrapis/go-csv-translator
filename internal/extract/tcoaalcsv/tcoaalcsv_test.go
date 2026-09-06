package tcoaalcsv

import (
	"strings"
	"testing"

	"github.com/Thrapis/go-csv-translator/internal/extract"
)

// fixture mirrors the real dialogue.csv shape in miniature: metadata header,
// one of each translatable sub-section, then two Section blocks (one with a
// 2-row replica and a CHOICE(1) row).
const fixture = "Version,,,\r\n" +
	"3.0.13 : deadbeef,,,\r\n" +
	",,,\r\n" +
	"Language,Font File,Font Size,\r\n" +
	"English,GameFont,28,\r\n" +
	",,,\r\n" +
	"Labels,English,Translation,\r\n" +
	"Game,The Coffin,,\r\n" +
	",,,\r\n" +
	"Menus,Translation,,\r\n" +
	"New Game,,,\r\n" +
	",,,\r\n" +
	"Speakers,English,Translation,\r\n" +
	"id1,Ashley,,\r\n" +
	",,,\r\n" +
	"Descriptions,Item,English,Translation\r\n" +
	"k1,Axe,Chop chop,\r\n" +
	",,,\r\n" +
	",,,\r\n" +
	"Section,Map001.json,,\r\n" +
	"ID,Source,English,Translation\r\n" +
	"a1,Narrator,\"He said, quietly\",\r\n" +
	"a2,Andrew,line one ,\r\n" +
	"a2,Andrew,line two,\r\n" +
	"c1,CHOICE(1),Call again.,\r\n" +
	",,,\r\n" +
	",,,\r\n" +
	"Section,Map002.json,,\r\n" +
	"ID,Source,English,Translation\r\n" +
	"b1,Narrator,Nice.,\r\n"

func TestExtractSlots(t *testing.T) {
	f := format{}
	lines, s, err := f.Extract(strings.NewReader(fixture), ",")
	if err != nil {
		t.Fatal(err)
	}
	// Labels(Game), Menus(New Game), Speakers(Ashley), Descriptions(Chop chop),
	// Map001: a1, a2, a2, c1 ; Map002: b1  => 9
	if len(lines) != 9 {
		t.Fatalf("got %d translatable lines, want 9: %+v", len(lines), lines)
	}
	if lines[0].Value != "The Coffin" {
		t.Errorf("Labels source = %q", lines[0].Value)
	}
	if lines[1].Value != "New Game" {
		t.Errorf("Menus source = %q", lines[1].Value)
	}
	if lines[4].Value != "He said, quietly" || lines[4].Tag != "a1" {
		t.Errorf("dialogue a1 = %+v", lines[4])
	}
	if lines[5].Tag != "a2" || lines[6].Tag != "a2" {
		t.Errorf("replica tags = %q,%q want a2,a2", lines[5].Tag, lines[6].Tag)
	}
	if s.Extra == nil {
		t.Fatal("Settings.Extra not set")
	}
}

func TestParseWriteIdentity(t *testing.T) {
	d, err := ParseDoc(strings.NewReader(fixture))
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	if err := d.Write(&b); err != nil {
		t.Fatal(err)
	}
	if b.String() != fixture {
		t.Errorf("parse+write not identity:\n got: %q\nwant: %q", b.String(), fixture)
	}
}

func TestComposeWritesCorrectColumns(t *testing.T) {
	f := format{}
	lines, s, err := f.Extract(strings.NewReader(fixture), ",")
	if err != nil {
		t.Fatal(err)
	}
	for i := range lines {
		lines[i].Value = "<" + lines[i].Value + ">"
	}
	var b strings.Builder
	if err := f.Compose(&b, lines, s, ","); err != nil {
		t.Fatal(err)
	}
	out := b.String()

	for _, want := range []string{
		"Game,The Coffin,<The Coffin>,\r\n", // Labels: col1 kept, col2 filled
		"New Game,<New Game>,,\r\n",         // Menus: col0 kept, col1 filled
		"id1,Ashley,<Ashley>,\r\n",          // Speakers
		"k1,Axe,Chop chop,<Chop chop>\r\n",  // Descriptions: col2 kept, col3 filled
		"a1,Narrator,\"He said, quietly\",\"<He said, quietly>\"\r\n",
		"c1,CHOICE(1),Call again.,<Call again.>\r\n",
		"b1,Narrator,Nice.,<Nice.>\r\n",
	} {
		if !strings.Contains(out, want) {
			t.Errorf("output missing %q\n---\n%s", want, out)
		}
	}
	// metadata untouched
	for _, keep := range []string{"Version,,,\r\n", "3.0.13 : deadbeef,,,\r\n", "English,GameFont,28,\r\n"} {
		if !strings.Contains(out, keep) {
			t.Errorf("metadata row changed, missing %q", keep)
		}
	}
}

func TestRegistered(t *testing.T) {
	if _, err := extract.Get("tcoaal-csv"); err != nil {
		t.Fatal(err)
	}
}

func TestSectionsAndRecords(t *testing.T) {
	d, err := ParseDoc(strings.NewReader(fixture))
	if err != nil {
		t.Fatal(err)
	}
	got := d.Sections()
	if len(got) != 2 || got[0] != "Map001.json" || got[1] != "Map002.json" {
		t.Fatalf("sections = %v", got)
	}
	recs := d.SectionRecords("Map001.json")
	if len(recs) == 0 || recs[0][0] != "ID" {
		t.Fatalf("Map001 records start = %v", recs)
	}
	if recs[len(recs)-1][0] != "c1" {
		t.Errorf("Map001 last record = %v, want c1 row", recs[len(recs)-1])
	}
}
