// Command csvsplit is an inspection helper: it breaks a combined TCOAAL
// dialogue.csv into one file per "Section,<Map>.json" block (each a clean
// ID,Source,English,Translation CSV) under a folder named after the input.
//
// The main translator reads the combined file directly and does not need this.
//
//	csvsplit -in "path/to/dialogue.csv" [-out path/to/dir]
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/Thrapis/go-csv-translator/internal/extract/tcoaalcsv"
)

func main() {
	in := flag.String("in", "", "combined dialogue.csv to split (required)")
	out := flag.String("out", "", "output directory (default: <in> without extension)")
	flag.Parse()

	if *in == "" {
		fmt.Fprintln(os.Stderr, "usage: csvsplit -in <dialogue.csv> [-out <dir>]")
		os.Exit(2)
	}
	if err := run(*in, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(inPath, outDir string) error {
	f, err := os.Open(inPath)
	if err != nil {
		return err
	}
	defer f.Close()

	doc, err := tcoaalcsv.ParseDoc(f)
	if err != nil {
		return err
	}

	if outDir == "" {
		outDir = strings.TrimSuffix(inPath, filepath.Ext(inPath))
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		return err
	}

	for _, name := range doc.Sections() {
		dst := filepath.Join(outDir, name+".csv")
		wf, err := os.Create(dst)
		if err != nil {
			return err
		}
		err = tcoaalcsv.WriteRecords(wf, doc.SectionRecords(name))
		wf.Close()
		if err != nil {
			return fmt.Errorf("write %s: %w", dst, err)
		}
	}
	fmt.Printf("wrote %d section files to %s\n", len(doc.Sections()), outDir)
	return nil
}
