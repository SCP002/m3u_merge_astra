package slice

import (
	"reflect"

	"github.com/samber/lo"
)

// Prepend returns new slice with <elm> added to the beginning of <inp>
func Prepend[T any](inp []T, elm T) []T {
	return append([]T{elm}, inp...)
}

// AppendNew returns <inp> with every element of <elms> added to the end of <inp> if it's not in <inp> and runs
// <callback>.
func AppendNew[T comparable](inp []T, callback func(T), elms ...T) []T {
	for _, elm := range elms {
		if !lo.Contains(inp, elm) {
			inp = append(inp, elm)
			callback(elm)
		}
	}
	return inp
}

// RemoveLast returns new slice with the last occurence of <tElm> removed from <inp>
func RemoveLast[T any](inp []T, tElm T) (out []T) {
	_, tIdx, _ := lo.FindLastIndexOf(inp, func(cElm T) bool {
		return reflect.DeepEqual(tElm, cElm)
	})
	for cIdx, e := range inp {
		if cIdx != tIdx {
			out = append(out, e)
		}
	}
	return
}
