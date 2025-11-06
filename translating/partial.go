package translating

import "fmt"

type PartialString struct {
	Parts []*StringPart
}

type StringPart struct {
	Type int
	// String, Variable
	Value string
	// Gender, Ternary
	Parts []*StringPart
}

func (ps *PartialString) Print() {
	for _, v := range ps.Parts {
		v.Print()
	}
}

func (sp *StringPart) Print() {
	fmt.Printf("%d -> \"%s\"\n", sp.Type, sp.Value)
	for _, v := range sp.Parts {
		v.Print()
	}
}
