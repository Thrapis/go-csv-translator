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
