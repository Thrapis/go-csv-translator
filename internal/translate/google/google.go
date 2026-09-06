// Package google is a Translator backend wrapping the unofficial free Google
// Translate endpoint. It registers as "google" and is typically used only as a
// fallback behind lingvanex.
package google

import (
	"context"
	"log/slog"

	"github.com/Thrapis/go-csv-translator/internal/translate"

	translategooglefree "github.com/bas24/googletranslatefree"
)

func init() {
	translate.Register("google", func(_ translate.Options, _ *slog.Logger) (translate.Translator, error) {
		return Client{}, nil
	})
}

// Client is stateless.
type Client struct{}

func (Client) Translate(ctx context.Context, text, from, to string) (string, error) {
	if err := ctx.Err(); err != nil {
		return "", err
	}
	// The underlying library is not context-aware and performs a blocking HTTP
	// call; the ctx check above is a best effort for cancellation between calls.
	return translategooglefree.Translate(text, from, to)
}
