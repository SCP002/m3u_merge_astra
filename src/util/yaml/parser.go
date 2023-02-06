package yaml

import (
	"fmt"
	"m3u_merge_astra/util/parse"
	"m3u_merge_astra/util/scan"
	"m3u_merge_astra/util/slice"
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"
)

// ValType represents YAML value type
type ValType uint8

const (
	None ValType = iota // Should be used to declare empty sections
	Scalar
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

// BadValueError represents error thrown if specified Node.ValType is incompatible with Node.Values
type BadValueError struct {
	ValType ValType
	Values  []string
	Reason  string
}

// Error is used to satisfy golang error interface
func (e BadValueError) Error() string {
	return e.Reason
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
// <afterPath> is formatted as "key.subkey".
//
// If <sectionEnd> is true, insert after the indented section end, not first line.
//
// Can return errors defined in this package: BadValueError, PathNotFoundError.
func Insert(input []byte, afterPath string, sectionEnd bool, node Node) ([]byte, error) {
	if node.ValType == None && len(node.Values) > 0 {
		msg := "None value type can't have values"
		return input, errors.Wrap(BadValueError{ValType: node.ValType, Values: node.Values, Reason: msg}, "Bad value")
	}
	if node.ValType == Scalar && len(node.Values) > 1 {
		msg := "Scalar value type can't have more than 1 value"
		return input, errors.Wrap(BadValueError{ValType: node.ValType, Values: node.Values, Reason: msg}, "Bad value")
	}
	if node.ValType != None && len(node.Values) == 0 {
		msg := "Only None value type can be used without values"
		return input, errors.Wrap(BadValueError{ValType: node.ValType, Values: node.Values, Reason: msg}, "Bad value")
	}

	output := []rune(string(input))

	step := 2
	output = setIndent(output, step)
	insertIdx, depth, err := insertIndex(output, afterPath, sectionEnd, step)
	if err != nil {
		return input, errors.Wrap(err, "Get insert location")
	}

	indent := strings.Repeat(" ", step * depth)
	newlineSeq := "\r\n"
	chunk := ""

	// Add top newline
	if node.StartNewline {
		chunk += newlineSeq
	}

	// Add comment
	for _, line := range node.HeadComment {
		chunk += indent + "# " + line + newlineSeq
	}

	// Add key
	chunk += indent + node.Key + ":"

	// Add values
	switch node.ValType {
	case None:
		chunk += newlineSeq
	case Scalar:
		chunk += " "
	case Sequence:
		chunk += newlineSeq
		node.Values = lo.Map(node.Values, func(line string, _ int) string {
			if strings.HasPrefix(line, "- ") {
				return indent + strings.Repeat(" ", step) + line
			}
			// If sequence value, add 2 spaces to align keys and values
			return "  " + indent + strings.Repeat(" ", step) + line
		})
	case List, Map:
		chunk += newlineSeq
		node.Values = lo.Map(node.Values, func(line string, _ int) string {
			return indent + strings.Repeat(" ", step) + line
		})
	}

	for _, line := range node.Values {
		chunk += line + newlineSeq
	}

	// Add bottom newline
	if node.EndNewline {
		chunk += newlineSeq
	}

	// Insert chunk into output
	output = slices.Insert(output, insertIdx, []rune(chunk)...)

	return []byte(string(output)), nil
}

// setIndent returns copy of <input> with the specified <tIndent> set
func setIndent(input []rune, tIndent int) []rune {
	// indentPair represents integer pair
	type indentPair struct {
		old int
		new int
	}

	var parentsIndents []indentPair

	// getParentIndent returns new indent of the parent of the <line> or 0 if not found (indentPair.new default value)
	getParentIndent := func(line string) int {
		// Find closest section header which old indent is lower than <line> has
		indent, _ := lo.Find(parentsIndents, func(parentIndent indentPair) bool {
			return parentIndent.old < parse.GetIndent(line)
		})
		return indent.new
	}

	listRx := regexp.MustCompile(`^ *(- )+`)

	// getHyphensAmount returns amount of starting "- " in the <line>
	getHyphensAmount := func(line string) int {
		hyphens := 0
		if matchList := listRx.FindStringSubmatch(line); len(matchList) > 0 {
			hyphens = strings.Count(matchList[0], "- ")
		}
		return hyphens
	}

	var output []rune
	prevLineHyphensAmount := 0

	sc := scan.New(input, 0)
	for sc.Lines(false) {
		trimLine := strings.TrimSpace(sc.Line)
		isFolder := strings.HasSuffix(trimLine, ":")
		isComment := strings.HasPrefix(trimLine, "#")
		hypensAmount := getHyphensAmount(trimLine)
		isSeqValue := !isFolder && !isComment && hypensAmount == 0 && prevLineHyphensAmount == 1

		cIndent := parse.GetIndent(sc.Line)
		parentIndent := getParentIndent(sc.Line)
		newIndent := parentIndent

		// TODO: Try to optimize
		if cIndent > 0 {
			newIndent += tIndent
			if isSeqValue {
				newIndent += 2 // Add 2 to align sequence keys and values
			}
		}
		if isFolder {
			parentsIndents = slice.Prepend(parentsIndents, indentPair{old: cIndent, new: newIndent})
		} else if hypensAmount > 1 {
			parentsIndents = slice.Prepend(parentsIndents, indentPair{old: cIndent, new: parentIndent + 2})
		}

		sc.Line = strings.Repeat(" ", newIndent) + strings.TrimLeft(sc.Line, " ")
		output = append(output, []rune(sc.Line)...)

		prevLineHyphensAmount = hypensAmount
	}

	return output
}

// insertIndex returns index of <input> pointing at the location where new item should be inserted by <path> and it's
// depth as the second value.
//
// If <sectionEnd> is true, returns index of the indented section end.
//
// If <path> is empty, returns length of <input>.
//
// To work properly, indentation used in <input> should be specified as <tIndent> (run setIndent() function).
//
// Returns 0, 0 and error if given <path> is not found in <input>.
func insertIndex(input []rune, path string, sectionEnd bool, tIndent int) (int, int, error) {
	err := PathNotFoundError{Path: path}

	if path == "" {
		return len(input), 0, nil // len == index + 1
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

	depth := 0
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
		isFolder := strings.HasSuffix(sc.Line, ":")
		// If folder and depth not grew
		if isFolder && cIndent <= lastIndent {
			return 0, 0, err
		}

		sc.Line = strings.ReplaceAll(sc.Line, `"`, ``)
		sc.Line = strings.ReplaceAll(sc.Line, `'`, ``)

		// If path entry with correct name is found and it's indent is equal to previous + 1 depth level
		if strings.HasPrefix(sc.Line, folders[folderIdx]) && cIndent == lastIndent + tIndent {
			if isFolder {
				depth++
			}
			// If last path entry
			if folderIdx == len(folders) - 1 {
				if sectionEnd && depth > 0 {
					depth--
				}
				if sectionEnd {
					return sectionEndIdx(sc.RuneIdx, cIndent), depth, nil
				}
				return sc.LineEndIdx + 1, depth, nil
			}
			lastIndent = cIndent
			folderIdx++
		}
	}

	return 0, 0, err
}
