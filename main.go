package main

import (
	"tw-translator/game/thecoffinofandyandleyley"
	"tw-translator/game/thecoffinofandyandleyley/extracting"
	"tw-translator/translating"
)

func main() {
	// Translating
	settings := thecoffinofandyandleyley.NewTheCoffinOfAndyAndLeyleySettings()
	settings.SourceFolder = "sourcePath"
	settings.DestinationFolder = "targetPath"
	settings.SourceLang = "ru"
	settings.TargetLang = "be"
	settings.SourceFolderNameReplace = "en"
	settings.TargetFolderNameReplace = "be"
	settings.Exract = extracting.Extract
	settings.Compose = extracting.Compose

	translating.StartTranslation(settings)

	// File splitting TCOAAL
	// filepath := "sourceCsv"
	// splitting.SplitInFiles(filepath, "\r\n\r\n\r\n")

	// Lingvanex Translator API Call
	// fmt.Println(lingvanex.Translate("Что у тебя там? Всё в порядке?", "ru", "be"))
}
