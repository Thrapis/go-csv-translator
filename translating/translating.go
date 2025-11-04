package translating

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"tw-translator/extracting"
	"tw-translator/lingvanex"
	"tw-translator/utils"

	translategooglefree "github.com/bas24/googletranslatefree"
)

func StartTranslation(settings *TranslationSettings) {
	folder := getFolder(settings.SourceFolder)

	// folder.PrintDeep()

	translateFolder(folder, settings.DestinationFolder, settings)
}

func getFolder(directory string) *Folder {
	path, name := filepath.Split(directory)
	current := &Folder{
		Name:    name,
		Path:    path,
		Folders: make([]Folder, 0),
		Files:   make([]File, 0),
	}

	entries, _ := os.ReadDir(directory)
	for _, entry := range entries {
		if entry.IsDir() {
			subpath := filepath.Join(directory, entry.Name())
			subFolder := getFolder(subpath)
			current.Folders = append(current.Folders, *subFolder)
		} else {
			current.Files = append(current.Files, File{
				FullName: entry.Name(),
				Path:     directory,
			})
		}
	}
	return current
}

func translateFolder(sourceFolder *Folder, destinationFolder string, settings *TranslationSettings) {
	if _, err := os.Stat(destinationFolder); os.IsNotExist(err) {
		os.MkdirAll(destinationFolder, os.ModePerm)
	}

	for _, file := range sourceFolder.Files {
		// file to read
		readFile, err := os.OpenFile(file.FullPath(), os.O_RDONLY, os.ModePerm)
		if err != nil {
			panic(err)
		}
		defer readFile.Close()

		lines := []*extracting.DataLine{}

		// extract lines
		extractionSettings, err := settings.Exract(readFile, &lines, settings.Delimeter)
		if err != nil {
			panic(err)
		}

		if settings.SkipFirstLine {
			lines = lines[1:]
		}

		if settings.MultiRowReplicas {
			translateMRRLines(lines, settings)
		} else {
			translateLines(lines, settings)
		}

		filePathToWrite := filepath.Join(destinationFolder, file.FullName)

		// file to write
		writeFile, err := os.OpenFile(filePathToWrite, os.O_RDWR|os.O_TRUNC|os.O_CREATE, os.ModePerm)
		if err != nil {
			panic(err)
		}
		defer writeFile.Close()

		// write lines
		settings.Compose(extractionSettings, writeFile, &lines, settings.Delimeter)
	}

	for _, folder := range sourceFolder.Folders {
		destinationFolderJoin := folder.Name
		if destinationFolderJoin == settings.SourceFolderNameReplace {
			destinationFolderJoin = settings.TargetFolderNameReplace
		}
		translateFolder(&folder, filepath.Join(destinationFolder, destinationFolderJoin), settings)
	}
}

func translateLines(in_out []*extracting.DataLine, settings *TranslationSettings) {
	for i, line := range in_out {
		partialString := settings.Analyse(line.Value)

		for _, part := range settings.PartialStringGetTypeString(partialString) {
			trimmed := strings.TrimSpace(part.Value)

			if len(trimmed) == 0 {
				continue
			}

			leadingSpaces := utils.CountLeadingSpaces(part.Value)
			finalSpaces := utils.CountFinalSpaces(part.Value)

			isUpper := utils.IsUpper([]rune(trimmed)[0])

			translated, err := lingvanex.Translate(trimmed, settings.SourceLang, settings.TargetLang)
			if err != nil {
				fmt.Println("---- Lingvanex failed - Try Google Api ----")
				translated, err = translategooglefree.Translate(trimmed, settings.SourceLang, settings.TargetLang)
				if err != nil {
					panic(err)
				}
			}

			runed := []rune(translated)
			if isUpper {
				translated = strings.ToUpper(string(runed[0])) + string(runed[1:])
			} else {
				translated = strings.ToLower(string(runed[0])) + string(runed[1:])
			}

			part.Value = fmt.Sprintf("%s%s%s", strings.Repeat(" ", leadingSpaces), translated, strings.Repeat(" ", finalSpaces))
		}

		translatedValue := settings.PartialStringString(partialString)

		fmt.Printf("%d%% (%d of %d) - Translating: \"%.32s\" - \"%.32s\"\n", i*100/len(in_out), i, len(in_out), line.Value, translatedValue)

		in_out[i].Value = translatedValue
	}
}

