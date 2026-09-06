package tcoaal

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParasiteReplica(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "dialogue.txt")
	// Two replicas, CRLF line endings, blank-line separated.
	content := "#abc123\r\n" +
		": Hello there.\r\n" +
		": How are you?\r\n" +
		"\r\n" +
		"#def456\r\n" +
		": Goodbye.\r\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	var src parasiteSource

	got, err := src.replica(file, "abc123,SomeSource")
	if err != nil {
		t.Fatal(err)
	}
	if want := "Hello there. How are you?"; got != want {
		t.Errorf("replica abc123 = %q, want %q", got, want)
	}

	got, err = src.replica(file, "def456,Other")
	if err != nil {
		t.Fatal(err)
	}
	if want := "Goodbye."; got != want {
		t.Errorf("replica def456 = %q, want %q", got, want)
	}

	// An unknown id is not an error: the pipeline falls back to machine
	// translation for replicas the parasite file does not cover.
	got, err = src.replica(file, "missing,X")
	if err != nil {
		t.Errorf("unknown id returned error: %v", err)
	}
	if got != "" {
		t.Errorf("unknown id returned %q, want empty", got)
	}
}

func TestParasiteTXTInlineForm(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "dialogue.txt")
	content := "[SPEAKERS]\r\n" +
		"#sp1 : Эшли\r\n" +
		"#sp2 : Эндрю\r\n" +
		"\r\n" +
		"[Map001.json]\r\n" +
		"#d1 (Narrator)\r\n" +
		": Файна.\r\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	var src parasiteSource
	for tag, want := range map[string]string{"sp1": "Эшли", "sp2": "Эндрю", "d1": "Файна."} {
		got, err := src.replica(file, tag)
		if err != nil {
			t.Fatal(err)
		}
		if got != want {
			t.Errorf("%s = %q, want %q", tag, got, want)
		}
	}
}

func TestParasiteCSV(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "dialogue.csv")
	content := "Speakers,English,Translation,\r\n" +
		"sp1,Ashley,Эшли,\r\n" +
		",,,\r\n" +
		",,,\r\n" +
		"Section,Map001.json,,\r\n" +
		"ID,Source,English,Translation\r\n" +
		"d1,Narrator,part one ,частка адна \r\n" +
		"d1,Narrator,part two,частка два\r\n"
	if err := os.WriteFile(file, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
	var src parasiteSource
	if got, _ := src.replica(file, "#d1"); got != "частка адна частка два" {
		t.Errorf("multi-row csv replica = %q", got)
	}
	if got, _ := src.replica(file, "sp1,Ashley"); got != "Эшли" {
		t.Errorf("csv speaker = %q", got)
	}
	if got, _ := src.replica(file, "nope"); got != "" {
		t.Errorf("unknown csv id = %q", got)
	}
}

func TestParasiteSourceCachesFile(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "dialogue.txt")
	if err := os.WriteFile(file, []byte("#id1\r\n: Cached.\r\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	var src parasiteSource
	if _, err := src.replica(file, "id1,S"); err != nil {
		t.Fatal(err)
	}

	// Remove the file; a cached source must still resolve.
	if err := os.Remove(file); err != nil {
		t.Fatal(err)
	}
	got, err := src.replica(file, "id1,S")
	if err != nil {
		t.Fatalf("expected cache hit, got error: %v", err)
	}
	if got != "Cached." {
		t.Errorf("got %q, want %q", got, "Cached.")
	}
}
