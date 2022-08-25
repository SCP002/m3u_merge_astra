package slice

import (
	"m3u_merge_astra/util/copier"
	"sort"

	"github.com/sirupsen/logrus"
)

// Sort returns copy of <list> sorted by Name field, printing the description of <what> it sorts using <log>
func Sort[T Named](log *logrus.Logger, list []T, what string) (out []T) {
	log.Infof("Sorting %v\n", what)

	out = copier.PDeep(list)

	sort.SliceStable(out, func(i, j int) bool {
		return out[i].GetName() < out[j].GetName()
	})

	return
}
