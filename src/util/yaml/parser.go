package yaml

import (
	"fmt"
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

// ...
func pathIndex(input []rune, path string) (int, error) {
	if len(input) == 0 {
		return 0, PathNotFoundError{Path: path}
	}
	trimPath := strings.TrimRight(path, ":")
	if trimPath == "" {
		return len(input), nil // len == index + 1
	}

	folders := lo.Map(strings.Split(trimPath, "."), func(folder string, _ int) string {
		return folder + ":"
	})

	folderIdx := 0
	line := ""
	for read, char := range input {
		line += string(char)

		if char != '\n' {
			continue
		}

		line = strings.Trim(line, " ")

		if strings.HasPrefix(line, "#") {
			line = ""
			continue
		}

		if strings.HasPrefix(line, folders[folderIdx]) {
			if folderIdx == len(folders)-1 {
				return read + 1, nil // read == index, add 1
			} else {
				folderIdx++
			}
		}

		line = ""
	}

	return 0, PathNotFoundError{Path: path}
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
