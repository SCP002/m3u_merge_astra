package slice

import (
	"regexp"

	"github.com/samber/lo"
)

// RxMatchAny returns true if <rx> matches any of <elms>
func RxMatchAny(rx regexp.Regexp, elms... string) bool {
	return lo.ContainsBy(elms, func(elm string) bool {
		return rx.MatchString(elm)
	})
}

// AnyRxMatch returns true if any element of <rxList> matches <tElm>
func AnyRxMatch(rxList []regexp.Regexp, tElm string) bool {
	return lo.ContainsBy(rxList, func(rx regexp.Regexp) bool {
		return rx.MatchString(tElm)
	})
}

// AnyRxMatchAny returns true if any element of <rxList> matches any of <elms>
func AnyRxMatchAny(rxList []regexp.Regexp, elms... string) bool {
	return lo.ContainsBy(elms, func(elm string) bool {
		return AnyRxMatch(rxList, elm)
	})
}
