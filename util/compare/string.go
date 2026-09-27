package compare

import (
	"m3u_merge_astra/cfg"
	"m3u_merge_astra/util/simplify"
	"strings"

	"github.com/samber/lo"
)

// IsNameSame returns true if standardized <lName> is equal to standardized <rName> using transliteration settings and
// aliases from <cfg>.
func IsNameSame(cfg cfg.General, lName, rName string) bool {
	if lName == rName {
		return true
	}

	lSimpleName := simplify.Name(lName)
	rSimpleName := simplify.Name(rName)
	if lSimpleName == rSimpleName {
		return true
	}

	if cfg.SimilarTranslit {
		if remap(lSimpleName, cfg.SimilarTranslitMap) == remap(rSimpleName, cfg.SimilarTranslitMap) {
			return true
		}
	}

	if cfg.FullTranslit {
		if remap(lSimpleName, cfg.FullTranslitMap) == remap(rSimpleName, cfg.FullTranslitMap) {
			return true
		}
	}

	if cfg.NameAliases {
		if firstAlias(lSimpleName, cfg.SimpleNameAliasList) == firstAlias(rSimpleName, cfg.SimpleNameAliasList) {
			return true
		}
	}

	return false
}

// remap returns remapped <inp> using <dict>
func remap(inp string, dict map[string]string) string {
	var sb strings.Builder
	for _, oldChar := range inp {
		newChar := dict[string(oldChar)]
		if newChar != "" {
			sb.WriteString(newChar)
		} else {
			sb.WriteRune(oldChar)
		}
	}
	return sb.String()
}

// firstAlias returns first alias for <name> from <aliases> or <name> if not found
func firstAlias(name string, aliases [][]string) string {
	for _, set := range aliases {
		if lo.Contains(set, name) {
			return set[0]
		}
	}
	return name
}
