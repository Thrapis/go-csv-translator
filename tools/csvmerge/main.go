// Command csvmerge concatenates every file in a folder into one multi-section
// TCOAAL CSV.
//
//	csvmerge -in path/to/folder -out path/to/merged.csv
package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/Thrapis/go-csv-translator/internal/csvtool"
)

func main() {
	in := flag.String("in", "", "folder of per-section CSV files (required)")
	out := flag.String("out", "", "merged CSV to write (required)")
	flag.Parse()

	if *in == "" || *out == "" {
		fmt.Fprintln(os.Stderr, "usage: csvmerge -in <folder> -out <file.csv>")
		os.Exit(2)
	}
	if err := csvtool.Merge(*in, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
