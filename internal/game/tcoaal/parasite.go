package tcoaal

import (
	"fmt"
	"os"
	"strings"
	"sync"

	"github.com/Thrapis/go-csv-translator/internal/extract/tcoaalcsv"
)

// parasiteSource resolves an existing human translation for a replica id from
// one or more parasite files, each parsed once into an id -> text map. Both
// layouts are supported: the TXT dump ("#id (Speaker)" blocks with ": text"
// lines, plus "#id : text" one-liners for speakers/items) and the combined
// dialogue.csv (the Translation column, keyed by row id).
type parasiteSource struct {
	mu    sync.Mutex
	cache map[string]map[string]string // path -> (bare id -> translation)
}

func (p *parasiteSource) load(path string) (map[string]string, error) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if m, ok := p.cache[path]; ok {
		return m, nil
	}

	var (
		m   map[string]string
		err error
	)
	if strings.HasSuffix(strings.ToLower(path), ".csv") {
		m, err = loadCSVParasite(path)
	} else {
		m, err = loadTXTParasite(path)
	}
	if err != nil {
		return nil, err
	}

	if p.cache == nil {
		p.cache = map[string]map[string]string{}
	}
	p.cache[path] = m
	return m, nil
}

// replica returns the human translation for the id portion of tag ("id" or
// "#id" or "id,Source"), or "" when the file has no entry.
func (p *parasiteSource) replica(file, tag string) (string, error) {
	m, err := p.load(file)
	if err != nil {
		return "", err
	}
	id := strings.TrimPrefix(strings.Split(tag, ",")[0], "#")
	if id == "" {
		return "", nil
	}
	return m[id], nil
}

func loadCSVParasite(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, fmt.Errorf("read parasite file %s: %w", path, err)
	}
	defer f.Close()

	doc, err := tcoaalcsv.ParseDoc(f)
	if err != nil {
		return nil, fmt.Errorf("parse parasite file %s: %w", path, err)
	}
	return doc.TranslationsByID(), nil
}

func loadTXTParasite(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read parasite file %s: %w", path, err)
	}
	lines := strings.Split(strings.ReplaceAll(string(raw), "\r\n", "\n"), "\n")

	out := map[string]string{}
	for i := 0; i < len(lines); i++ {
		line := lines[i]
		if !strings.HasPrefix(line, "#") {
			continue
		}
		head, rest, hasColon := strings.Cut(line, " : ")
		id := strings.TrimPrefix(strings.Fields(head)[0], "#")

		// "#id : value" one-liner (speakers, items, single choices).
		if hasColon && !strings.Contains(head, "(") {
			if _, exists := out[id]; !exists {
				out[id] = rest
			}
			continue
		}

		// "#id (Speaker)" block header: gather the following ": " lines.
		var b strings.Builder
		for j := i + 1; j < len(lines); j++ {
			l := lines[j]
			if l == "" || strings.HasPrefix(l, "#") || strings.HasPrefix(l, "[") {
				break
			}
			if !strings.HasPrefix(l, ":") {
				continue
			}
			v := strings.TrimPrefix(strings.TrimPrefix(l, ":"), " ")
			if b.Len() != 0 {
				b.WriteByte(' ')
			}
			b.WriteString(v)
		}
		if _, exists := out[id]; !exists {
			out[id] = b.String()
		}
	}
	return out, nil
}
