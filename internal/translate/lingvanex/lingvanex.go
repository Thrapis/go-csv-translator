// Package lingvanex is a Translator backend that talks to the local Lingvanex
// HTTP server (see internal/backend/lingvanex for its process supervisor). It
// registers as "lingvanex".
package lingvanex

import (
	"context"
	"fmt"
	"io"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/Thrapis/go-csv-translator/internal/translate"
)

func init() {
	translate.Register("lingvanex", func(opts translate.Options, _ *slog.Logger) (translate.Translator, error) {
		addr := opts.LingvanexAddress
		if addr == "" {
			addr = "127.0.0.1"
		}
		port := opts.LingvanexPort
		if port == 0 {
			port = 8000
		}
		return &Client{
			baseURL: fmt.Sprintf("http://%s:%d", addr, port),
			http:    &http.Client{Timeout: 60 * time.Second},
		}, nil
	})
}

// Client posts text to the Lingvanex server and returns the plain-text body.
type Client struct {
	baseURL string
	http    *http.Client
}

func (c *Client) Translate(ctx context.Context, text, from, to string) (string, error) {
	return c.post(ctx, from, to, text)
}

// TranslateBatch sends all texts in one request (newline-delimited; the server
// translates each line and returns them the same way).
func (c *Client) TranslateBatch(ctx context.Context, texts []string, from, to string) ([]string, error) {
	if len(texts) == 0 {
		return nil, nil
	}
	for i, t := range texts {
		if strings.ContainsAny(t, "\r\n") {
			texts[i] = strings.NewReplacer("\r", " ", "\n", " ").Replace(t)
		}
	}
	body, err := c.post(ctx, from, to, strings.Join(texts, "\n"))
	if err != nil {
		return nil, err
	}
	lines := strings.Split(body, "\n")
	if len(lines) != len(texts) {
		return nil, fmt.Errorf("lingvanex: sent %d lines, got %d back", len(texts), len(lines))
	}
	return lines, nil
}

func (c *Client) post(ctx context.Context, from, to, payload string) (string, error) {
	url := fmt.Sprintf("%s?from=%s&to=%s", c.baseURL, from, to)
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, strings.NewReader(payload))
	if err != nil {
		return "", err
	}

	res, err := c.http.Do(req)
	if err != nil {
		return "", err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if res.StatusCode != http.StatusOK {
		return "", fmt.Errorf("lingvanex: status %d: %s", res.StatusCode, body)
	}
	return string(body), nil
}
