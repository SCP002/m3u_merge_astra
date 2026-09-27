package rnd

import (
	"github.com/samber/lo"
)

// String returns random string
func String(length uint, uppercase bool, numbers bool) string {
	if length == 0 {
		return ""
	}

	charset := lo.LowerCaseLettersCharset
	if uppercase {
		charset = append(charset, lo.UpperCaseLettersCharset...)
	}
	if numbers {
		charset = append(charset, lo.NumbersCharset...)
	}
	return lo.RandomString(int(length), charset)
}
