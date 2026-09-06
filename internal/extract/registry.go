package extract

import (
	"fmt"
	"sort"
)

var formats = map[string]func() Format{}

// Register makes a Format available under id. It is meant to be called from a
// format package's init. It panics on a duplicate id, which can only be a
// programming error.
func Register(id string, factory func() Format) {
	if _, dup := formats[id]; dup {
		panic(fmt.Sprintf("extract: format %q registered twice", id))
	}
	formats[id] = factory
}

// Get returns a fresh Format for id.
func Get(id string) (Format, error) {
	factory, ok := formats[id]
	if !ok {
		return nil, fmt.Errorf("extract: unknown format %q (known: %v)", id, Names())
	}
	return factory(), nil
}

// Names lists the registered format ids, sorted.
func Names() []string {
	out := make([]string, 0, len(formats))
	for id := range formats {
		out = append(out, id)
	}
	sort.Strings(out)
	return out
}
