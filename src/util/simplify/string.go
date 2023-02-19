package simplify

import (
	"regexp"
	"strings"
)

// regEx represents every character to remove from name before comparsion
var regEx regexp.Regexp = *regexp.MustCompile("[" +
	// All control characters such as \x00
	"[:cntrl:]" +
	// All blank (whitespace) characters
	"[:space:]" +
	// All punctuation characters (except the + symbol: prevents 'Name 2' == 'Name (+2)')
	"-!\"#$%&'()*,./:;<=>?@[\\]^_`{|}~" +
	"]+")

// Name returns simplified <inp> (lowercase, no special characters)
func Name(inp string) string {
	out := strings.ToLower(inp)
	return regEx.ReplaceAllString(out, "")
}
