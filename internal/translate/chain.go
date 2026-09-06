package translate

import (
	"context"
	"fmt"
	"log/slog"
)

// Chain tries its backends in order and returns the first success. If a backend
// fails it logs and moves to the next; if all fail it returns the last error.
type Chain struct {
	backends []Translator
	log      *slog.Logger
}

// NewChain builds a Chain from an explicit backend list (used in tests).
func NewChain(log *slog.Logger, backends ...Translator) *Chain {
	return &Chain{backends: backends, log: log}
}

func (c *Chain) Translate(ctx context.Context, text, from, to string) (string, error) {
	var err error
	for i, b := range c.backends {
		var out string
		out, err = b.Translate(ctx, text, from, to)
		if err == nil {
			return out, nil
		}
		if c.log != nil {
			c.log.Warn("translator backend failed, falling back",
				"position", i+1, "of", len(c.backends), "error", err)
		}
	}
	return "", fmt.Errorf("all %d translator backends failed, last error: %w", len(c.backends), err)
}
