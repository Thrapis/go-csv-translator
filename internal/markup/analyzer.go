package markup

// Analyzer tokenizes a raw localized string into a PartialString and can turn it
// back into a string. Each supported game provides one implementation describing
// that engine's markup syntax.
//
// Contract: for any input s produced by the game, Render(Analyze(s)) == s, and
// mutating only the Value of the parts returned by Translatable before calling
// Render substitutes the translated text while keeping every control symbol.
type Analyzer interface {
	// Analyze splits text into ordered parts.
	Analyze(text string) *PartialString
	// Translatable returns the subset of leaf parts whose Value is free text
	// meant to be translated. The returned pointers alias the tree, so writing
	// to part.Value updates what Render emits.
	Translatable(ps *PartialString) []*StringPart
	// Render reassembles a (possibly mutated) PartialString into a string.
	Render(ps *PartialString) string
}
