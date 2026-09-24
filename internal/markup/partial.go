// Package markup models a localized string as an ordered sequence of parts,
// separating human-translatable text from engine control symbols (colour codes,
// variables, gender/ternary constructs, ...) so only the text is sent to a
// translator and the symbols are put back verbatim afterwards.
package markup

import "fmt"

// PartialString is a string decomposed into ordered parts by an Analyzer.
type PartialString struct {
	Parts []*StringPart
}

// StringPart is one fragment of a PartialString. Type is defined by the game
// package that produced it; Value holds leaf text, Parts holds children for
// composite constructs such as gender or ternary forms.
type StringPart struct {
	Type  int
	Value string
	Parts []*StringPart
}

// Print writes the tree to stdout; useful when debugging an Analyzer.
func (ps *PartialString) Print() {
	for _, p := range ps.Parts {
		p.Print()
	}
}

// Print writes this part and its children to stdout.
func (sp *StringPart) Print() {
	fmt.Printf("%d -> %q\n", sp.Type, sp.Value)
	for _, p := range sp.Parts {
		p.Print()
	}
}
