package extracting

import (
	"encoding/csv"
	"fmt"
	"io"
	"strings"

	"tw-translator/extracting"

	"github.com/gocarina/gocsv"
	"golang.org/x/net/html/charset"
)

const (
	crfl = "\r\n"
	lf   = "\n"
)

type tcoaal struct {
	ID          string `csv:"ID"`
	Source      string `csv:"Source"`
	English     string `csv:"English"`
	Translation string `csv:"Translation"`
}

func Extract(in io.Reader, out *[]*extracting.DataLine, delimeter string) (*extracting.Settings, error) {
	rows := make([]tcoaal, 0)

	gocsv.SetCSVReader(func(in io.Reader) gocsv.CSVReader {
		r := csv.NewReader(in)
		r.Comma = ','
		return r
	})

	enc, _ := charset.Lookup("utf8")

	settings := &extracting.Settings{
		Encoding:      enc,
		LineDelimeter: crfl,
	}

	gocsv.Unmarshal(in, &rows)

	*out = make([]*extracting.DataLine, 0)

	for _, row := range rows {
		*out = append(*out, &extracting.DataLine{
			Key:   fmt.Sprintf("%s,%s,%s", row.ID, row.Source, row.English),
			Value: row.English,
			Tag:   fmt.Sprintf("%s,%s", row.ID, row.Source),
		})
	}

	return settings, nil
}

func Compose(settings *extracting.Settings, out io.Writer, in *[]*extracting.DataLine, delimeter string) error {
	gocsv.SetCSVWriter(func(out io.Writer) *gocsv.SafeCSVWriter {
		writer := csv.NewWriter(out)
		writer.Comma = ','
		return gocsv.NewSafeCSVWriter(writer)
	})

	rows := make([]tcoaal, 0, len(*in))

	for _, dataLine := range *in {
		parts := strings.SplitN(dataLine.Key, ",", 3)
		rows = append(rows, tcoaal{
			ID:          parts[0],
			Source:      parts[1],
			English:     parts[2],
			Translation: dataLine.Value,
		})
	}

	str, err := gocsv.MarshalString(&rows)
	if err != nil {
		return err
	}

	_, err = out.Write([]byte(str))

	return err
}