func translateMRRLines(in_out []*extracting.DataLine, settings *TranslationSettings) {
	start := 0
	end := 0

	for i := range in_out {

		if start == i && i != len(in_out)-1 {
			continue
		}

		if start != i && settings.AreSameReplica(in_out[start], in_out[i]) {
			continue
		}

		// Collect all rows replica
		end = i - 1
		totalValue := ""
		if settings.Parasitizing {
			totalValue = settings.ParasiteReplica(settings.ParasitizingFilePath, in_out[start].Tag)
		} else {
			totalValueBuilder := strings.Builder{}
			for j := start; j <= end; j++ {
				totalValueBuilder.WriteString(in_out[j].Value)
			}
			totalValue = totalValueBuilder.String()
		}

		// If empty
		if len(strings.TrimSpace(totalValue)) == 0 {
			start = end + 1
			continue
		}

		// Calculate filled rows
		filled := 0
		for j := start; j <= end; j++ {
			if len(strings.TrimSpace(in_out[j].Value)) != 0 {
				filled++
			}
		}

		// Splitting into Partials
		partialString := settings.Analyse(totalValue)

		// Translate Partials
		for _, part := range settings.PartialStringGetTypeString(partialString) {
			trimmed := strings.TrimSpace(part.Value)

			if len(trimmed) == 0 {
				continue
			}

			leadingSpaces := utils.CountLeadingSpaces(part.Value)
			finalSpaces := utils.CountFinalSpaces(part.Value)

			isUpper := utils.IsUpper([]rune(trimmed)[0])

			translated, err := lingvanex.Translate(trimmed, settings.SourceLang, settings.TargetLang)
			if err != nil {
				fmt.Println("---- Lingvanex failed - Try Google Api ----")
				translated, err = translategooglefree.Translate(trimmed, settings.SourceLang, settings.TargetLang)
				if err != nil {
					panic(err)
				}
			}
			// translated, err := translategooglefree.Translate(trimmed, settings.SourceLang, settings.TargetLang)
			// if err != nil {
			// 	panic(err)
			// }

			runed := []rune(translated)
			if isUpper {
				translated = strings.ToUpper(string(runed[0])) + string(runed[1:])
			} else {
				translated = strings.ToLower(string(runed[0])) + string(runed[1:])
			}

			part.Value = fmt.Sprintf("%s%s%s", strings.Repeat(" ", leadingSpaces), translated, strings.Repeat(" ", finalSpaces))
		}

		// Join Partials into string
		translatedValue := settings.PartialStringString(partialString)

		// Log progress
		fmt.Printf("%d%% (%d of %d) - Translating: \"%.32s\" - \"%.32s\"\n", i*100/len(in_out), i, len(in_out), in_out[start].Value, translatedValue)

		// Write translated
		translatedRows := splitSentence(translatedValue, filled)
		compensation := 0
		for j := start; j <= end; j++ {
			if len(strings.TrimSpace(in_out[j].Value)) == 0 {
				compensation++
				continue
			}
			in_out[j].Value = translatedRows[j-start-compensation]
		}

		start = end + 1
	}
}

func splitSentence(sentence string, n int) []string {
	if n <= 0 {
		return []string{}
	}

	words := strings.Fields(sentence)
	totalWords := len(words)

	// If fewer words than parts, pad with empty strings
	if totalWords < n {
		result := make([]string, n)
		for i := 0; i < totalWords; i++ {
			result[i] = words[i]
		}
		// remaining entries are already ""
		return result
	}

	// Otherwise, distribute words evenly
	baseSize := totalWords / n
	remainder := totalWords % n

	result := make([]string, 0, n)
	start := 0

	for i := 0; i < n; i++ {
		end := start + baseSize
		if i < remainder {
			end++
		}
		part := strings.Join(words[start:end], " ")
		result = append(result, part)
		start = end
	}

	return result
}

// func splitSentence(sentence string, n int) []string {
// 	if n <= 0 {
// 		return []string{}
// 	}

// 	words := strings.Fields(sentence)
// 	totalWords := len(words)
// 	if n > totalWords {
// 		n = totalWords
// 	}

// 	baseSize := totalWords / n
// 	remainder := totalWords % n

// 	result := make([]string, 0, n)
// 	start := 0

// 	for i := 0; i < n; i++ {
// 		end := start + baseSize
// 		if i < remainder {
// 			end++
// 		}
// 		part := strings.Join(words[start:end], " ")
// 		result = append(result, part)
// 		start = end
// 	}

// 	return result
// }
