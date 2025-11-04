package translating

import "tw-translator/extracting"

type TranslationSettings struct {
	Analyse                    PartialStringAnalyse
	PartialStringGetTypeString PartialStringGetTypeString
	PartialStringString        PartialStringString
	Delimeter                  string
	SkipFirstLine              bool
	SourceFolder               string
	DestinationFolder          string
	SourceLang                 string
	TargetLang                 string
	SourceFolderNameReplace    string
	TargetFolderNameReplace    string
	Exract                     extracting.Extractor
	Compose                    extracting.Composer
	MultiRowReplicas           bool
	Parasitizing               bool
	ParasitizingFilePath       string
	AreSameReplica             AreSameReplica
	ParasiteReplica            ParasiteReplica
}
