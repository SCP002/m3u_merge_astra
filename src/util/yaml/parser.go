package yaml

import (
	"fmt"
	"m3u_merge_astra/util/parse"
	"m3u_merge_astra/util/scan"
	"m3u_merge_astra/util/slice"
	"reflect"
	"regexp"
	"sort"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
	"golang.org/x/exp/slices"
)

// PathNotFoundError represents error thrown if specified path not found in given YAML
type PathNotFoundError struct {
	Path string
}

// Error is used to satisfy golang error interface
func (e PathNotFoundError) Error() string {
	return fmt.Sprintf("Can not find the specified path: %v", e.Path)
}

// BadDataError represents error thrown if specified node data is incorrect
type BadDataError struct {
	Data   any
	Reason string
}

// Error is used to satisfy golang error interface
func (e BadDataError) Error() string {
	return e.Reason
}

// Node represents YAML comment, key and value
type Node struct {
	StartNewline bool // Add new line character(s) before content?
	HeadComment  []string
	Data         any  // Keys and values. Can be types from values.go
	EndNewline   bool // Add new line character(s) after content?
}

// Insert returns copy of the YAML bytes <input> with <node> inserted <afterPath>.
//
// <afterPath> is formatted as "key.subkey".
//
// If <sectionEnd> is true, insert after the indented section end, not first line.
//
// Can return errors defined in this package: BadDataError, PathNotFoundError.
func Insert(input []byte, afterPath string, sectionEnd bool, node Node) ([]byte, error) {
	// Return error if node data is not nil and keys or values are empty
	{
		errMsg := "Can not set empty key or value, use nil instead"
		err := errors.Wrap(BadDataError{Data: node.Data, Reason: errMsg}, "Validate node data")

		switch data := node.Data.(type) {
		case nil:
			//
		case Key:
			if data.Key == "" {
				return input, err
			}
		case Scalar:
			if data.Key == "" || data.Value == "" {
				return input, err
			}
		case Sequence:
			if data.Key == "" || len(data.Sets) == 0 {
				return input, err
			}
			for _, set := range data.Sets {
				if len(set) == 0 {
					return input, err
				}
				for _, pair := range set {
					if pair.Key == "" || pair.Value == "" {
						return input, err
					}
				}
			}
		case List:
			if data.Key == "" || len(data.Values) == 0 {
				return input, err
			}
			for _, value := range data.Values {
				if value.Value == "" {
					return input, err
				}
			}
		case NestedList:
			if data.Key == "" {
				return input, err
			}
			flatTree, _ := flatten(data.Tree)
			for i, branch := range flatTree {
				// Only root (first element) is allowed without values
				if i > 0 && branch.Value.Value == "" {
					return input, err
				}
			}
		case Map:
			if data.Key == "" || len(data.Map) == 0 {
				return input, err
			}
			for key, value := range data.Map {
				if key == "" || value.Value == "" {
					return input, err
				}
			}
		default:
			current := reflect.TypeOf(data).Name()
			allowed := []string{
				"nil",
				reflect.TypeOf(Key{}).Name(),
				reflect.TypeOf(Scalar{}).Name(),
				reflect.TypeOf(Sequence{}).Name(),
				reflect.TypeOf(List{}).Name(),
				reflect.TypeOf(NestedList{}).Name(),
				reflect.TypeOf(Map{}).Name(),
			}
			errMsg := fmt.Sprintf("Invalid data type: %v, allowed types are: %v", current, allowed)
			err := errors.Wrap(BadDataError{Data: node.Data, Reason: errMsg}, "Validate node data")
			return input, err
		}
	}

	// Prepare and get insert location
	output := []rune(string(input))

	step := 2
	output = setIndent(output, step)
	insertIdx, depth, err := insertIndex(output, afterPath, sectionEnd, step)
	if err != nil {
		return input, errors.Wrap(err, "Get insert location")
	}

	indent := strings.Repeat(" ", step * depth)
	newlineSeq := "\r\n"
	commentSeq := "# "
	listValSeq := "- "
	listAlignSeq := "  "
	keyValDelimSeq := ": "
	chunk := ""

	// Add top newline
	if node.StartNewline {
		chunk += newlineSeq
	}

	// Add comment
	for _, line := range node.HeadComment {
		chunk += indent + commentSeq + line + newlineSeq
	}

	// Add keys and values
	switch data := node.Data.(type) {
	case nil:
		//
	case Key:
		chunk += indent
		if data.Commented {
			chunk += commentSeq
		}
		chunk += data.Key + ":" + newlineSeq
	case Scalar:
		chunk += indent
		if data.Commented {
			chunk += commentSeq
		}
		chunk += data.Key + keyValDelimSeq + data.Value + newlineSeq
	case Sequence:
		chunk += indent + data.Key + ":" + newlineSeq
		for _, set := range data.Sets {
			for i, pair := range set {
				chunk += indent + strings.Repeat(" ", step)
				if pair.Commented {
					chunk += commentSeq
				}
				if i == 0 {
					chunk += listValSeq
				} else {
					chunk += listAlignSeq
				}
				chunk += pair.Key + keyValDelimSeq + pair.Value + newlineSeq
			}
		}
	case List:
		chunk += indent + data.Key + ":" + newlineSeq
		for _, value := range data.Values {
			chunk += indent + strings.Repeat(" ", step)
			if value.Commented {
				chunk += commentSeq
			}
			chunk += listValSeq + value.Value + newlineSeq
		}
	case NestedList:
		// TODO: Try to optimize (look at flatten()?)
		// TODO: Merge NestedList with List?
		chunk += indent + data.Key + ":" + newlineSeq
		flatTree, maxDepth := flatten(data.Tree)
		for i, branch := range flatTree {
			// Root (first element) should not contain values, skip it
			if i == 0 { // TODO: Remove if changed in flatten
				continue
			}
			// Add extra spaces on top of regular indent to align deep values
			chunk += indent + strings.Repeat(" ", step) + strings.Repeat(listAlignSeq, branch.Depth - 1)
			if branch.Value.Commented {
				chunk += commentSeq
			}
			// Add hyphens based on how deep value is
			chunk += strings.Repeat(listValSeq, maxDepth-branch.Depth + 1) + branch.Value.Value + newlineSeq
		}
	case Map:
		chunk += indent + data.Key + ":" + newlineSeq
		// Transform map to slice and sort it to ensure key order
		pairs := lo.MapToSlice(data.Map, func(key string, value Value) Pair {
			return Pair{Key: key, Value: value.Value, Commented: value.Commented}
		})
		sort.SliceStable(pairs, func(i, j int) bool {
			return pairs[i].Key < pairs[j].Key
		})
		for _, pair := range pairs {
			chunk += indent + strings.Repeat(" ", step)
			if pair.Commented {
				chunk += commentSeq
			}
			chunk += pair.Key + keyValDelimSeq + pair.Value + newlineSeq
		}
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

		if cIndent > 0 {
			newIndent += tIndent
			if isSeqValue {
				newIndent += 2 // Add 2 to align sequence keys and values
			}
		}
		if isFolder {
			parentsIndents = slice.Prepend(parentsIndents, indentPair{old: cIndent, new: newIndent})
		} else if hypensAmount > 1 {
			// Indent nested lists with two spaces regardless of <tIndent>
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

// flatten returns <tree> as a single level deep slice and maximum depth of the <tree> as the second value
func flatten(tree ValueTree) (out []ValueTree, maxDepth int) {
	if tree.Value.Value != "" {
		tree.Depth++
	}
	maxDepth = tree.Depth
	out = append(out, tree)
	for _, child := range tree.Children {
		child.Depth = tree.Depth
		var flatTree []ValueTree
		flatTree, maxDepth = flatten(child)
		out = append(out, flatTree...)
	}
	return out, maxDepth
}
