package pipeline

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
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

// SameReplica mirrors the real tcoaal rule: empty tags never group.
func (tagGrouper) SameReplica(a, b extract.DataLine) bool { return a.Tag != "" && a.Tag == b.Tag }

// perFileParasitizer maps file path -> (tag -> translation).
type perFileParasitizer map[string]map[string]string

func (m perFileParasitizer) Replica(file, tag string) (string, error) { return m[file][tag], nil }

// maskAnalyzer implements markup.Masker: a trailing "~<markup>" is masked as §0§.
type maskAnalyzer struct{ wholeStringAnalyzer }

func (maskAnalyzer) Mask(ps *markup.PartialString) (string, []string) {
	text, marker, ok := strings.Cut(ps.Parts[0].Value, "~")
	if !ok {
		return ps.Parts[0].Value, nil
	}
	return text + "§0§", []string{marker}
}

func TestPipelineParasitizingMultiFileAndFallback(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "out")
	// p1: covered by file B only. p2: covered by both -> A wins. p3: neither
	// (and is last -> also checks the fixed last-group bug).
	mustWrite(t, filepath.Join(srcDir, "d.txt"),
		"p1\talpha\np1\tbeta\np2\tgamma\np3\tdelta\n")

	fileA, fileB := "A", "B"
	p, err := New(Deps{
		Analyzer:   wholeStringAnalyzer{},
		Format:     taggedTabFormat{},
		Translator: prefixTranslator{},
		Grouper:    tagGrouper{},
		Parasitizer: perFileParasitizer{
			fileA: {"p2": "afromtwo"},
			fileB: {"p1": "bfromone", "p2": "bfromtwo"},
		},
		Options: Options{
			SourceFolder:      srcDir,
			DestFolder:        dstDir,
			SourceLang:        "ru",
			TargetLang:        "be",
			Delimiter:         "\t",
			MultiRowReplicas:  true,
			Parasitizing:      true,
			ParasitizingFiles: []string{fileA, fileB},
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
	if !strings.Contains(got, "bfromone") {
		t.Errorf("p1 not sourced from file B: %q", got)
	}
	if !strings.Contains(got, "afromtwo") || strings.Contains(got, "bfromtwo") {
		t.Errorf("p2 should come from file A, not B: %q", got)
	}
	if !strings.Contains(got, "p3\tt:delta") {
		t.Errorf("p3 not machine-translated (last-group bug?): %q", got)
	}
}

// sentinelCorrupter echoes "T:"+text but strips digits from §i§ sentinels,
// simulating a model that mangled them.
type sentinelCorrupter struct{}

func (sentinelCorrupter) Translate(_ context.Context, text, _, _ string) (string, error) {
	return "T:" + regexpDigitsInSentinel.ReplaceAllString(text, "§§"), nil
}

var regexpDigitsInSentinel = regexp.MustCompile(`§\d+§`)

func TestPipelineMaskerFallsBackOnCorruptedSentinels(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "out")
	mustWrite(t, filepath.Join(srcDir, "d.txt"), "a\tHello there~END\n")

	p, err := New(Deps{
		Analyzer:   maskAnalyzer{},
		Format:     taggedTabFormat{},
		Translator: sentinelCorrupter{},
		Options: Options{
			SourceFolder: srcDir, DestFolder: dstDir,
			SourceLang: "en", TargetLang: "be", Delimiter: "\t",
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
	if strings.Contains(got, "§") {
		t.Errorf("corrupted sentinel leaked (no fallback?): %q", got)
	}
	if !strings.Contains(got, "Hello there") {
		t.Errorf("fallback lost the text: %q", got)
	}
}

func TestPipelineConcurrentReplicas(t *testing.T) {
	srcDir := t.TempDir()
	var b strings.Builder
	for i := 0; i < 200; i++ {
		fmt.Fprintf(&b, "r%d\tword%d\n", i, i) // 200 single-row replicas
	}
	mustWrite(t, filepath.Join(srcDir, "d.txt"), b.String())

	run := func(conc int) string {
		out := filepath.Join(t.TempDir(), "out")
		p, err := New(Deps{
			Analyzer: wholeStringAnalyzer{}, Format: taggedTabFormat{},
			Translator: prefixTranslator{}, Grouper: tagGrouper{},
			Options: Options{
				SourceFolder: srcDir, DestFolder: out,
				SourceLang: "ru", TargetLang: "be", Delimiter: "\t",
				MultiRowReplicas: true, Concurrency: conc,
			},
			Logger: discardLogger(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := p.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
		return mustRead(t, filepath.Join(out, "d.txt"))
	}

	if seq, par := run(1), run(8); seq != par {
		t.Errorf("concurrency changed the output:\n seq: %q\n par: %q", seq, par)
	}
}

// batchEcho implements translate.BatchTranslator; counts single vs batch calls.
type batchEcho struct {
	mu           sync.Mutex
	singleCalls  int
	batchCalls   int
	batchedItems int
}

func (b *batchEcho) Translate(_ context.Context, text, _, _ string) (string, error) {
	b.mu.Lock()
	b.singleCalls++
	b.mu.Unlock()
	return "T:" + text, nil
}
func (b *batchEcho) TranslateBatch(_ context.Context, texts []string, _, _ string) ([]string, error) {
	b.mu.Lock()
	b.batchCalls++
	b.batchedItems += len(texts)
	b.mu.Unlock()
	out := make([]string, len(texts))
	for i, s := range texts {
		out[i] = "T:" + s
	}
	return out, nil
}

func TestPipelineBatchedMaskerDeterministic(t *testing.T) {
	srcDir := t.TempDir()
	var body strings.Builder
	for i := 0; i < 700; i++ { // > 2 chunks of 256
		fmt.Fprintf(&body, "r%d\tline %d here~M%d\n", i, i, i)
	}
	mustWrite(t, filepath.Join(srcDir, "d.txt"), body.String())

	run := func(conc int) (string, *batchEcho) {
		out := filepath.Join(t.TempDir(), "out")
		be := &batchEcho{}
		p, err := New(Deps{
			Analyzer: maskAnalyzer{}, Format: taggedTabFormat{},
			Translator: be, Grouper: tagGrouper{},
			Options: Options{
				SourceFolder: srcDir, DestFolder: out,
				SourceLang: "ru", TargetLang: "be", Delimiter: "\t",
				MultiRowReplicas: true, Concurrency: conc,
			},
			Logger: discardLogger(),
		})
		if err != nil {
			t.Fatal(err)
		}
		if err := p.Run(context.Background()); err != nil {
			t.Fatal(err)
		}
		return mustRead(t, filepath.Join(out, "d.txt")), be
	}

	seq, be1 := run(1)
	par, be8 := run(8)
	if seq != par {
		t.Errorf("concurrency changed batched output")
	}
	if be1.batchCalls == 0 || be1.singleCalls != 0 {
		t.Errorf("expected batched calls only, got batch=%d single=%d", be1.batchCalls, be1.singleCalls)
	}
	if be1.batchedItems != 700 || be8.batchedItems != 700 {
		t.Errorf("batched item count: seq=%d par=%d, want 700", be1.batchedItems, be8.batchedItems)
	}
	if be8.batchCalls < 3 { // 700 items / 256 => 3 chunks
		t.Errorf("expected >=3 chunks, got %d", be8.batchCalls)
	}
}

func TestPipelineCarryOverUsedVerbatim(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "out")
	// c1 was translated in a previous run; p1 is new.
	mustWrite(t, filepath.Join(srcDir, "d.txt"), "c1\told line~M\np1\tnew line~M\n")

	prev, srcParasite := "PREV", "SRC"
	be := &batchEcho{}
	p, err := New(Deps{
		Analyzer:   maskAnalyzer{},
		Format:     taggedTabFormat{},
		Translator: be,
		Grouper:    tagGrouper{},
		Parasitizer: perFileParasitizer{
			prev:        {"c1": "верш з мінулага"},
			srcParasite: {"p1": "novaja krynica"},
		},
		Options: Options{
			SourceFolder: srcDir, DestFolder: dstDir,
			SourceLang: "ru", TargetLang: "be", Delimiter: "\t",
			MultiRowReplicas:  true,
			Parasitizing:      true,
			CarryOverFiles:    []string{prev},
			ParasitizingFiles: []string{srcParasite},
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
	if !strings.Contains(got, "c1\tверш з мінулага\n") {
		t.Errorf("c1 not carried over verbatim: %q", got)
	}
	if !strings.Contains(got, "p1\tt:novaja krynica\n") { // translated from its source parasite
		t.Errorf("p1 not translated: %q", got)
	}
	if be.batchedItems != 1 {
		t.Errorf("carried replica should not be sent to the translator; batched %d items", be.batchedItems)
	}
}

func TestPipelineMaskerPath(t *testing.T) {
	srcDir := t.TempDir()
	dstDir := filepath.Join(t.TempDir(), "out")
	mustWrite(t, filepath.Join(srcDir, "d.txt"),
		"a\tHello there~\\c[0]\nb\tone\nb\ttwo\n")

	p, err := New(Deps{
		Analyzer:   maskAnalyzer{},
		Format:     taggedTabFormat{},
		Translator: prefixTranslator{}, // echoes "T:" + input, sentinels intact
		Grouper:    tagGrouper{},
		Options: Options{
			SourceFolder:     srcDir,
			DestFolder:       dstDir,
			SourceLang:       "en",
			TargetLang:       "be",
			Delimiter:        "\t",
			MultiRowReplicas: true,
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
	// row a: whole string translated once, marker \c[0] spliced back, no leftover §.
	if !strings.Contains(got, "a\tT:Hello there\\c[0]\n") {
		t.Errorf("row a masker output wrong: %q", got)
	}
	if strings.Contains(got, "§") {
		t.Errorf("leftover sentinel in output: %q", got)
	}
	// replica b: whole translation on the first row, second row blanked.
	// source "onetwo" is lower-case, so the translation's first letter is too.
	lines := strings.Split(strings.TrimRight(got, "\n"), "\n")
	if len(lines) != 3 || lines[1] != "b\tt:onetwo" || lines[2] != "b\t" {
		t.Errorf("replica not collapsed onto first row: %q", got)
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
