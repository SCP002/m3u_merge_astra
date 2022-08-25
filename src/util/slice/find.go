package slice

import (
	"m3u_merge_astra/cfg"
	"m3u_merge_astra/util/conv"

	"github.com/samber/lo"
)

// FindNamed returns <list> entry, it's index and true if .Name() of it matching the <name> or empty object, -1
// and false if not found.
//
// Both names standartized before comparsion using transliteration settings from <cfg>.
func FindNamed[T Named](cfg cfg.General, list []T, name string) (T, int, bool) {
	return lo.FindIndexOf(list, func(o T) bool {
		return conv.IsNameSame(cfg, o.GetName(), name)
	})
}

// EverySimilar runs callback <cb> for every entry of <list> starting from index <startIdx> if it's .Name() matching the
// <name>.
//
// Both names standartized before comparsion using transliteration settings from <cfg>.
func EverySimilar[T Named](cfg cfg.General, list []T, name string, startIdx int, cb func(foundObj T, foundIdx int)) {
	for currIdx := startIdx; currIdx < len(list); currIdx++ {
		currElm := list[currIdx]
		if conv.IsNameSame(cfg, currElm.GetName(), name) {
			cb(currElm, currIdx)
		}
	}
}

// GetSimilar returns copy of <list> only with entries whose .Name() matching the <name>.
//
// Both names standartized before comparsion using transliteration settings from <cfg>.
func GetSimilar[T Named](cfg cfg.General, list []T, name string) []T {
	return lo.Filter(list, func(elm T, idx int) bool {
		return conv.IsNameSame(cfg, elm.GetName(), name)
	})
}

// HasAnySimilar returns true if <list> contains entry with .Name() matching any of <names>.
//
// Both names standartized before comparsion using transliteration settings from <cfg>.
func HasAnySimilar[T Named](cfg cfg.General, list []T, names... string) bool {
	return lo.ContainsBy(list, func(elm T) bool {
		return lo.SomeBy(names, func(name string) bool {
			return conv.IsNameSame(cfg, elm.GetName(), name)
		})
	})
}
