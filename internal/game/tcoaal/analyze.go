package tcoaal

import (
	"regexp"

	"github.com/Thrapis/go-csv-translator/internal/markup"
)

// Part types produced by this analyzer.
const (
	typeString = iota
	typeStyleSymbol
	typeFontSizeSymbol
	typeColorSymbol
	typeQuotesSymbol
	typeBracketSymbol
	typeSpamSymbol
)

const (
	styleSymbolPattern    = `(\\f[ibr]{1})`
	fontSizeSymbolPattern = `(\\[\{\}]{1})`
	colorSymbolPattern    = `(\\c\[[0-9]{1}\])`
	quotesSymbolPattern   = `([\"]{1})`
	bracketSymbolPattern  = `(([\(]|[\)]){1})`
	spamSymbolPattern     = `(([\.]|[\?]|[\!]|(\.\s){1,}){2,})`
)

type patternType struct {
	re  *regexp.Regexp
	typ int
}

// symbolPatterns is the ordered list of non-text constructs. It is shared by
// segmentation (Analyze) and classification (detectPart).
var symbolPatterns = []patternType{
	{regexp.MustCompile(styleSymbolPattern), typeStyleSymbol},
	{regexp.MustCompile(fontSizeSymbolPattern), typeFontSizeSymbol},
	{regexp.MustCompile(colorSymbolPattern), typeColorSymbol},
	{regexp.MustCompile(quotesSymbolPattern), typeQuotesSymbol},
	{regexp.MustCompile(bracketSymbolPattern), typeBracketSymbol},
	{regexp.MustCompile(spamSymbolPattern), typeSpamSymbol},
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
	var b []byte
	for _, p := range ps.Parts {
		b = append(b, p.Value...)
	}
	return string(b)
}
