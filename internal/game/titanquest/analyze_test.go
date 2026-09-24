package titanquest

import "testing"

func TestAnalyzeRoundTrip(t *testing.T) {
	a := analyzer{}
	cases := []string{
		"Plain description text.",
		"Deals {^r}fire damage{^n} over time.",
		"Requires [Level 10] to equip.",
		"Bonus for %s1 seconds.",
		"{^o}Legendary{^n} [Set Item] with %d charges",
		"",
	}
	for _, in := range cases {
		if got := a.Render(a.Analyze(in)); got != in {
			t.Errorf("round trip mismatch\n in: %q\nout: %q", in, got)
		}
	}
}

func TestTranslatableOnlyText(t *testing.T) {
	a := analyzer{}
	ps := a.Analyze("Deals {^r}fire{^n} damage for %s1 turns")
	for _, p := range a.Translatable(ps) {
		if p.Type != typeString {
			t.Errorf("translatable part has non-string type %d (%q)", p.Type, p.Value)
		}
	}
}
