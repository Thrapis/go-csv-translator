package csvtool

import (
	"os"
	"path/filepath"
	"testing"
)

const (
	alphaBody = "ID,Source,English,Translation\r\n1,x,Hello,\r\n"
	betaBody  = "ID,Source,English,Translation\r\n2,y,World,\r\n"
)

func TestMerge(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "sections")
	if err := os.MkdirAll(src, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "alpha.csv"), []byte(alphaBody), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(src, "beta.csv"), []byte(betaBody), 0o644); err != nil {
		t.Fatal(err)
	}

	out := filepath.Join(dir, "merged.csv")
	if err := Merge(src, out); err != nil {
		t.Fatal(err)
	}

	got, err := os.ReadFile(out)
	if err != nil {
		t.Fatal(err)
	}
	want := "Section,alpha\r\n" + alphaBody + sectionSeparator + "Section,beta\r\n" + betaBody
	if string(got) != want {
		t.Errorf("merged output mismatch\n got: %q\nwant: %q", got, want)
	}
}

func TestSplit(t *testing.T) {
	dir := t.TempDir()
	in := filepath.Join(dir, "export.csv")
	// Section bodies do NOT end with CRLF before the 3-CRLF separator, matching
	// the shape of the game's own export that Split is designed to consume.
	content := "Section,alpha\r\nBODY-A" + sectionSeparator + "Section,beta\r\nBODY-B"
	if err := os.WriteFile(in, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	if err := Split(in); err != nil {
		t.Fatal(err)
	}

	outDir := filepath.Join(dir, "export")
	// Final section: exact body.
	if got := readFile(t, filepath.Join(outDir, "beta.csv")); got != "BODY-B" {
		t.Errorf("beta.csv = %q, want %q", got, "BODY-B")
	}
	// Earlier sections keep the trailing separator: SplitAfter leaves it on the
	// chunk. Documented quirk of the original implementation, preserved here.
	if got := readFile(t, filepath.Join(outDir, "alpha.csv")); got != "BODY-A"+sectionSeparator {
		t.Errorf("alpha.csv = %q, want %q", got, "BODY-A"+sectionSeparator)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
