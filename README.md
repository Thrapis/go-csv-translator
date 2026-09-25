# go-csv-translator

Machine-translates a game's localization files. Each file is run through a
pipeline:

```
extract  ->  analyze markup  ->  translate text parts  ->  render  ->  compose
```

so engine control symbols (colour codes, variables, gender/ternary forms, ...)
are left untouched and only human-readable text is sent to a translator.

Supported games: **tcoaal** (The Coffin of Andy and Leyley), **cyberpunk2077**
(Cyberpunk 2077), **taleworld** (Mount & Blade II: Bannerlord), **titanquest**
(Titan Quest).

## Usage

Everything about a run lives in a YAML config; nothing is compiled in.

```bash
go run ./cmd/gametranslator -config config/tcoaal.ru-be.yaml
```

Copy one of the `config/*.example.yaml` files, edit the paths, and drop the
`.example` from the name (real configs are gitignored). Every field is
documented inline in the examples; the essentials:

| Field | Meaning |
|---|---|
| `game` | which game plugin to use |
| `source.folder` / `destination.folder` | input tree / output tree (recursed) |
| `source.format` | file layout id; empty = the game's default |
| `source.files` | base-name glob (e.g. `dialogue.*`); empty = every file |
| `destination.folderNameMap` | rename sub-folders on output, e.g. `ru: be` |
| `language.source` / `language.target` | translation direction |
| `translators` | ordered fallback chain, e.g. `[lingvanex, google]` |
| `lingvanex.manage` | when true, start the Lingvanex server on launch and stop it on exit |
| `multiRowReplicas` | translate consecutive same-tag rows as one sentence |
| `parasitizing.carryOver` | target-language files (e.g. a previous run's output); a matching entry is copied verbatim, never re-translated |
| `parasitizing.files` | source-language translations (TXT or CSV); still translated, but a better source than the game's own. Checked after carryOver; first match wins, the rest is machine-translated |

Matching: a **dialogue** replica by its `#id`; a **LABELS / MENUS / FONT /
CREDITS** entry by section + key text (so `Save` in `LABELS` and in `MENUS`
stay distinct); **LANGUAGE** by the section name. A source parasite covers
everything but the localization-identity rows (LANGUAGE / FONT / CREDITS) —
those are filled only from `carryOver`, and without a match keep the source
value (they are never machine-translated). `[VERSION]` always comes from the
new source.

### TCOAAL formats

"The Coffin of Andy and Leyley" ships its localization as one combined file in
two interchangeable layouts. Point `source.folder` at the folder that holds it
and select the layout with `source.format`:

| `source.format` | file |
|---|---|
| `tcoaal-csv` (default) | `dialogue.csv` — 4-column CSV |
| `tcoaal-txt` | `dialogue.txt` — `key : value` lines |

Both read and write the combined file directly (no split/merge step). `[VERSION]`
passes through from the source; `[LANGUAGE]` / `[FONT]` / `[CREDITS]` come from a
`carryOver` localization or stay as the source value; the
rest — menu labels, speaker/item names, descriptions and all dialogue — is
translated in place, and every non-translated byte is preserved.

Each string is translated **whole**: engine markup (`\c[2]`, `\fi`, quotes,
`...`) is swapped for `§i§` sentinels, the whole line goes to the translator in
one request, and the markup is spliced back — so the model sees full sentences,
not fragments. A multi-row replica is translated as one unit and the result is
placed on its first row with the rest blanked (RPG-Maker boxes re-wrap, but a
very long replica can overflow a fixed box).

Ctrl+C stops the run promptly; partial output is left on disk and a managed
Lingvanex server is shut down.

### Updating to a new game version

Point `parasitizing.carryOver` at the previous version's output and run against
the new source. Replica ids are stable across minor versions, and a changed
source string gets a new id, so a matched id is copied verbatim and only the
new / changed lines are translated — seconds instead of the full run. The
`replicas` log line reports the split (`carried_over` / `from_parasite` /
`machine_translated`).

### Cyberpunk 2077

The game's localization is exported with WolvenKit as a tree of
`*.json.json` files (`localization/ru-ru/onscreens/…`, `…/subtitles/…`). The
translator never touches that JSON. Instead it goes through a Crowdin-ready CSV
tree:

```bash
# 1. WolvenKit JSON -> Crowdin CSV (one x.csv per x.json.json)
go run ./tools/cp77loc export -in ".../raw" -out ".../flat-ru"
# 2. machine-translate: fills the translation column (flat-ru -> flat-be)
go run ./cmd/gametranslator -config config/cyberpunk2077.ru-be.yaml
# 3. (optional) human review in a CAT tool (OmegaT, Weblate, ...) via XLIFF 1.2
go run ./tools/cp77loc xliff -in ".../flat-be" -out ".../xliff-be"
# 4. CSV or XLIFF -> WolvenKit JSON, using the original export as the template
go run ./tools/cp77loc import -template ".../raw" -in ".../flat-be" -out ".../be"
go run ./tools/cp77loc import -template ".../raw" -in ".../xliff-be" -out ".../be"   # after review
```

In the XLIFF, engine markup (tags, `{vars}`, `\n`, the non-text parts of
kiroshi/mothertongue) becomes locked `<ph>` placeholders, which a reviewer can
move but not edit. Typography such as `—` and `…` stays ordinary text. Machine
translations arrive in state `needs-review-translation`, and the CSV context
column becomes a `<note>`. On import, each placeholder is replaced with the
exact source markup. A unit whose placeholders were lost, duplicated or
invented is rejected, and it keeps the source text.

The CSV is `id,source,translation,context`. The id is the entry's `primaryKey`
(on-screen texts) or `stringId` (subtitles). The male-V variant of a gendered
line gets `id@male`, and the context column carries the `secondaryKey`. Empty
strings, `[en_us]…` voice-over placeholders and the game's deliberately
corrupted glitch text are not exported. Glitch text is the "Corrupted file"
shards and e-mails, stored as mojibake with U+0080–U+009F control characters.
On import, those strings keep their original bytes.
`onscreens_final.json.json` duplicates `onscreens.json.json`, so it is not
exported, and on import it gets the same translations. Crowdin reads the files
as-is with this `crowdin.yml` entry:

```yaml
files:
  - source: /flat-ru/**/*.csv
    translation: /flat-be/**/%original_file_name%
    scheme: "identifier,source_phrase,translation,context"
    first_line_contains_header: true
```

`import` edits only the variant string literals in the template bytes, so every
other byte stays exactly as exported. It encodes new text the way WolvenKit
does (ASCII with uppercase `\uXXXX`). An empty translation cell keeps the source
text. `cp77loc verify -in .../raw` proves the round-trip: every string is
re-encoded and the files must come out byte-identical.

Markup handling:
- `<Rich …>`, `<Input …>`, `<Image …>`, `</>`, `{variables}` and symbol runs are
  masked, and each string is translated whole.
- In `<kiroshi …/>` and `<mothertongue …/>`, only the `t` / `b` / `a` attribute
  text is translated. The foreign original (`o`, `m`) is kept.
- Long texts are split at their `\n` line breaks, so each request stays within
  the model's input length.
- Text with no Cyrillic letters (codes, Latin names) is never sent to the
  translator.
- Characters the ru→be model doesn't reproduce are kept verbatim. This was
  measured on the full onscreens run: `—` became `-` in 100% of rows, `…`
  became `...` in 97%, and `–` became `우` in 93%. Rare Cyrillic, kaomoji,
  box drawing and `§` are also kept verbatim. Only Russian/Belarusian letters,
  ASCII and `« » № € „ “ ” ‘ ’ ® ° © · ×` reach the model.

## Lingvanex server

`third_party/lingvanex-server/` holds the local Python translation server
(CTranslate2 + SentencePiece). With `lingvanex.manage: true` the app spawns and
supervises it. To run it yourself instead, set `manage: false` and start
`third_party/lingvanex-server/start.bat` (or `py server.py`). Model files are
gitignored.

## Extra tools

```bash
# inspection helper: split a combined TCOAAL dialogue.csv into per-section files
go run ./tools/csvsplit -in "path/to/dialogue.csv"

# Cyberpunk 2077: WolvenKit JSON <-> Crowdin CSV (see "Cyberpunk 2077" above)
go run ./tools/cp77loc export|import|verify ...
```

## Layout

```
cmd/gametranslator     entrypoint: flags, config, signals, wiring
internal/
  config               YAML schema + loader/validation
  app                  config -> wired pipeline
  pipeline             folder walk + extract/translate/compose orchestration
  markup               PartialString model + Analyzer interface
  extract              Extractor/Composer interfaces + format registry
    delimited          generic "key<delim>value"        (id: "delimited")
    crowdincsv         id,source,translation,context     (id: "crowdin-csv")
    tcoaalcsv          TCOAAL combined dialogue.csv      (id: "tcoaal-csv")
    tcoaaltxt          TCOAAL combined dialogue.txt      (id: "tcoaal-txt")
  translate            Translator interface + backend registry + fallback Chain
    lingvanex          HTTP client for the local server
    google             free Google Translate fallback
  backend/lingvanex    Lingvanex server process supervisor
  game                 Game interface + optional capabilities + registry
    tcoaal / cyberpunk2077 / taleworld / titanquest
  wolvenkit            WolvenKit JSON reader + byte-exact string splicer
  xliff                XLIFF 1.2 reader/writer with locked <ph> markup
  textutil             small string helpers
  plugins              blank-imports every game/format/backend to register them
tools/csvsplit         inspection helper (combined dialogue.csv -> section files)
tools/cp77loc          Cyberpunk 2077 WolvenKit JSON <-> Crowdin CSV / XLIFF converter
```

A section-aware format keeps the document skeleton in `extract.Settings.Extra`
and returns only translatable cells as `DataLine`s; its `Compose` splices the
translations back and re-serialises.

## Adding a game

1. New package under `internal/game/<name>/`.
2. Implement `game.Game` (`Name`, `Analyzer`, `DefaultFormat`, `DefaultDelimiter`).
   Optionally implement `game.ReplicaGrouper` and/or `game.Parasitizer`.
3. Implement `markup.Analyzer` for the engine's markup (`Analyze`, `Translatable`,
   `Render`; keep `Render(Analyze(s)) == s`).
4. If the file layout is new, add an `extract.Format` and register it; otherwise
   return an existing id (`"delimited"`, `"crowdin-csv"`, `"tcoaal-csv"`,
   `"tcoaal-txt"`) from `DefaultFormat`.
5. `game.Register(New())` in the package `init`, and add a blank import to
   `internal/plugins`.
