package slice

import (
	"regexp"

	"github.com/samber/lo"
)

// RxAnyMatch returns true if any element of <rxList> matches <tElm>
func RxAnyMatch(rxList []regexp.Regexp, tElm string) bool {
	return lo.ContainsBy(rxList, func(rx regexp.Regexp) bool {
		return rx.MatchString(tElm)
	})
}
