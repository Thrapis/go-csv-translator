package extracting

import (
	"fmt"
	"io"
	"strings"

	"tw-translator/extracting"

	"golang.org/x/net/html/charset"
)

const (
	crfl = "\r\n"
	lf   = "\n"
)

func Extract(in io.Reader, out *[]*extracting.DataLine, delimeter string) (*extracting.Settings, error) {
	bytes, err := io.ReadAll(in)
	if err != nil {
		return nil, err
	}

	// enc, _, _ := charset.DetermineEncoding(bytes, "")
	enc, _ := charset.Lookup("utf8")

	utf8Bytes, err := enc.NewDecoder().Bytes(bytes)
	if err != nil {
		return nil, err
	}

	text := string(utf8Bytes)

	lineDelimeter := lf
	if strings.Contains(text, crfl) {
		lineDelimeter = crfl
	}

	settings := &extracting.Settings{
		Encoding:      enc,
		LineDelimeter: lineDelimeter,
	}

	lines := strings.Split(text, lineDelimeter)

	*out = make([]*extracting.DataLine, 0)

	for _, line := range lines {
		values := strings.SplitN(line, delimeter, 4)
		if len(values) > 3 {
			*out = append(*out, &extracting.DataLine{
				Key:   fmt.Sprintf("%s,%s,%s", values[0], values[1], values[2]),
				Value: values[2],
				Tag:   fmt.Sprintf("%s,%s", values[0], values[1]),
			})
		}
	}
	return settings, nil
}

func Compose(settings *extracting.Settings, out io.Writer, in *[]*extracting.DataLine, delimeter string) error {
	text := strings.Builder{}

	for _, dataLine := range *in {
		line := fmt.Sprintf("%s%s%s%s", dataLine.Key, delimeter, dataLine.Value, settings.LineDelimeter)
		text.WriteString(line)
	}

	bytes := []byte(text.String())

	fileBytes, err := settings.Encoding.NewEncoder().Bytes(bytes)
	if err != nil {
		return err
	}

	_, err = out.Write(fileBytes)
	return err
}
