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
| `destination.folderNameMap` | rename sub-folders on output, e.g. `ru: be` |
| `language.source` / `language.target` | translation direction |
| `translators` | ordered fallback chain, e.g. `[lingvanex, google]` |
| `lingvanex.manage` | when true, start the Lingvanex server on launch and stop it on exit |
| `multiRowReplicas` | translate consecutive same-tag rows as one sentence |
| `parasitizing` | reuse an existing human translation file instead of MT |
| `postMerge` | after translating, stitch the output tree back into one CSV |

Ctrl+C stops the run promptly; partial output is left on disk and a managed
Lingvanex server is shut down.

## Lingvanex server

`third_party/lingvanex-server/` holds the local Python translation server
(CTranslate2 + SentencePiece). With `lingvanex.manage: true` the app spawns and
supervises it. To run it yourself instead, set `manage: false` and start
`third_party/lingvanex-server/start.bat` (or `py server.py`). Model files are
gitignored.

## Extra tools

```bash
go run ./tools/csvsplit -in export.csv          # one multi-section CSV -> per-section files
go run ./tools/csvmerge -in folder -out out.csv # per-section files -> one multi-section CSV
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
    delimited          generic "key<delim>value"      (id: "delimited")
    tcoaalcsv          TCOAAL export CSV               (id: "tcoaal-csv")
  translate            Translator interface + backend registry + fallback Chain
    lingvanex          HTTP client for the local server
    google             free Google Translate fallback
  backend/lingvanex    Lingvanex server process supervisor
  game                 Game interface + optional capabilities + registry
    tcoaal / taleworld / titanquest
  textutil             small string helpers
  csvtool              split/merge implementation
  plugins              blank-imports every game/format/backend to register them
tools/csvsplit, tools/csvmerge
```

## Adding a game

1. New package under `internal/game/<name>/`.
2. Implement `game.Game` (`Name`, `Analyzer`, `DefaultFormat`, `DefaultDelimiter`).
   Optionally implement `game.ReplicaGrouper` and/or `game.Parasitizer`.
3. Implement `markup.Analyzer` for the engine's markup (`Analyze`, `Translatable`,
   `Render`; keep `Render(Analyze(s)) == s`).
4. If the file layout is new, add an `extract.Format` and register it; otherwise
   return an existing id (`"delimited"`, `"tcoaal-csv"`) from `DefaultFormat`.
5. `game.Register(New())` in the package `init`, and add a blank import to
   `internal/plugins`.
