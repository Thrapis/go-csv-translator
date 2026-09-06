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

	if _, err := src.replica(file, "missing,X"); err == nil {
		t.Error("expected error for unknown replica id")
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
