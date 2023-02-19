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
	if cfg.NameAliases && firstSimpleAlias(lName, cfg.NameAliasList) == firstSimpleAlias(rName, cfg.NameAliasList) {
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

// firstSimpleAlias returns first simplified alias for <name> from <aliases> or simple <name> if not found
func firstSimpleAlias(name string, aliases [][]string) string {
	name = simplify.Name(name)
	for _, set := range aliases {
		simpleSet := lo.Map(set, func(alias string, _ int) string {
			return simplify.Name(alias)
		})
		if lo.Contains(simpleSet, name) {
			return simpleSet[0]
		}
	}
	return name
}
