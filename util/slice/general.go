package slice

import (
	"strings"

	"github.com/google/go-cmp/cmp"
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
			if callback != nil {
				callback(elm)
			}
		}
	}
	return inp
}

// RemoveLast returns new slice with the last occurence of <tElm> removed from <inp>
func RemoveLast[T any](inp []T, tElm T) (out []T) {
	_, tIdx, _ := lo.FindLastIndexOf(inp, func(cElm T) bool {
		return cmp.Equal(tElm, cElm)
	})
	for cIdx, e := range inp {
		if cIdx != tIdx {
			out = append(out, e)
		}
	}
	return
}

// RemoveFirst returns new slice with the first occurence of <tElm> removed from <inp>
func RemoveFirst[T any](inp []T, tElm T) (out []T) {
	_, tIdx, _ := lo.FindIndexOf(inp, func(cElm T) bool {
		return cmp.Equal(tElm, cElm)
	})
	for cIdx, e := range inp {
		if cIdx != tIdx {
			out = append(out, e)
		}
	}
	return
}

// Filled returns new slice of <times> amount of <elm>
func Filled[T any](elm T, times int) []T {
	out := []T{}
	for i := 0; i < times; i++ {
		out = append(out, elm)
	}
	return out
}

// ContainsAny returns true if <inp> contains any of <elms>
func ContainsAny(inp string, elms ...string) bool {
	return lo.SomeBy(elms, func(elm string) bool {
		return strings.Contains(inp, elm)
	})
}

// HasAnyPrefix returns true if <inp> has any prefix from <prefixes>
func HasAnyPrefix(inp string, prefixes ...string) bool {
	return lo.SomeBy(prefixes, func(prefix string) bool {
		return strings.HasPrefix(inp, prefix)
	})
}

// IsAllEmpty reurns true if every slice in <inp> is empty
func IsAllEmpty[T any](inp ...[]T) bool {
	return len(lo.Flatten(inp)) == 0
}

// MapFindDuplBy returns <inp> with every element transformed by <transform>.
//
// <transform> invokes with element, it's index and boolean whether element is a duplicate or not.
//
// <by> is used to generate the criterion by which uniqueness is computed.
func MapFindDuplBy[T, R any, U comparable](inp []T, by func(elm T) U, transform func(elm T, idx int, dupl bool) R) []R {
	result := make([]R, len(inp))
	seen := make(map[U]struct{}, len(inp))

	for idx, elm := range inp {
		key := by(elm)

		_, dupl := seen[key]

		result[idx] = transform(elm, idx, dupl)
		seen[key] = struct{}{}
	}

	return result
}
