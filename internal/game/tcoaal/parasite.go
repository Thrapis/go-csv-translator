package tcoaal

import (
	"fmt"
	"os"
	"strings"
	"sync"
)

// parasiteSource lazily loads and caches a "parasite" dialogue file: a dump of
// an existing human translation where each replica is a block starting with a
// "#<ID>" line followed by ": <text>" lines, blocks separated by a blank line.
type parasiteSource struct {
	mu   sync.Mutex
	path string
	data string // newline-normalised contents
}

func (p *parasiteSource) load(path string) (string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.path == path && p.data != "" {
		return p.data, nil
	}
	raw, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read parasite file %s: %w", path, err)
	}
	p.path = path
	p.data = strings.ReplaceAll(string(raw), "\r\n", "\n")
	return p.data, nil
}

// replica returns the human translation for the replica identified by the ID
// portion of tag ("ID,Source"), joining its ": " lines with single spaces.
func (p *parasiteSource) replica(file, tag string) (string, error) {
	data, err := p.load(file)
	if err != nil {
		return "", err
	}

	// Tags arrive as a bare id (CSV format) or "#id" (TXT format); the parasite
	// file keys replicas as "#id".
	id := "#" + strings.TrimPrefix(strings.Split(tag, ",")[0], "#")
	start := strings.Index(data, id)
	if start < 0 {
		// Version drift between the source and the parasite file leaves some
		// ids unmatched; the pipeline falls back to machine translation.
		return "", nil
	}

	end := len(data)
	if rel := strings.Index(data[start:], "\n\n"); rel >= 0 {
		end = start + rel
	}

	lines := strings.Split(data[start:end], "\n")

	var b strings.Builder
	for _, line := range lines[1:] {
		if !strings.HasPrefix(line, ": ") {
			continue
		}
		if b.Len() != 0 {
			b.WriteByte(' ')
		}
		b.WriteString(strings.TrimPrefix(line, ": "))
	}
	return b.String(), nil
}
