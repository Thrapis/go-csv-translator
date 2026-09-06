package taleworld

import "testing"

func TestAnalyzeRoundTripPlainAndVariable(t *testing.T) {
	a := analyzer{}
	cases := []string{
		"You have arrived at the settlement.",
		"Welcome, {PLAYER_NAME}, to our humble town.",
		"{GOLD_AMOUNT} denars have been added.",
		"",
	}
	for _, in := range cases {
		if got := a.Render(a.Analyze(in)); got != in {
			t.Errorf("round trip mismatch\n in: %q\nout: %q", in, got)
		}
	}
}

func TestAnalyzeSegmentsVariables(t *testing.T) {
	a := analyzer{}
	ps := a.Analyze("Hello {NAME}, you owe {AMOUNT} denars.")

	var texts, vars int
	for _, p := range ps.Parts {
		switch p.Type {
		case typeString:
			texts++
		case typeVariable:
			vars++
		}
	}
	if vars != 2 {
		t.Errorf("expected 2 variable parts, got %d", vars)
	}
	if texts < 2 {
		t.Errorf("expected the plain text to be split around the variables, got %d text parts", texts)
	}
}

func TestTranslatableSkipsVariables(t *testing.T) {
	a := analyzer{}
	ps := a.Analyze("Hello {NAME}, welcome.")
	for _, p := range a.Translatable(ps) {
		if p.Type == typeVariable {
			t.Errorf("variable part %q must not be translatable", p.Value)
		}
	}
}
