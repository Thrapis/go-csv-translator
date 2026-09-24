package tcoaal

import (
	"testing"

	"github.com/Thrapis/go-csv-translator/internal/extract"
)

func TestSameReplica(t *testing.T) {
	g := New()
	dl := func(tag string) extract.DataLine { return extract.DataLine{Tag: tag} }

	if !g.SameReplica(dl("abc"), dl("abc")) {
		t.Error("same non-empty tag should group")
	}
	if g.SameReplica(dl("abc"), dl("xyz")) {
		t.Error("different tags should not group")
	}
	// Header-section rows carry no tag; they must each stand alone, not merge
	// into one giant replica.
	if g.SameReplica(dl(""), dl("")) {
		t.Error("empty tags must not group")
	}
}
