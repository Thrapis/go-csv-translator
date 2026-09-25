// Package plugins blank-imports every built-in game, translation backend and
// file format so their init functions register them. Import it once (for side
// effects) from main; nothing else should need to know the full list.
package plugins

import (
	_ "github.com/Thrapis/go-game-translator/internal/extract/crowdincsv"
	_ "github.com/Thrapis/go-game-translator/internal/extract/delimited"
	_ "github.com/Thrapis/go-game-translator/internal/extract/tcoaalcsv"
	_ "github.com/Thrapis/go-game-translator/internal/extract/tcoaaltxt"
	_ "github.com/Thrapis/go-game-translator/internal/game/cyberpunk2077"
	_ "github.com/Thrapis/go-game-translator/internal/game/taleworld"
	_ "github.com/Thrapis/go-game-translator/internal/game/tcoaal"
	_ "github.com/Thrapis/go-game-translator/internal/game/titanquest"
	_ "github.com/Thrapis/go-game-translator/internal/translate/google"
	_ "github.com/Thrapis/go-game-translator/internal/translate/lingvanex"
)
