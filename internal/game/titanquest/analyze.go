package titanquest

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/Thrapis/go-csv-translator/internal/markup"
)

// Part types produced by this analyzer.
const (
	typeString = iota
	typeSpecialCurve
	typeSpecialSquare
	typeVariable
)

const (
	specialCurvePattern  = `\{([\^\%\.\:\+a-zA-Z0-9_ ]+)\}`
	specialSquarePattern = `\[([\^\%\.\:\+a-zA-Z0-9_ ]+)\]`
	variablePattern      = `(\%[a-zA-Z0-9_])`
)

type patternType struct {
	re  *regexp.Regexp
	typ int
}

var symbolPatterns = []patternType{
	{regexp.MustCompile(specialCurvePattern), typeSpecialCurve},
	{regexp.MustCompile(specialSquarePattern), typeSpecialSquare},
	{regexp.MustCompile(variablePattern), typeVariable},
}

type analyzer struct{}

func (analyzer) Analyze(text string) *markup.PartialString {
	result := &markup.PartialString{Parts: make([]*markup.StringPart, 0)}

	focus := text
	for len(focus) > 0 {
		firstIndex := len(focus)
		lastIndex := len(focus)

		for _, pt := range symbolPatterns {
			loc := pt.re.FindStringIndex(focus)
			if loc != nil && loc[0] < firstIndex {
				firstIndex, lastIndex = loc[0], loc[1]
			}
		}

		if firstIndex == len(focus) {
			result.Parts = append(result.Parts, detectPart(focus))
			break
		}

		before := focus[:firstIndex]
		target := focus[firstIndex:lastIndex]
		after := focus[lastIndex:]

		if len(before) > 0 {
			result.Parts = append(result.Parts, detectPart(before))
		}
		result.Parts = append(result.Parts, detectPart(target))

		if len(after) == 0 {
			break
		}
		focus = after
	}

	return result
}

func detectPart(text string) *markup.StringPart {
	for _, pt := range symbolPatterns {
		if pt.re.MatchString(text) {
			return &markup.StringPart{Type: pt.typ, Value: pt.re.FindStringSubmatch(text)[1]}
		}
	}
	return &markup.StringPart{Type: typeString, Value: text}
}

func (analyzer) Translatable(ps *markup.PartialString) []*markup.StringPart {
	out := make([]*markup.StringPart, 0, len(ps.Parts))
	for _, p := range ps.Parts {
		if p.Type == typeString {
			out = append(out, p)
		}
	}
	return out
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
	case typeSpecialCurve:
		return fmt.Sprintf("{%s}", sp.Value)
	case typeSpecialSquare:
		return fmt.Sprintf("[%s]", sp.Value)
	case typeVariable:
		return sp.Value
	default:
		return sp.Value
	}
}
