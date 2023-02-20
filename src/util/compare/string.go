package compare

import (
	"m3u_merge_astra/cfg"
	"m3u_merge_astra/util/simplify"

	"github.com/samber/lo"
)

// IsNameSame returns true if standardized <lName> is equal to standardized <rName> using transliteration settings and
// aliases from <cfg>.
func IsNameSame(cfg cfg.General, lName, rName string) bool {
	lName = simplify.Name(lName)
	rName = simplify.Name(rName)
	if lName == rName {
		return true
	}
	if cfg.SimilarTranslit && remap(lName, cfg.SimilarTranslitMap) == remap(rName, cfg.SimilarTranslitMap) {
		return true
	}
	if cfg.FullTranslit && remap(lName, cfg.FullTranslitMap) == remap(rName, cfg.FullTranslitMap) {
		return true
	}
	if cfg.NameAliases && firstAlias(lName, cfg.SimpleNameAliasList) == firstAlias(rName, cfg.SimpleNameAliasList) {
		return true
	}
	return false
}

// remap returns remapped <inp> using <dict>
func remap(inp string, dict map[string]string) (out string) {
	for _, oldChar := range inp {
		newChar := dict[string(oldChar)]
		if newChar != "" {
			out += newChar
		} else {
			out += string(oldChar)
		}
	}
	return
}

// firstAlias returns first alias for <name> from <aliases> or <name> if not found.
//
// This function assumes both <name> and <aliases> arguments are simplified before use:
//
// See simplify.Name() and cfg.Root.General.SimplifyAliases().
func firstAlias(name string, aliases [][]string) string {
	for _, set := range aliases {
		if lo.Contains(set, name) {
			return set[0]
		}
	}
	return name
}
