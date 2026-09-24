// Package plugins blank-imports every built-in game, translation backend and
// file format so their init functions register them. Import it once (for side
// effects) from main; nothing else should need to know the full list.
package plugins

import (
	_ "github.com/Thrapis/go-csv-translator/internal/extract/delimited"
	_ "github.com/Thrapis/go-csv-translator/internal/extract/tcoaalcsv"
	_ "github.com/Thrapis/go-csv-translator/internal/extract/tcoaaltxt"
	_ "github.com/Thrapis/go-csv-translator/internal/game/taleworld"
	_ "github.com/Thrapis/go-csv-translator/internal/game/tcoaal"
	_ "github.com/Thrapis/go-csv-translator/internal/game/titanquest"
	_ "github.com/Thrapis/go-csv-translator/internal/translate/google"
	_ "github.com/Thrapis/go-csv-translator/internal/translate/lingvanex"
)
