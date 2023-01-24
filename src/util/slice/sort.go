package slice

import (
	"m3u_merge_astra/util/copier"
	"sort"
)

// Sort returns copy of <list> sorted by Name field
func Sort[T Named](list []T) (out []T) {
	out = copier.MustDeep(list)

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].GetName() < out[j].GetName()
	})

	return
}
