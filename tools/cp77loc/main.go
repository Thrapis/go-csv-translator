// Command cp77loc converts Cyberpunk 2077 localization between WolvenKit's
// JSON export (*.json.json) and Crowdin CSV (id,source,translation,context),
// the format the translator and Crowdin both work on.
//
//	cp77loc export -in raw -out flat          # JSON tree -> CSV tree
//	cp77loc import -template raw -in flat-be -out be
//	cp77loc verify -in raw                    # byte-exact re-encode check
//
// export writes one CSV per resource that has text, mirroring the tree
// (x.json.json -> x.csv). Rows: the female (default) variant under the entry
// id (primaryKey / stringId), the male variant under id@male. Empty strings,
// "[en_us]..." voice-over placeholders and the game's deliberately corrupted
// glitch text are not exported (import keeps them as they are).
// onscreens_final.json.json is skipped: it duplicates onscreens.json.json in
// the same folder. (Other "*_final" files are ordinary subtitle scenes.)
//
// import copies the template tree and writes every non-empty translation cell
// into its entry, touching nothing else. onscreens_final.json.json takes the
// translations of onscreens.csv. Files without a CSV are copied unchanged.
package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strings"

	"github.com/Thrapis/go-csv-translator/internal/extract/crowdincsv"
	"github.com/Thrapis/go-csv-translator/internal/game/cyberpunk2077"
	"github.com/Thrapis/go-csv-translator/internal/wolvenkit"
)

const (
	jsonExt = ".json.json"
	// finalName duplicates its sibling onscreensName; they share one CSV.
	finalName     = "onscreens_final" + jsonExt
	onscreensName = "onscreens" + jsonExt
)

func main() {
	if len(os.Args) < 2 {
		usage()
	}
	var err error
	switch cmd, args := os.Args[1], os.Args[2:]; cmd {
	case "export":
		err = cmdExport(args)
	case "import":
		err = cmdImport(args)
	case "verify":
		err = cmdVerify(args)
	default:
		usage()
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `usage:
  cp77loc export -in <raw json tree> -out <csv tree>
  cp77loc import -template <raw json tree> -in <translated csv tree> -out <json tree>
  cp77loc verify -in <raw json tree>`)
	os.Exit(2)
}

func required(fs *flag.FlagSet, args []string, names ...string) {
	_ = fs.Parse(args)
	for _, n := range names {
		if fs.Lookup(n).Value.String() == "" {
			fmt.Fprintf(os.Stderr, "%s: -%s is required\n", fs.Name(), n)
			fs.Usage()
			os.Exit(2)
		}
	}
}

// isFinal reports whether rel is the on-screen duplicate onscreens_final.
func isFinal(rel string) bool { return filepath.Base(rel) == finalName }

// csvRel maps a resource path to its CSV path; onscreens_final shares
// onscreens.csv.
func csvRel(rel string) string {
	if isFinal(rel) {
		rel = filepath.Join(filepath.Dir(rel), onscreensName)
	}
	return strings.TrimSuffix(rel, jsonExt) + ".csv"
}

// walkFiles calls fn with the slash-free relative path of every file under root.
func walkFiles(root string, fn func(rel string) error) error {
	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return err
		}
		rel, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		return fn(rel)
	})
}

func writeFile(path string, b []byte) error {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, b, 0o644)
}

// --- export ------------------------------------------------------------------

func cmdExport(args []string) error {
	fl := flag.NewFlagSet("export", flag.ExitOnError)
	in := fl.String("in", "", "WolvenKit JSON tree (required)")
	out := fl.String("out", "", "CSV tree to write (required)")
	required(fl, args, "in", "out")

	var files, rows int
	var sk skipped
	err := walkFiles(*in, func(rel string) error {
		if !strings.HasSuffix(rel, jsonExt) || isFinal(rel) {
			return nil
		}
		b, err := os.ReadFile(filepath.Join(*in, rel))
		if err != nil {
			return err
		}
		f, err := wolvenkit.Parse(b)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		rs, dups := exportRows(f, &sk)
		if len(dups) > 0 {
			fmt.Fprintf(os.Stderr, "warn: %s: duplicate ids %v (import applies one translation to all)\n", rel, dups)
		}
		if len(rs) == 0 {
			return nil
		}
		var buf bytes.Buffer
		if err := crowdincsv.Write(&buf, rs); err != nil {
			return err
		}
		files++
		rows += len(rs)
		return writeFile(filepath.Join(*out, csvRel(rel)), buf.Bytes())
	})
	if err != nil {
		return err
	}
	fmt.Printf("exported %d rows into %d files (skipped: %d VO placeholders, %d glitch strings) -> %s\n",
		rows, files, sk.placeholders, sk.glitch, *out)
	return nil
}

// skipped counts strings left out of the export because they are never
// translated.
type skipped struct{ placeholders, glitch int }

