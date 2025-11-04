package tcoaal

import (
	"fmt"
	"os"
	"strings"
)

var loaded bool = false
var parasiteFile string

func ParasiteReplica(filePath, tag string) string {
	if !loaded {
		bytes, err := os.ReadFile(filePath)
		if err != nil {
			panic(err)
		}

		parasiteFile = string(bytes)
	}

	replicaId := fmt.Sprintf("#%s", strings.Split(tag, ",")[0])
	replicaStartIndex := strings.Index(parasiteFile, replicaId)
	replicaEndIndex := replicaStartIndex + strings.Index(parasiteFile[replicaStartIndex:], "\r\n\r\n")
	if replicaEndIndex == replicaStartIndex-1 {
		replicaEndIndex = len(parasiteFile)
	}

	replicaCut := parasiteFile[replicaStartIndex:replicaEndIndex]

	// fmt.Printf("]%s[\n", replicaCut)

	replicaSplits := strings.Split(replicaCut, "\r\n")

	replicaValueBuilder := strings.Builder{}
	for i := 1; i < len(replicaSplits); i++ {
		if strings.HasPrefix(replicaSplits[i], ": ") {
			stringToAdd := strings.Replace(strings.Replace(replicaSplits[i], ": ", "", 1), "\r\n", "", 1)
			if replicaValueBuilder.Len() != 0 {
				replicaValueBuilder.WriteString(" ")
			}
			replicaValueBuilder.WriteString(stringToAdd)
		}
	}

	return replicaValueBuilder.String()
}
