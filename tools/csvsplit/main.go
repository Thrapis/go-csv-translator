// Command csvsplit breaks one multi-section TCOAAL CSV into per-section files
// under a folder named after the input file.
//
//	csvsplit -in path/to/export.csv
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Thrapis/go-csv-translator/internal/csvtool"
)

func main() {
	in := flag.String("in", "", "multi-section CSV to split (required)")
	flag.Parse()

	if *in == "" {
		fmt.Fprintln(os.Stderr, "usage: csvsplit -in <file.csv>")
		os.Exit(2)
	}
	if err := csvtool.Split(*in); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
