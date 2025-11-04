package tcoaal

import (
	"strings"

	"tw-translator/translating"
)

func PartialStringGetTypeString(ps *translating.PartialString) []*translating.StringPart {
	result := make([]*translating.StringPart, 0, len(ps.Parts))

	for _, part := range ps.Parts {
		result = append(result, StringPartGetTypeString(part)...)
	}

	return result
}

func PartialStringString(ps *translating.PartialString) string {
	builder := strings.Builder{}
	for _, part := range ps.Parts {
		builder.WriteString(StringPartString(part))
	}
	return builder.String()
}

const (
	TypeString = iota
	TypeStyleSymbol
	TypeFontSizesymbol
	TypeColorSymbol
	TypeQuotesSymbol
)

func StringPartGetTypeString(sp *translating.StringPart) []*translating.StringPart {
	result := make([]*translating.StringPart, 0)

	if sp.Type == TypeString {
		result = append(result, sp)
	}

	return result
}

func StringPartString(sp *translating.StringPart) string {
	switch sp.Type {
	default:
		fallthrough
	case TypeString:
		return sp.Value
	case TypeStyleSymbol:
		return sp.Value
	case TypeFontSizesymbol:
		return sp.Value
	case TypeColorSymbol:
		return sp.Value
	case TypeQuotesSymbol:
		return sp.Value
	}
}
