# go-csv-translator

Machine-translates a game's localization files. Each file is run through a
pipeline:

```
extract  ->  analyze markup  ->  translate text parts  ->  render  ->  compose
```

so engine control symbols (colour codes, variables, gender/ternary forms, ...)
are left untouched and only human-readable text is sent to a translator.

Supported games: **tcoaal** (The Coffin of Andy and Leyley), **taleworld**
(Mount & Blade II: Bannerlord), **titanquest** (Titan Quest).

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
    tcoaalcsv          TCOAAL combined dialogue.csv      (id: "tcoaal-csv")
    tcoaaltxt          TCOAAL combined dialogue.txt      (id: "tcoaal-txt")
  translate            Translator interface + backend registry + fallback Chain
    lingvanex          HTTP client for the local server
    google             free Google Translate fallback
  backend/lingvanex    Lingvanex server process supervisor
  game                 Game interface + optional capabilities + registry
    tcoaal / taleworld / titanquest
  textutil             small string helpers
  plugins              blank-imports every game/format/backend to register them
tools/csvsplit         inspection helper (combined dialogue.csv -> section files)
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
   return an existing id (`"delimited"`, `"tcoaal-csv"`, `"tcoaal-txt"`) from
   `DefaultFormat`.
5. `game.Register(New())` in the package `init`, and add a blank import to
   `internal/plugins`.
