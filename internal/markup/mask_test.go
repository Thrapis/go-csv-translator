package markup

import "testing"

func TestUnmask(t *testing.T) {
	m := []string{`\c[2]`, `\c[0]`, `"`}

	if got := Unmask(`§0§Andy§1§: §2§hi§2§`, m); got != `\c[2]Andy\c[0]: "hi"` {
		t.Errorf("normal: %q", got)
	}
	// reordered sentinels
	if got := Unmask(`§2§hi§2§ §1§Andy§0§`, m); got != `"hi" \c[0]Andy\c[2]` {
		t.Errorf("reordered: %q", got)
	}
	// a dropped sentinel is appended so the markup is not lost
	if got := Unmask(`§0§Andy: hi`, m); got != `\c[2]Andy: hi\c[0]"` {
		t.Errorf("missing appended: %q", got)
	}
	// unknown index left untouched
	if got := Unmask(`§9§x`, m); got != `§9§x` {
		t.Errorf("unknown: %q", got)
	}
}

func TestSentinelsIntact(t *testing.T) {
	if !SentinelsIntact(`§0§Andy§1§: §2§hi§3§`, 4) {
		t.Error("clean string should be intact")
	}
	if SentinelsIntact(`§0§Andy§1§: hi`, 4) {
		t.Error("missing §2§/§3§ should not be intact")
	}
	if SentinelsIntact(`§0§Andy§§: hi§1§`, 2) {
		t.Error("stray §§ should not be intact")
	}
	if SentinelsIntact(`§0§Andy§0§`, 2) {
		t.Error("duplicate index / missing §1§ should not be intact")
	}
	if !SentinelsIntact(`plain text`, 0) {
		t.Error("n=0 with no sentinels is intact")
	}
}

func TestStripSentinels(t *testing.T) {
	if got := StripSentinels(`§0§ a §12§ b§3§`); got != ` a  b` {
		t.Errorf("got %q", got)
	}
}
