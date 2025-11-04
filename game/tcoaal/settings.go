package tcoaal

import "tw-translator/translating"

func NewTheCoffinOfAndyAndLeyleySettings() *translating.TranslationSettings {
	return &translating.TranslationSettings{
		Analyse:                    Analyse,
		PartialStringGetTypeString: PartialStringGetTypeString,
		PartialStringString:        PartialStringString,
		Delimeter:                  ",",
		SkipFirstLine:              true,
	}
}
