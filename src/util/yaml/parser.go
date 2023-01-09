package yaml

import (
	"fmt"
	"m3u_merge_astra/util/parse"
	"m3u_merge_astra/util/scan"
	"m3u_merge_astra/util/slice"
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

// Node represents YAML comment, key and value
type Node struct {
	StartNewline bool // Add blank line before content?
	HeadComment  []string
	Key          string
	ValType      ValType // Determines if value will be inserted at the new line after the key and it's indentations
	Values       []string
	EndNewline   bool // Add blank line after content?
}

// Insert returns copy of the YAML bytes <input> with <node> inserted <afterPath>.
//
// <afterPath> is formatted as "key.subkey:".
//
// If <sectionEnd> is true, insert after the indented section end, not first line.
func Insert(input []byte, afterPath string, sectionEnd bool, node Node) ([]byte, error) { // TODO: This
	output := []rune(string(input))

	indent := 2
	output = setIndent(output, indent)

	//

	return []byte(string(output)), nil
}

// setIndent returns copy of <input> with the specified <tIndent> set
func setIndent(input []rune, tIndent int) []rune {
	// indentPair represents integer pair
	type indentPair struct {
		old int
		new int
	}

	var foldersIndents []indentPair

	// getParentIndent returns new indent of the parent of the <line>
	getParentIndent := func(line string) int {
		// Find closest folder (section header) which old indent is lower than <line> has
		indent, _ := lo.Find(foldersIndents, func(folderIndent indentPair) bool {
			return folderIndent.old < parse.GetIndent(line)
		})
		return indent.new
	}

	var output []rune
	prevLineHyphenPrefix := false

	sc := scan.New(input, 0)
	for sc.Lines(false) {
		trimLine := strings.TrimSpace(sc.Line)
		isFolder := strings.HasSuffix(trimLine, ":")
		hasHyphenPrefix := strings.HasPrefix(trimLine, "-")
		isComment := strings.HasPrefix(trimLine, "#")
		isSeqValue := !isFolder && !isComment && !hasHyphenPrefix && prevLineHyphenPrefix

		cIndent := parse.GetIndent(sc.Line)
		newIndent := 0

		if cIndent > 0 {
			newIndent = getParentIndent(sc.Line) + tIndent
			if isSeqValue {
				newIndent += 2 // Add 2 to align sequence keys and values
			}
		}
		if isFolder {
			foldersIndents = slice.Prepend(foldersIndents, indentPair{old: cIndent, new: newIndent})
		}

		sc.Line = strings.Repeat(" ", newIndent) + strings.TrimLeft(sc.Line, " ")
		output = append(output, []rune(sc.Line)...)

		prevLineHyphenPrefix = hasHyphenPrefix
	}

	return output
}

// insertIndex returns index of <input> pointing at the location where new item should be inserted by <path>.
//
// If <sectionEnd> is true, returns index of the indented section end.
//
// If <path> is empty, returns length of <input>.
//
// To work properly, indentation used in <input> should be specified as <tIndent> (run setIndent() function).
//
// Returns 0 and error if given <path> is not found in <input>.
func insertIndex(input []rune, path string, sectionEnd bool, tIndent int) (int, error) {
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

	// sectionEndIdx returns the starting index of the first line found in <input> beginning from the <startIdx> if it's
	// indent equals to or lower than <indent>.
	//
	// Return ending index of the last line encountered if no appropriate index found.
	sectionEndIdx := func(startIdx, indent int) int {
		sc := scan.New(input, startIdx)
		for sc.Lines(true) {
			if parse.GetIndent(sc.Line) <= indent {
				return sc.LineStartIdx + 1
			}
		}
		return sc.LineEndIdx + 1
	}

	folderIdx := 0
	lastIndent := -tIndent // Set initial indent to negative target so first folder with indent 0 will have proper depth
	sc := scan.New(input, 0)
	for sc.Lines(true) {
		cIndent := parse.GetIndent(sc.Line)
		sc.Line = strings.TrimSpace(sc.Line)

		// If comment, continue. Guard in case if path folder starts with #.
		if strings.HasPrefix(sc.Line, "#") {
			continue
		}
		// If folder and depth not grew
		if strings.HasSuffix(sc.Line, ":") && cIndent <= lastIndent {
			return 0, err
		}

		sc.Line = strings.ReplaceAll(sc.Line, `"`, ``)
		sc.Line = strings.ReplaceAll(sc.Line, `'`, ``)

		// If folder with correct name is found and it's indent is equal to previous + 1 depth level
		if strings.HasPrefix(sc.Line, folders[folderIdx]) && cIndent == lastIndent + tIndent {
			if folderIdx == len(folders) - 1 {
				return lo.Ternary(sectionEnd, sectionEndIdx(sc.RuneIdx, cIndent), sc.LineEndIdx + 1), nil
			}
			lastIndent = cIndent
			folderIdx++
		}
	}

	return 0, err
}
