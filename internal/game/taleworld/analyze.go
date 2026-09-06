package taleworld

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Thrapis/go-csv-translator/internal/markup"
)

// Part types produced by this analyzer.
const (
	typeString = iota
	typeVariable
	typeGender
	typeTernary
)

const (
	variablePattern = `^\{([a-zA-Z0-9_а-яА-ЯёЁ ]+)\}$`
	genderPattern   = `^\{([a-zA-Z0-9_а-яА-ЯёЁ ]+)\/([a-zA-Z0-9_а-яА-ЯёЁ ]+)\}$`
	ternaryPattern  = `^\{(\w+)\?(.*?)\:(.*?)\}$`
)

var (
	variableRe = regexp.MustCompile(variablePattern)
	genderRe   = regexp.MustCompile(genderPattern)
	ternaryRe  = regexp.MustCompile(ternaryPattern)
)

type analyzer struct{}

// Analyze walks the string tracking `{`/`}` nesting and emits a part for each
// top-level `{...}` group and each run of plain text between them.
func (analyzer) Analyze(text string) *markup.PartialString {
	result := &markup.PartialString{Parts: make([]*markup.StringPart, 0)}

	from := 0
	openBrackets := 0
	for i, char := range text {
		switch char {
		case '{':
			if openBrackets == 0 {
				if from != i {
					result.Parts = append(result.Parts, detectPart(text[from:i]))
				}
				from = i
			}
			openBrackets++
		case '}':
			if openBrackets == 1 {
				if from != i {
					result.Parts = append(result.Parts, detectPart(text[from:i+1]))
				}
				from = i + 1
			}
			openBrackets--
		}
	}
	if from <= len(text)-1 {
		result.Parts = append(result.Parts, detectPart(text[from:]))
	}

	return result
}

func detectPart(text string) *markup.StringPart {
	switch {
	case variableRe.MatchString(text):
		return &markup.StringPart{Type: typeVariable, Value: variableRe.FindStringSubmatch(text)[1]}
	case genderRe.MatchString(text):
		g := genderRe.FindStringSubmatch(text)
		return &markup.StringPart{Type: typeGender, Parts: []*markup.StringPart{
			detectPart(g[1]), detectPart(g[2]),
		}}
	case ternaryRe.MatchString(text):
		g := ternaryRe.FindStringSubmatch(text)
		return &markup.StringPart{Type: typeTernary, Parts: []*markup.StringPart{
			{Type: typeVariable, Value: g[1]},
			detectPart(g[2]),
			detectPart(g[3]),
		}}
	default:
		return &markup.StringPart{Type: typeString, Value: text}
	}
}

func (analyzer) Translatable(ps *markup.PartialString) []*markup.StringPart {
	out := make([]*markup.StringPart, 0, len(ps.Parts))
	for _, p := range ps.Parts {
		out = append(out, translatablePart(p)...)
	}
	return out
}

func translatablePart(sp *markup.StringPart) []*markup.StringPart {
	switch sp.Type {
	case typeString:
		return []*markup.StringPart{sp}
	case typeGender:
		return sp.Parts
	case typeTernary:
		out := translatablePart(sp.Parts[1])
		return append(out, translatablePart(sp.Parts[2])...)
	default:
		return nil
	}
}

func (analyzer) Render(ps *markup.PartialString) string {
	var b strings.Builder
	for _, p := range ps.Parts {
		b.WriteString(renderPart(p))
	}
	return b.String()
}

func renderPart(sp *markup.StringPart) string {
	switch sp.Type {
	case typeVariable:
		return fmt.Sprintf("{%s}", sp.Value)
	case typeGender:
		return fmt.Sprintf("{%v/%v}", sp.Parts[0], sp.Parts[1])
	case typeTernary:
		return fmt.Sprintf("{%s?%v:%v}", sp.Parts[0].Value, sp.Parts[1], sp.Parts[2])
	default:
		return sp.Value
	}
}
