package parse

import (
	"regexp"
	"strings"

	"github.com/samber/lo"
)

// indentRx represents regex which 1st capturing group contains 1 or more space characters in the beginnging of the line
var indentRx = regexp.MustCompile(`^( +).*`)

// GetIndent returns amount of space characters in the beginning of the <line>
func GetIndent(line string) int {
	if matches := indentRx.FindStringSubmatch(line); len(matches) > 1 {
		return len(matches[1])
	}
	return 0
}

// LastPathItem returns last item in <path> split by <delim> or <path> if <delim> is empty or last item is empty
func LastPathItem(path, delim string) string {
	if delim == "" {
		return path
	}
	item, _ := lo.Last(strings.Split(path, delim))
	return lo.Ternary(item == "", path, item)
}
