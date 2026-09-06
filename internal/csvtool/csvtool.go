// Package csvtool splits one multi-section TCOAAL CSV into per-section files and
// merges them back. The section separator is three blank CRLF lines; a section
// header row is "Section,<name>".
package csvtool

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

const sectionSeparator = "\r\n\r\n\r\n"

// Split writes each section of inFile to "<inFile without extension>/<section>.csv".
func Split(inFile string) error {
	dir, ext, _ := strings.Cut(inFile, ".")
	if err := os.MkdirAll(dir, os.ModePerm); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}

	data, err := os.ReadFile(inFile)
	if err != nil {
		return fmt.Errorf("read %s: %w", inFile, err)
	}

	for _, chunk := range strings.SplitAfter(string(data), sectionSeparator) {
		header, body, _ := strings.Cut(chunk, "\r\n")
		_, section, ok := strings.Cut(header, ",")
		if !ok || section == "" {
			continue
		}
		out := filepath.Join(dir, section+"."+ext)
		if err := os.WriteFile(out, []byte(body), os.ModePerm); err != nil {
			return fmt.Errorf("write %s: %w", out, err)
		}
	}
	return nil
}

// Merge concatenates every file in dir into outFile, prefixing each with a
// "Section,<filename without .csv>" header and separating sections with blank lines.
func Merge(dir, outFile string) error {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return fmt.Errorf("read dir %s: %w", dir, err)
	}

	var b strings.Builder
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			return fmt.Errorf("read %s: %w", entry.Name(), err)
		}
		if b.Len() != 0 {
			b.WriteString(sectionSeparator)
		}
		section := strings.TrimSuffix(entry.Name(), ".csv")
		fmt.Fprintf(&b, "Section,%s\r\n", section)
		b.Write(data)
	}

	if err := os.WriteFile(outFile, []byte(b.String()), 0o644); err != nil {
		return fmt.Errorf("write %s: %w", outFile, err)
	}
	return nil
}
