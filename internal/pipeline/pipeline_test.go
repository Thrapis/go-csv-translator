package pipeline

import (
	"context"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/Thrapis/go-csv-translator/internal/extract"
	"github.com/Thrapis/go-csv-translator/internal/markup"
)

// --- test doubles -----------------------------------------------------------

type wholeStringAnalyzer struct{}

func (wholeStringAnalyzer) Analyze(text string) *markup.PartialString {
	return &markup.PartialString{Parts: []*markup.StringPart{{Value: text}}}
}
func (wholeStringAnalyzer) Translatable(ps *markup.PartialString) []*markup.StringPart {
	return ps.Parts
}
func (wholeStringAnalyzer) Render(ps *markup.PartialString) string {
	var b strings.Builder
	for _, p := range ps.Parts {
		b.WriteString(p.Value)
	}
	return b.String()
}

type prefixTranslator struct{}

func (prefixTranslator) Translate(_ context.Context, text, _, _ string) (string, error) {
	return "T:" + text, nil
}

// tabFormat reads/writes "key\tvalue" lines.
type tabFormat struct{}

func (tabFormat) Extract(r io.Reader, _ string) ([]extract.DataLine, *extract.Settings, error) {
	b, err := io.ReadAll(r)
	if err != nil {
		return nil, nil, err
	}
	var lines []extract.DataLine
	for _, ln := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		if ln == "" {
			continue
		}
		k, v, _ := strings.Cut(ln, "\t")
		lines = append(lines, extract.DataLine{Key: k, Value: v})
	}
	return lines, &extract.Settings{LineDelimiter: "\n"}, nil
}
func (tabFormat) Compose(w io.Writer, lines []extract.DataLine, s *extract.Settings, _ string) error {
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l.Key + "\t" + l.Value + s.LineDelimiter)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

func discardLogger() *slog.Logger { return slog.New(slog.NewTextHandler(io.Discard, nil)) }

// --- tests ----------------------------------------------------------------

func TestPipelineTranslatesTreeAndRenamesFolders(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "out")

	mustWrite(t, filepath.Join(srcDir, "a.txt"), "greeting\tHello\nfarewell\tBye\n")
	if err := os.MkdirAll(filepath.Join(srcDir, "en"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWrite(t, filepath.Join(srcDir, "en", "b.txt"), "x\tWorld\n")

	p, err := New(Deps{
		Analyzer:   wholeStringAnalyzer{},
		Format:     tabFormat{},
		Translator: prefixTranslator{},
		Options: Options{
			SourceFolder:  srcDir,
			DestFolder:    dstDir,
			FolderNameMap: map[string]string{"en": "be"},
			SourceLang:    "en",
			TargetLang:    "be",
			Delimiter:     "\t",
		},
		Logger: discardLogger(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	if got := mustRead(t, filepath.Join(dstDir, "a.txt")); got != "greeting\tT:Hello\nfarewell\tT:Bye\n" {
		t.Errorf("a.txt = %q", got)
	}
	if got := mustRead(t, filepath.Join(dstDir, "be", "b.txt")); got != "x\tT:World\n" {
		t.Errorf("dst/be/b.txt = %q", got)
	}
}

func TestPipelineStopsOnCancelledContext(t *testing.T) {
	srcDir := t.TempDir()
	mustWrite(t, filepath.Join(srcDir, "a.txt"), "k\tv\n")

	p, err := New(Deps{
		Analyzer:   wholeStringAnalyzer{},
		Format:     tabFormat{},
		Translator: prefixTranslator{},
		Options: Options{
			SourceFolder: srcDir,
			DestFolder:   filepath.Join(t.TempDir(), "out"),
			SourceLang:   "en",
			TargetLang:   "be",
			Delimiter:    "\t",
		},
		Logger: discardLogger(),
	})
	if err != nil {
		t.Fatal(err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	if err := p.Run(ctx); err != context.Canceled {
		t.Fatalf("want context.Canceled, got %v", err)
	}
}

// taggedTabFormat reads/writes "tag\tvalue" lines, exposing tag as DataLine.Tag.
type taggedTabFormat struct{}

func (taggedTabFormat) Extract(r io.Reader, _ string) ([]extract.DataLine, *extract.Settings, error) {
	b, _ := io.ReadAll(r)
	var lines []extract.DataLine
	for _, ln := range strings.Split(strings.TrimRight(string(b), "\n"), "\n") {
		if ln == "" {
			continue
		}
		tag, v, _ := strings.Cut(ln, "\t")
		lines = append(lines, extract.DataLine{Key: tag, Value: v, Tag: tag})
	}
	return lines, &extract.Settings{LineDelimiter: "\n"}, nil
}
func (taggedTabFormat) Compose(w io.Writer, lines []extract.DataLine, s *extract.Settings, _ string) error {
	var b strings.Builder
	for _, l := range lines {
		b.WriteString(l.Key + "\t" + l.Value + s.LineDelimiter)
	}
	_, err := io.WriteString(w, b.String())
	return err
}

type tagGrouper struct{}

func (tagGrouper) SameReplica(a, b extract.DataLine) bool { return a.Tag == b.Tag }

type mapParasitizer map[string]string

func (m mapParasitizer) Replica(_ string, tag string) (string, error) { return m[tag], nil }

func TestPipelineParasitizingFallsBackToMT(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "out")
	// p1: two rows, covered by the parasite file. p2: one row, not covered.
	// z: trailing group so p2 is not last (see the known last-group bug).
	mustWrite(t, filepath.Join(srcDir, "d.txt"),
		"p1\talpha\np1\tbeta\np2\tgamma\nz\tzzz\n")

	p, err := New(Deps{
		Analyzer:    wholeStringAnalyzer{},
		Format:      taggedTabFormat{},
		Translator:  prefixTranslator{},
		Grouper:     tagGrouper{},
		Parasitizer: mapParasitizer{"p1": "human one two"},
		Options: Options{
			SourceFolder:     srcDir,
			DestFolder:       dstDir,
			SourceLang:       "ru",
			TargetLang:       "be",
			Delimiter:        "\t",
			MultiRowReplicas: true,
			Parasitizing:     true,
			ParasitizingFile: "unused",
		},
		Logger: discardLogger(),
	})
	if err != nil {
		t.Fatal(err)
	}
	if err := p.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	got := mustRead(t, filepath.Join(dstDir, "d.txt"))
	// p1 rows come from the parasite text ("human one two" translated, then
	// split across the 2 rows); p2 falls back to MT of its own source "gamma".
	// (translateParts lower-cases the first letter to match the source word.)
	if !strings.Contains(got, "p1\tt:human") {
		t.Errorf("p1 not sourced from parasite: %q", got)
	}
	if !strings.Contains(got, "p2\tt:gamma") {
		t.Errorf("p2 did not fall back to MT of source: %q", got)
	}
}

func mustWrite(t *testing.T, path, body string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(body), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	b, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
