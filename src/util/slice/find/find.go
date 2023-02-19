package find

import (
	"m3u_merge_astra/cfg"
	"m3u_merge_astra/util/compare"
	"m3u_merge_astra/util/slice"

	"github.com/samber/lo"
)

// IndexOrElse returns unmodified <list>, <list> entry and it's index if <predicate> returns true or <list> with
// <fallback>, <fallback> and it's index (len - 1) if not found.
func IndexOrElse[T any](list []T, fallback T, predicate func(elm T) bool) ([]T, T, int) {
	elm, idx, found := lo.FindIndexOf(list, predicate)
	if !found {
		list = append(list, fallback)
		return list, fallback, len(list) - 1
	}
	return list, elm, idx
}

// Named returns <list> entry, it's index and true if .Name() of it matching the <name> or empty object, -1 and false if
// not found.
//
// Both names standartized before comparsion using transliteration settings from <cfg>.
func Named[T slice.Named](cfg cfg.General, list []T, name string) (T, int, bool) {
	return lo.FindIndexOf(list, func(o T) bool {
		return compare.IsNameSame(cfg, o.GetName(), name)
	})
}

// EverySimilar runs callback <cb> for every entry of <list> starting from index <start> if it's .Name() matching the
// <name>.
//
// Both names standartized before comparsion using transliteration settings from <cfg>.
func EverySimilar[T slice.Named](cfg cfg.General, list []T, name string, start int, cb func(foundObj T, foundIdx int)) {
	for currIdx := start; currIdx < len(list); currIdx++ {
		currElm := list[currIdx]
		if compare.IsNameSame(cfg, currElm.GetName(), name) {
			cb(currElm, currIdx)
		}
	}
}

// GetSimilar returns copy of <list> only with entries whose .Name() matching the <name>.
//
// Both names standartized before comparsion using transliteration settings from <cfg>.
func GetSimilar[T slice.Named](cfg cfg.General, list []T, name string) []T {
	return lo.Filter(list, func(elm T, idx int) bool {
		return compare.IsNameSame(cfg, elm.GetName(), name)
	})
}

// HasAnySimilar returns true if <list> contains entry with .Name() matching any of <names>.
//
// Both names standartized before comparsion using transliteration settings from <cfg>.
func HasAnySimilar[T slice.Named](cfg cfg.General, list []T, names ...string) bool {
	return lo.ContainsBy(list, func(elm T) bool {
		return lo.SomeBy(names, func(name string) bool {
			return compare.IsNameSame(cfg, elm.GetName(), name)
		})
	})
}
