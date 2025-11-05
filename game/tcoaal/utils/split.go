package utils

import (
	"os"
	"path"
	"strings"
)

func SplitInFiles(filepath string) {
	newdir, ext, _ := strings.Cut(filepath, ".")
	os.Mkdir(newdir, os.ModePerm)

	data, err := os.ReadFile(filepath)
	if err != nil {
		panic(err)
	}

	text := string(data)
	texts := strings.SplitAfter(text, delimeter)

	for _, v := range texts {
		before, after, _ := strings.Cut(v, "\r\n")
		_, sectionName, _ := strings.Cut(before, ",")

		newFileName := path.Join(newdir, sectionName+"."+ext)
		writeFile, err := os.OpenFile(newFileName, os.O_RDWR|os.O_TRUNC|os.O_CREATE, os.ModePerm)
		if err != nil {
			panic(err)
		}
		defer writeFile.Close()

		writeFile.WriteString(after)
	}
}
