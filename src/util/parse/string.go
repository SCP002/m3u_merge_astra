package parse

import "regexp"

// indentRx represents regex which 1st capturing group contains 1 or more space characters in the beginnging of the line
var indentRx = regexp.MustCompile(`^( +).*`)

// GetIndent returns amount of space characters in the beginning of the <line>
func GetIndent(line string) int {
	if matches := indentRx.FindStringSubmatch(line); len(matches) > 1 {
		return len(matches[1])
	}
	return 0
}
