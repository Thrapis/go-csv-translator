// Package pipeline is the translation orchestrator: it walks the source tree,
// runs each file through extract -> analyze -> translate -> render -> compose,
// and writes the result into the destination tree.
package pipeline

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"github.com/Thrapis/go-csv-translator/internal/extract"
	"github.com/Thrapis/go-csv-translator/internal/game"
	"github.com/Thrapis/go-csv-translator/internal/markup"
	"github.com/Thrapis/go-csv-translator/internal/translate"

	"log/slog"
)

// Options are the per-run knobs, mapped from config by internal/app.
type Options struct {
	SourceFolder     string
	DestFolder       string
	FileGlob         string // base-name glob; "" means every file
	FolderNameMap    map[string]string
	SourceLang       string
	TargetLang       string
	Delimiter        string
	SkipFirstLine    bool
	MultiRowReplicas bool
	Parasitizing     bool
	ParasitizingFile string
}

// Pipeline holds everything one run needs. Build it with New.
type Pipeline struct {
	analyzer    markup.Analyzer
	format      extract.Format
	translator  translate.Translator
	grouper     game.ReplicaGrouper // nil when the game has no replica grouping
	parasitizer game.Parasitizer    // nil unless Options.Parasitizing
	opts        Options
	log         *slog.Logger
}

// Deps are the collaborators Pipeline needs; grouper and parasitizer may be nil.
type Deps struct {
	Analyzer    markup.Analyzer
	Format      extract.Format
	Translator  translate.Translator
	Grouper     game.ReplicaGrouper
	Parasitizer game.Parasitizer
	Options     Options
	Logger      *slog.Logger
}

// New validates deps and returns a ready Pipeline.
func New(d Deps) (*Pipeline, error) {
	switch {
	case d.Analyzer == nil:
		return nil, fmt.Errorf("pipeline: analyzer is required")
	case d.Format == nil:
		return nil, fmt.Errorf("pipeline: format is required")
	case d.Translator == nil:
		return nil, fmt.Errorf("pipeline: translator is required")
	case d.Logger == nil:
		return nil, fmt.Errorf("pipeline: logger is required")
	}
	if d.Options.MultiRowReplicas && d.Options.Parasitizing && d.Parasitizer == nil {
		return nil, fmt.Errorf("pipeline: parasitizing enabled but game provides no parasitizer")
	}
	return &Pipeline{
		analyzer:    d.Analyzer,
		format:      d.Format,
		translator:  d.Translator,
		grouper:     d.Grouper,
		parasitizer: d.Parasitizer,
		opts:        d.Options,
		log:         d.Logger,
	}, nil
}

// Run translates the whole source tree into the destination tree. It stops early
// if ctx is cancelled.
func (p *Pipeline) Run(ctx context.Context) error {
	root, err := scanFolder(p.opts.SourceFolder)
	if err != nil {
		return err
	}
	return p.translateFolder(ctx, root, p.opts.DestFolder)
}

func (p *Pipeline) translateFolder(ctx context.Context, src *node, dst string) error {
	if err := os.MkdirAll(dst, os.ModePerm); err != nil {
		return fmt.Errorf("create %s: %w", dst, err)
	}

	for _, f := range src.files {
		if err := ctx.Err(); err != nil {
			return err
		}
		if !p.matchesGlob(f.name) {
			continue
		}
		if err := p.translateFile(ctx, f, dst); err != nil {
			return fmt.Errorf("%s: %w", f.path, err)
		}
	}

	for i := range src.folders {
		sub := &src.folders[i]
		name := sub.name
		if mapped, ok := p.opts.FolderNameMap[name]; ok {
			name = mapped
		}
		if err := p.translateFolder(ctx, sub, filepath.Join(dst, name)); err != nil {
			return err
		}
	}
	return nil
}

func (p *Pipeline) matchesGlob(name string) bool {
	if p.opts.FileGlob == "" {
		return true
	}
	ok, _ := filepath.Match(p.opts.FileGlob, name)
	return ok
}

func (p *Pipeline) translateFile(ctx context.Context, f fileRef, dstDir string) error {
	in, err := os.Open(f.path)
	if err != nil {
		return err
	}
	defer in.Close()

	lines, settings, err := p.format.Extract(in, p.opts.Delimiter)
	if err != nil {
		return err
	}
	if p.opts.SkipFirstLine && len(lines) > 0 {
		lines = lines[1:]
	}

	if p.opts.MultiRowReplicas {
		err = p.translateReplicas(ctx, lines)
	} else {
		err = p.translateRows(ctx, lines)
	}
	if err != nil {
		return err
	}

	outPath := filepath.Join(dstDir, f.name)
	out, err := os.OpenFile(outPath, os.O_RDWR|os.O_TRUNC|os.O_CREATE, os.ModePerm)
	if err != nil {
		return err
	}
	defer out.Close()

	return p.format.Compose(out, lines, settings, p.opts.Delimiter)
}