func exportRows(f *wolvenkit.File, sk *skipped) (rows []crowdincsv.Row, dups []string) {
	seen := map[string]bool{}
	for _, e := range f.Entries {
		gendered := e.Female != "" && e.Male != ""
		for _, male := range []bool{false, true} {
			text := e.Female
			if male {
				text = e.Male
			}
			if text == "" {
				continue
			}
			if cyberpunk2077.IsVOPlaceholder(text) {
				sk.placeholders++
				continue
			}
			if cyberpunk2077.IsGlitchText(text) {
				sk.glitch++
				continue
			}
			key := e.Key(male)
			if seen[key] {
				dups = append(dups, key)
				continue
			}
			seen[key] = true
			rows = append(rows, crowdincsv.Row{ID: key, Source: text, Context: context(e, gendered, male)})
		}
	}
	return rows, dups
}

func context(e wolvenkit.Entry, gendered, male bool) string {
	var parts []string
	if e.SecondaryKey != "" {
		parts = append(parts, e.SecondaryKey)
	}
	switch {
	case gendered && male:
		parts = append(parts, "male V variant")
	case gendered:
		parts = append(parts, "female V variant")
	}
	return strings.Join(parts, " | ")
}

// --- import ------------------------------------------------------------------

func cmdImport(args []string) error {
	fl := flag.NewFlagSet("import", flag.ExitOnError)
	tmpl := fl.String("template", "", "original WolvenKit JSON tree (required)")
	in := fl.String("in", "", "translated CSV tree (required)")
	out := fl.String("out", "", "JSON tree to write (required)")
	required(fl, args, "template", "in", "out")

	cache := map[string]map[string]string{} // csv rel -> id -> translation
	var files, copied, replaced, kept, unknown int

	err := walkFiles(*tmpl, func(rel string) error {
		src := filepath.Join(*tmpl, rel)
		b, err := os.ReadFile(src)
		if err != nil {
			return err
		}
		dst := filepath.Join(*out, rel)
		if !strings.HasSuffix(rel, jsonExt) {
			copied++
			return writeFile(dst, b)
		}

		cr := csvRel(rel)
		tr, ok := cache[cr]
		if !ok {
			tr, err = loadTranslations(filepath.Join(*in, cr))
			if err != nil {
				return err
			}
			cache[cr] = tr
		}
		if tr == nil {
			copied++
			return writeFile(dst, b)
		}

		f, err := wolvenkit.Parse(b)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		nb, st, err := wolvenkit.Splice(b, f, tr)
		if err != nil {
			return fmt.Errorf("%s: %w", rel, err)
		}
		if n := countUnknown(f, tr); n > 0 {
			unknown += n
			fmt.Fprintf(os.Stderr, "warn: %s: %d translated ids not in the template\n", rel, n)
		}
		files++
		replaced += st.Replaced
		kept += st.Kept
		return writeFile(dst, nb)
	})
	if err != nil {
		return err
	}
	fmt.Printf("imported %d files (%d strings translated, %d kept as source, %d unknown ids); %d copied unchanged -> %s\n",
		files, replaced, kept, unknown, copied, *out)
	return nil
}

// loadTranslations returns id -> non-empty translation, or nil if path does
// not exist.
func loadTranslations(path string) (map[string]string, error) {
	fh, err := os.Open(path)
	if errors.Is(err, fs.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	defer fh.Close()
	rows, err := crowdincsv.Read(fh)
	if err != nil {
		return nil, fmt.Errorf("%s: %w", path, err)
	}
	tr := make(map[string]string, len(rows))
	for _, r := range rows {
		if r.Translation != "" {
			tr[r.ID] = r.Translation
		}
	}
	return tr, nil
}

func countUnknown(f *wolvenkit.File, tr map[string]string) int {
	known := make(map[string]bool, 2*len(f.Entries))
	for _, e := range f.Entries {
		known[e.Key(false)], known[e.Key(true)] = true, true
	}
	n := 0
	for id := range tr {
		if !known[id] {
			n++
		}
	}
	return n
}

// --- verify ------------------------------------------------------------------

func cmdVerify(args []string) error {
	fl := flag.NewFlagSet("verify", flag.ExitOnError)
	in := fl.String("in", "", "WolvenKit JSON tree (required)")
	required(fl, args, "in")

	var files, strs int
	var bad []string
	err := walkFiles(*in, func(rel string) error {
		if !strings.HasSuffix(rel, jsonExt) {
			return nil
		}
		b, err := os.ReadFile(filepath.Join(*in, rel))
		if err != nil {
			return err
		}
		f, err := wolvenkit.Parse(b)
		if err != nil {
			bad = append(bad, fmt.Sprintf("%s: %v", rel, err))
			return nil
		}
		self := map[string]string{}
		for _, e := range f.Entries {
			self[e.Key(false)], self[e.Key(true)] = e.Female, e.Male
		}
		nb, st, err := wolvenkit.Splice(b, f, self)
		switch {
		case err != nil:
			bad = append(bad, fmt.Sprintf("%s: %v", rel, err))
		case !bytes.Equal(nb, b):
			bad = append(bad, rel+": re-encoded bytes differ")
		}
		files++
		strs += st.Replaced
		return nil
	})
	if err != nil {
		return err
	}
	for _, s := range bad {
		fmt.Fprintln(os.Stderr, "FAIL", s)
	}
	fmt.Printf("verified %d files, %d strings re-encoded, %d failures\n", files, strs, len(bad))
	if len(bad) > 0 {
		return fmt.Errorf("verify: %d files do not round-trip", len(bad))
	}
	return nil
}
