package tcoaaltxt

import (
	"strings"
	"testing"

	"github.com/Thrapis/go-csv-translator/internal/extract"
)

const fixture = "[VERSION]\r\n" +
	"3.0.13 : deadbeef\r\n" +
	"\r\n" +
	"[LANGUAGE]\r\n" +
	"English\r\n" +
	"\r\n" +
	"[LABELS]\r\n" +
	"Game : The Coffin\r\n" +
	"\r\n" +
	"[MENUS]\r\n" +
	"New Game    : New Game\r\n" +
	"\r\n" +
	"[SPEAKERS]\r\n" +
	"#id1 : Ashley\r\n" +
	"\r\n" +
	"[DESCRIPTIONS]\r\n" +
	"\r\n" +
	"#k1 (Axe)\r\n" +
	": Chop chop\r\n" +
	"\r\n" +
	"#k2 (Cap)\r\n" +
	": \r\n" +
	"\r\n" +
	"[Map001.json]\r\n" +
	"\r\n" +
	"#a1 (Narrator)\r\n" +
	": Who wouldn't love \r\n" +
	": being stuck at home?\r\n" +
	"\r\n" +
	"#a2 (Andrew)\r\n" +
	": \"Ah.\"\r\n" +
	"\r\n" +
	"[CHOICES]\r\n" +
	"#c1 : Call again.\r\n" +
	"\r\n"

func TestExtractSlots(t *testing.T) {
	f := format{}
	lines, s, err := f.Extract(strings.NewReader(fixture), "")
	if err != nil {
		t.Fatal(err)
	}
	// Game, New Game, Ashley, "Chop chop", "" (k2), 2×Map001 a1, a2 "Ah.", c1 => 9
	if len(lines) != 9 {
		t.Fatalf("got %d slots, want 9: %+v", len(lines), lines)
	}
	if lines[0].Value != "The Coffin" {
		t.Errorf("LABELS value = %q", lines[0].Value)
	}
	if lines[1].Value != "New Game" { // padded key stripped, value only
		t.Errorf("MENUS value = %q", lines[1].Value)
	}
	if lines[2].Value != "Ashley" || lines[2].Tag != "#id1" {
		t.Errorf("SPEAKERS slot = %+v", lines[2])
	}
	if lines[3].Value != "Chop chop" || lines[3].Tag != "#k1" {
		t.Errorf("DESCRIPTIONS slot = %+v", lines[3])
	}
	// Map001 a1 has two value lines sharing the tag
	if lines[5].Tag != "#a1" || lines[6].Tag != "#a1" {
		t.Errorf("replica tags = %q,%q want #a1,#a1", lines[5].Tag, lines[6].Tag)
	}
	if lines[5].Value != "Who wouldn't love " {
		t.Errorf("value keeps trailing space: %q", lines[5].Value)
	}
	if lines[8].Value != "Call again." || lines[8].Tag != "#c1" {
		t.Errorf("CHOICES slot = %+v", lines[8])
	}
	_ = s
}

func TestComposeRoundTrip(t *testing.T) {
	f := format{}
	lines, s, err := f.Extract(strings.NewReader(fixture), "")
	if err != nil {
		t.Fatal(err)
	}
	var b strings.Builder
	if err := f.Compose(&b, lines, s, ""); err != nil {
		t.Fatal(err)
	}
	if b.String() != fixture {
		t.Errorf("round trip differs:\n got: %q\nwant: %q", b.String(), fixture)
	}
}

func TestComposeSubstitutes(t *testing.T) {
	f := format{}
	lines, s, err := f.Extract(strings.NewReader(fixture), "")
	if err != nil {
		t.Fatal(err)
	}
	for i := range lines {
		if lines[i].Value != "" {
			lines[i].Value = "X"
		}
	}
	var b strings.Builder
	if err := f.Compose(&b, lines, s, ""); err != nil {
		t.Fatal(err)
	}
	out := b.String()
	for _, want := range []string{
		"Game : X\r\n",
		"New Game    : X\r\n", // padded key preserved
		"#id1 : X\r\n",
		"#k1 (Axe)\r\n: X\r\n",
		"#c1 : X\r\n",
		"3.0.13 : deadbeef\r\n", // VERSION untouched
		"English\r\n",           // LANGUAGE untouched
	} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "Who wouldn't love") {
		t.Error("dialogue value not substituted")
	}
}

func TestRegistered(t *testing.T) {
	if _, err := extract.Get("tcoaal-txt"); err != nil {
		t.Fatal(err)
	}
}
