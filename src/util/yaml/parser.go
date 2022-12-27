package yaml

import (
	"fmt"
	"m3u_merge_astra/util/scan"
	"regexp"
	"strings"

	"github.com/samber/lo"
)

// ValType represents YAML value type
type ValType uint8

const (
	Scalar ValType = iota
	Sequence
	List
	Map
)

// PathNotFoundError represents error thrown if specified path not found in given YAML
type PathNotFoundError struct {
	Path string
}

// Error is used to satisfy golang error interface
func (e PathNotFoundError) Error() string {
	return fmt.Sprintf("Can not find the specified path: %v", e.Path)
}

// insertIndex returns index of <input> pointing at the location where new item should be inserted by <path>.
//
// If <path> is empty, returns length of <input>.
//
// Returns 0 and error if given <path> is not found in <input>.
func insertIndex(input []rune, path string) (int, error) {
	err := PathNotFoundError{Path: path}

	if len(input) == 0 {
		return 0, err
	}

	path = strings.TrimRight(path, ":")
	if path == "" {
		return len(input), nil // len == index + 1
	}

	folders := lo.Map(strings.Split(path, "."), func(folder string, _ int) string {
		return folder + ":"
	})

	indentRx := regexp.MustCompile(`^( +).*`)

	// getIndent returns amount of space characters in the beginning of the <line>
	getIndent := func(line string) int {
		if matches := indentRx.FindStringSubmatch(line); len(matches) > 1 {
			return len(matches[1])
		}
		return 0
	}

	// sectionEndIdx returns the starting index of the first line found in <input> beginning from the <startIdx> if it's
	// indent equals to or lower than <tIndent>.
	//
	// Return ending index of the last line encountered if no appropriate index found.
	sectionEndIdx := func(startIdx, tIndent int) int {
		sc := scan.New(input, startIdx)
		for sc.Lines() {
			if getIndent(sc.Line) <= tIndent {
				return sc.LineStartIdx + 1
			}
		}
		return sc.LineEndIdx + 1
	}

	folderIdx := 0
	lastIndent := -1
	sc := scan.New(input, 0)
	for sc.Lines() {
		indent := getIndent(sc.Line)
		sc.Line = strings.TrimSpace(sc.Line)

		if strings.HasPrefix(sc.Line, "#") { // Guard in case if path folder starts with #
			continue
		}

		// If folder with correct name is found and it's indent is bigger than previous
		if strings.HasPrefix(sc.Line, folders[folderIdx]) && lastIndent < indent {
			if folderIdx == len(folders)-1 {
				return sectionEndIdx(sc.RuneIdx, indent), nil
			}
			lastIndent = indent
			folderIdx++
		}
	}

	return 0, err
}

// Insert returns copy of the YAML bytes <input> with <headComment>, <key> and <values> pasted <afterPath>.
//
// <afterPath> is formatted as "key.subkey:".
//
// <valType> determines if value will be pasted at the new line after the key and it's indentations.
func Insert(input []byte,
	afterPath string,
	headComment []string,
	key string,
	valType ValType,
	values ...string) ([]byte, error) { // TODO: This
	return input, nil
}
