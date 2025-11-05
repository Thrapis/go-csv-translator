package utils

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func MergeFiles(dirPath, newFilePath string) {
	entries, err := os.ReadDir(dirPath)
	if err != nil {
		panic(err)
	}

	resultFileBuilder := strings.Builder{}

	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}

		filePath := filepath.Join(dirPath, entry.Name())
		bytes, err := os.ReadFile(filePath)
		if err != nil {
			panic(err)
		}

		if resultFileBuilder.Len() != 0 {
			resultFileBuilder.WriteString(delimeter)
		}

		titleToWrite := strings.ReplaceAll(entry.Name(), ".csv", "")
		resultFileBuilder.WriteString(fmt.Sprintf("Section,%s\r\n", titleToWrite))
		resultFileBuilder.WriteString(string(bytes))
	}

	os.WriteFile(newFilePath, []byte(resultFileBuilder.String()), 0777)
}
