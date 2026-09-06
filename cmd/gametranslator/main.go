// Command gametranslator machine-translates a game's localization files. All
// behaviour comes from a YAML config file:
//
//	gametranslator -config path/to/config.yaml
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"log/slog"
	"os"
	"os/signal"
	"syscall"

	"github.com/Thrapis/go-csv-translator/internal/app"
	lingvanexsrv "github.com/Thrapis/go-csv-translator/internal/backend/lingvanex"
	"github.com/Thrapis/go-csv-translator/internal/config"
	_ "github.com/Thrapis/go-csv-translator/internal/plugins" // register games, formats, backends
)

func main() { os.Exit(run()) }

func run() int {
	cfgPath := flag.String("config", "", "path to the YAML config file (required)")
	flag.Parse()

	if *cfgPath == "" {
		fmt.Fprintln(os.Stderr, "usage: gametranslator -config <path>")
		return 2
	}

	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	log := newLogger(cfg.LogLevel)

	server := lingvanexsrv.New(cfg.Lingvanex, log)
	if err := server.Start(ctx); err != nil {
		log.Error("startup failed", "error", err)
		return 1
	}
	defer func() {
		if err := server.Stop(); err != nil {
			log.Error("lingvanex stop failed", "error", err)
		}
	}()

	application, err := app.New(cfg, log)
	if err != nil {
		log.Error("configuration error", "error", err)
		return 1
	}

	if err := application.Run(ctx); err != nil {
		if errors.Is(err, context.Canceled) {
			log.Warn("interrupted; partial output left on disk")
			return 130
		}
		log.Error("run failed", "error", err)
		return 1
	}

	log.Info("done")
	return 0
}

func newLogger(level string) *slog.Logger {
	var lv slog.Level
	if err := lv.UnmarshalText([]byte(level)); err != nil {
		lv = slog.LevelInfo
	}
	return slog.New(slog.NewTextHandler(os.Stderr, &slog.HandlerOptions{Level: lv}))
}
