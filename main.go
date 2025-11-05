package main

import (
	"tw-translator/game/tcoaal"
	"tw-translator/game/tcoaal/extracting"
	"tw-translator/translating"
)

func main() {
	// Translating
	settings := tcoaal.NewTheCoffinOfAndyAndLeyleySettings()
	settings.SourceFolder = "sourcePath"
	settings.DestinationFolder = "targetPath"
	settings.SourceLang = "ru"
	settings.TargetLang = "be"
	settings.SourceFolderNameReplace = "ru"
	settings.TargetFolderNameReplace = "be"
	settings.Exract = extracting.Extract
	settings.Compose = extracting.Compose
	settings.MultiRowReplicas = true
	settings.Parasitizing = true
	settings.ParasitizingFilePath = "parasitizingFilePath"

	translating.StartTranslation(settings)

	// File splitting TCOAAL
	// filepath := "filePath"
	// utils.SplitInFiles(filepath)

	// File merging TCOAAL
	// dirpath := "dirpath"
	// filepath := "filepath"
	// utils.MergeFiles(dirpath, filepath)

	// Lingvanex Translator API Call
	// fmt.Println(lingvanex.Translate("Что у тебя там? Всё в порядке?", "ru", "be"))
}
