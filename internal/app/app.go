// Package app turns a validated config into a wired pipeline and runs it,
// including the semantic checks that need the plugin registries.
package app

import (
	"context"
	"fmt"
	"log/slog"

	"github.com/Thrapis/go-csv-translator/internal/config"
	"github.com/Thrapis/go-csv-translator/internal/extract"
	"github.com/Thrapis/go-csv-translator/internal/game"
	"github.com/Thrapis/go-csv-translator/internal/pipeline"
	"github.com/Thrapis/go-csv-translator/internal/translate"
)

// App is a configured, ready-to-run translation job.
type App struct {
	cfg  *config.Config
	log  *slog.Logger
	pipe *pipeline.Pipeline
}

// New resolves the game, format and translators named in cfg and builds the
// pipeline. It fails if any name is unknown or a requested capability (e.g.
// parasitizing) is not supported by the chosen game.
func New(cfg *config.Config, log *slog.Logger) (*App, error) {
	g, err := game.Get(cfg.Game)
	if err != nil {
		return nil, err
	}

	formatID := cfg.Source.Format
	if formatID == "" {
		formatID = g.DefaultFormat()
	}
	format, err := extract.Get(formatID)
	if err != nil {
		return nil, err
	}

	delimiter := cfg.Delimiter
	if delimiter == "" {
		delimiter = g.DefaultDelimiter()
	}

	translator, err := translate.Build(cfg.Translators, translate.Options{
		LingvanexAddress: cfg.Lingvanex.Address,
		LingvanexPort:    cfg.Lingvanex.Port,
	}, log)
	if err != nil {
		return nil, err
	}

	grouper, _ := g.(game.ReplicaGrouper)
	parasitizer, _ := g.(game.Parasitizer)
	if cfg.Parasitizing.Enabled && parasitizer == nil {
		return nil, fmt.Errorf("game %q does not support parasitizing", cfg.Game)
	}

	pipe, err := pipeline.New(pipeline.Deps{
		Analyzer:    g.Analyzer(),
		Format:      format,
		Translator:  translator,
		Grouper:     grouper,
		Parasitizer: parasitizer,
		Options: pipeline.Options{
			SourceFolder:      cfg.Source.Folder,
			DestFolder:        cfg.Destination.Folder,
			FileGlob:          cfg.Source.Files,
			FolderNameMap:     cfg.Destination.FolderNameMap,
			SourceLang:        cfg.Language.Source,
			TargetLang:        cfg.Language.Target,
			Delimiter:         delimiter,
			SkipFirstLine:     cfg.SkipFirstLine,
			MultiRowReplicas:  cfg.MultiRowReplicas,
			Parasitizing:      cfg.Parasitizing.Enabled,
			ParasitizingFiles: cfg.Parasitizing.Files,
			Concurrency:       cfg.Concurrency,
		},
		Logger: log,
	})
	if err != nil {
		return nil, err
	}

	log.Info("configured run",
		"game", g.Name(), "format", formatID, "delimiter", delimiter,
		"direction", cfg.Language.Source+"->"+cfg.Language.Target,
		"translators", cfg.Translators)

	return &App{cfg: cfg, log: log, pipe: pipe}, nil
}

// Run executes the pipeline.
func (a *App) Run(ctx context.Context) error {
	return a.pipe.Run(ctx)
}
