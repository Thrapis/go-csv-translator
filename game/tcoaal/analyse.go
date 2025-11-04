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
)

func Analyse(text string) *translating.PartialString {
	var (
		styleSymbolRegex    = regexp.MustCompile(styleSymbolPattern)
		fontSizeSymbolRegex = regexp.MustCompile(fontSizeSymbolPattern)
		colorSymbolRegex    = regexp.MustCompile(colorSymbolPattern)
		quotesSymbolRegex   = regexp.MustCompile(quotesSymbolPattern)
	)

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

		styleSymbolIndeces := styleSymbolRegex.FindStringIndex(focusString)
		if styleSymbolIndeces != nil && styleSymbolIndeces[0] < fistIndex {
			fistIndex, lastIndex = styleSymbolIndeces[0], styleSymbolIndeces[1]
		}

		fontSizeSymbolIndeces := fontSizeSymbolRegex.FindStringIndex(focusString)
		if fontSizeSymbolIndeces != nil && fontSizeSymbolIndeces[0] < fistIndex {
			fistIndex, lastIndex = fontSizeSymbolIndeces[0], fontSizeSymbolIndeces[1]
		}

		colorSymbolIndeces := colorSymbolRegex.FindStringIndex(focusString)
		if colorSymbolIndeces != nil && colorSymbolIndeces[0] < fistIndex {
			fistIndex, lastIndex = colorSymbolIndeces[0], colorSymbolIndeces[1]
		}

		quotesIndeces := quotesSymbolRegex.FindStringIndex(focusString)
		if quotesIndeces != nil && quotesIndeces[0] < fistIndex {
			fistIndex, lastIndex = quotesIndeces[0], quotesIndeces[1]
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

func DetectPart(text string) *translating.StringPart {
	var (
		styleSymbolRegex    = regexp.MustCompile(styleSymbolPattern)
		fontSizeSymbolRegex = regexp.MustCompile(fontSizeSymbolPattern)
		colorSymbolRegex    = regexp.MustCompile(colorSymbolPattern)
		quotesSymbolRegex   = regexp.MustCompile(quotesSymbolPattern)
	)

	if match := styleSymbolRegex.MatchString(text); match {
		return &translating.StringPart{
			Type:  TypeStyleSymbol,
			Value: styleSymbolRegex.FindStringSubmatch(text)[1],
		}
	} else if match := fontSizeSymbolRegex.MatchString(text); match {
		return &translating.StringPart{
			Type:  TypeFontSizesymbol,
			Value: fontSizeSymbolRegex.FindStringSubmatch(text)[1],
		}
	} else if match := colorSymbolRegex.MatchString(text); match {
		return &translating.StringPart{
			Type:  TypeColorSymbol,
			Value: colorSymbolRegex.FindStringSubmatch(text)[1],
		}
	} else if match := quotesSymbolRegex.MatchString(text); match {
		return &translating.StringPart{
			Type:  TypeQuotesSymbol,
			Value: quotesSymbolRegex.FindStringSubmatch(text)[1],
		}
	}

	return &translating.StringPart{
		Type:  TypeString,
		Value: text,
	}
}
