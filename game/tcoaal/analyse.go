package tcoaal

import (
	"regexp"

	"tw-translator/translating"
)

const (
	styleSymbolPattern    = `(\\f[ibr]{1})`
	fontSizeSymbolPattern = `(\\[\{\}]{1})`
	colorSymbolPattern    = `(\\c\[[0-9]{1}\])`
	quotesSymbolPattern   = `([\"]{1})`
	spamSymbolPattern     = `([\.]{4,})`
)

func Analyse(text string) *translating.PartialString {
	regexes := []*regexp.Regexp{
		regexp.MustCompile(styleSymbolPattern),
		regexp.MustCompile(fontSizeSymbolPattern),
		regexp.MustCompile(colorSymbolPattern),
		regexp.MustCompile(quotesSymbolPattern),
		regexp.MustCompile(spamSymbolPattern),
	}

	result := &translating.PartialString{
		Parts: make([]*translating.StringPart, 0),
	}

	focusString := text

	for {
		if len(focusString) == 0 {
			break
		}

		fistIndex := len(focusString)
		lastIndex := len(focusString)

		for _, rx := range regexes {
			indeces := rx.FindStringIndex(focusString)
			if indeces != nil && indeces[0] < fistIndex {
				fistIndex, lastIndex = indeces[0], indeces[1]
			}
		}

		if fistIndex == len(focusString) {
			result.Parts = append(result.Parts, DetectPart(focusString))
			break
		}

		before := focusString[:fistIndex]
		target := focusString[fistIndex:lastIndex]
		after := focusString[lastIndex:]

		if len(before) > 0 {
			result.Parts = append(result.Parts, DetectPart(before))
		}

		result.Parts = append(result.Parts, DetectPart(target))

		if len(after) == 0 {
			break
		}
		focusString = after
	}

	return result
}

type matchToType struct {
	Regex *regexp.Regexp
	Type  int
}

func DetectPart(text string) *translating.StringPart {
	matchToTypes := []matchToType{
		{Regex: regexp.MustCompile(styleSymbolPattern), Type: TypeStyleSymbol},
		{Regex: regexp.MustCompile(fontSizeSymbolPattern), Type: TypeFontSizesymbol},
		{Regex: regexp.MustCompile(colorSymbolPattern), Type: TypeColorSymbol},
		{Regex: regexp.MustCompile(quotesSymbolPattern), Type: TypeQuotesSymbol},
		{Regex: regexp.MustCompile(spamSymbolPattern), Type: TypeSpamSymbol},
	}

	for _, mt := range matchToTypes {
		if match := mt.Regex.MatchString(text); match {
			return &translating.StringPart{
				Type:  mt.Type,
				Value: mt.Regex.FindStringSubmatch(text)[1],
			}
		}
	}

	return &translating.StringPart{
		Type:  TypeString,
		Value: text,
	}
}
