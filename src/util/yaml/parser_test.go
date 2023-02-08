package yaml

import (
	"m3u_merge_astra/util/copier"
	"m3u_merge_astra/util/slice"
	"os"
	"testing"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"
)

func TestPathNotFoundError(t *testing.T) {
	err := error(PathNotFoundError{Path: "a.b.c"})
	assert.Exactly(t, "Can not find the specified path: a.b.c", err.Error())
}

func TestBadValueError(t *testing.T) {
	err := error(BadValueError{Reason: "unknown"})
	assert.Exactly(t, "unknown", err.Error())
}

func TestInsert(t *testing.T) {
	input, err := os.ReadFile("insert_input_test.yaml")
	assert.NoError(t, err, "should read input file")
	inputOriginal := copier.TestDeep(t, input)

	// Error cases (bad value or path can not be found)
	afterPath := ""
	node := Node{Key: "new_key", ValType: None, Values: []string{"a"}}
	output, err := Insert(input, afterPath, false, node)
	assert.ErrorAs(t, err, &BadValueError{}, "should return bad value error if None type has value")
	assert.Exactly(t, input, output, "on error, output should stay the same as input")

	afterPath = "key"
	node = Node{Key: "new_key", ValType: Scalar, Values: []string{"a", "b"}}
	output, err = Insert(input, afterPath, false, node)
	assert.ErrorAs(t, err, &BadValueError{}, "should return bad value error if Scalar has more than 1 value")
	assert.Exactly(t, input, output, "on error, output should stay the same as input")

	afterPath = "nested_section"
	node = Node{Key: "new_key", ValType: Map}
	output, err = Insert(input, afterPath, false, node)
	assert.ErrorAs(t, err, &BadValueError{}, "should return bad value error if not None type has no values")
	assert.Exactly(t, input, output, "on error, output should stay the same as input")

	afterPath = "unknown_root_path"
	node = Node{Key: "new_key", ValType: Scalar, Values: []string{"true"}}
	output, err = Insert(input, afterPath, false, node)
	assert.ErrorIs(t, PathNotFoundError{Path: afterPath}, errors.UnwrapAll(err), "should return unknown path error")
	assert.Exactly(t, input, output, "on error, output should stay the same as input")

	afterPath = "sequences_section.unknown_subkey"
	node = Node{Key: "new_key", ValType: Sequence, Values: []string{"- a: 'b'", "c: 'd'"}}
	output, err = Insert(input, afterPath, false, node)
	assert.ErrorIs(t, PathNotFoundError{Path: afterPath}, errors.UnwrapAll(err), "should return unknown path error")
	assert.Exactly(t, input, output, "on error, output should stay the same as input")

	afterPath = "lists_section.list.item_2"
	node = Node{Key: "new_key", ValType: Scalar, Values: []string{"true"}}
	output, err = Insert(input, afterPath, false, node)
	assert.ErrorIs(t, PathNotFoundError{Path: afterPath}, errors.UnwrapAll(err),
		"should not resolve node value as proper path")
	assert.Exactly(t, input, output, "on error, output should stay the same as input")

	// Overflow check, should not panic
	node = Node{Key: "new_key", ValType: List, Values: slice.Filled("- a", 10000)}
	_, err = Insert(input, "", false, node)
	assert.NoError(t, err, "should not return error")

	// Regular behavior
	// First change
	node = Node{
		StartNewline: true,
		HeadComment:  []string{"New comment"},
		Key:          "new_key",
		ValType:      Scalar,
		Values:       []string{"'value_1'"},
	}
	output, err = Insert(input, "key", false, node)
	assert.NoError(t, err, "should not return error")

	assert.NotSame(t, &input, &output, "should return copy of input bytes")
	assert.Exactly(t, inputOriginal, input, "should not modify the source input bytes")

	// Futurer changes to sequences
	node = Node{
		HeadComment: []string{"New comment line 1", "New comment line 2"},
		Key:         "new_seqence",
		ValType:     Sequence,
		Values:      []string{"- key_1: 'value_1'", "val_1: 'value_2'", "- key_2: 'value_1'", "val_2: 'value_2'"},
		EndNewline:  true,
	}
	output, err = Insert(output, "sequences_section", false, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		StartNewline: true,
		HeadComment:  []string{"New comment"},
		Key:          "new_empty_sequence_with_comments",
		ValType:      Sequence,
		Values:       []string{"# - key_1: 'value_1'", "#   val_1: 'value_2'"},
		EndNewline:   true,
	}
	output, err = Insert(output, "sequences_section.sequence", true, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		Key:        "new_sequence_with_comments",
		ValType:    Sequence,
		Values:     []string{"# - key_1: 'value_1'", "#   val_1: 'value_2'", "- key_1: 'value_3'", "val_1: 'value_4'"},
		EndNewline: true,
	}
	output, err = Insert(output, "sequences_section.sequence_with_comments", true, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		HeadComment: []string{"New comment line 1", "New comment line 2"},
		Key:         "new_empty_sequence_with_comments_2",
		ValType:     Sequence,
		Values:      []string{"# - key_1: 'value_1'", "#   val_1: 'value_2'"},
		EndNewline:  true,
	}
	output, err = Insert(output, "sequences_section.empty_sequence_with_comments", true, node)
	assert.NoError(t, err, "should not return error")

	// Futurer changes to lists
	node = Node{
		HeadComment: []string{"New comment"},
		Key:         "new_list",
		ValType:     List,
		Values:      []string{"- 0"},
	}
	output, err = Insert(output, "lists_section", false, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		StartNewline: true,
		Key:          "new_list_with_comments",
		ValType:      List,
		Values:       []string{"- 'item_1'", "- 'item_2'", "# - 'item_3'"},
	}
	output, err = Insert(output, "lists_section.list_with_comments", true, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		StartNewline: true,
		Key:          "new_empty_list_with_comments",
		ValType:      List,
		Values:       []string{"# - 'item_1'"},
		EndNewline:   true,
	}
	output, err = Insert(output, "lists_section.new_list_with_comments", true, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		StartNewline: true,
		Key:          "new_empty_list_with_comments_2",
		ValType:      List,
		Values:       []string{"# - 'item_1'"},
		EndNewline:   true,
	}
	output, err = Insert(output, "lists_section.empty_list_with_comments", true, node)
	assert.NoError(t, err, "should not return error")

	// TODO: Nested lists
	// ...

	// Futurer changes scalars
	node = Node{
		HeadComment: []string{"Comment"},
		Key:         "new_int_item",
		ValType:     Scalar,
		Values:      []string{"1"},
	}
	output, err = Insert(output, "scalar_section.bool_item", false, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		Key:        "new_bool_item",
		ValType:    Scalar,
		Values:     []string{"false"},
		EndNewline: true,
	}
	output, err = Insert(output, "scalar_section.str_item", false, node)
	assert.NoError(t, err, "should not return error")

	// Add new map section to the end
	node = Node{
		HeadComment: []string{"New comment"},
		Key:         "new_map_section",
		ValType:     None,
	}
	output, err = Insert(output, "", false, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		Key:        "new_map",
		ValType:    Map,
		Values:     []string{"# key_1: 'value_1'", "key_2: 'value_2'", "key_3: 'value_3'"},
		EndNewline: true,
	}
	output, err = Insert(output, "new_map_section", false, node)
	assert.NoError(t, err, "should not return error")

	// Add new empty section to the end
	node = Node{
		Key:     "new_empty_section",
		ValType: None,
	}
	output, err = Insert(output, "", false, node)
	assert.NoError(t, err, "should not return error")

	// Compare result with the expected one
	expected, err := os.ReadFile("insert_expected_test.yaml")
	assert.NoError(t, err, "should read expected file")
	assert.Exactly(t, string(expected), string(output), "should produce the following YAML config")
}

func TestSetIndent(t *testing.T) {
	inputBytes, err := os.ReadFile("set_indent_input_test.yaml")
	assert.NoError(t, err, "should read input file")
	input := []rune(string(inputBytes))
	inputOriginal := copier.TestDeep(t, input)

	output := setIndent(input, 2)

	assert.NotSame(t, &input, &output, "should return copy of input")
	assert.Exactly(t, inputOriginal, input, "should not modify the source")

	expected, err := os.ReadFile("set_indent_2_expected_test.yaml")
	assert.NoError(t, err, "should read expected file")
	assert.Exactly(t, string(expected), string(output), "should produce the following YAML config")

	output = setIndent(input, 4)
	expected, err = os.ReadFile("set_indent_4_expected_test.yaml")
	assert.NoError(t, err, "should read expected file")
	assert.Exactly(t, string(expected), string(output), "should produce the following YAML config")
}

func TestInsertIndex(t *testing.T) {
	inputBytes, err := os.ReadFile("insert_input_test.yaml")
	assert.NoError(t, err, "should read input file")
	input := []rune(string(inputBytes))

	// Error cases (path can not be found)
	path := "unknown_root_path"
	index, depth, err := insertIndex(input, path, true, 2)
	assert.ErrorIs(t, PathNotFoundError{Path: path}, err, "should return error for unexisting paths")
	assert.Exactly(t, 0, index, "should return 0 index on error")
	assert.Exactly(t, 0, depth, "should return 0 depth on error")

	path = "nested_section.nested_section_3"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.ErrorIs(t, PathNotFoundError{Path: path}, err, "should return error for not full paths")
	assert.Exactly(t, 0, index, "should return 0 index on error")
	assert.Exactly(t, 0, depth, "should return 0 depth on error")

	path = "sequence"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.ErrorIs(t, PathNotFoundError{Path: path}, err, "should return error for not full paths")
	assert.Exactly(t, 0, index, "should return 0 index on error")
	assert.Exactly(t, 0, depth, "should return 0 depth on error")

	path = "nested_section.nested_section_2.key"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.ErrorIs(t, PathNotFoundError{Path: path}, err, "should return error for path keys with wrong nesting")
	assert.Exactly(t, 0, index, "should return 0 index on error")
	assert.Exactly(t, 0, depth, "should return 0 depth on error")

	path = "nested_section.sequence"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.ErrorIs(t, PathNotFoundError{Path: path}, err, "should return error for path keys with wrong nesting")
	assert.Exactly(t, 0, index, "should return 0 index on error")
	assert.Exactly(t, 0, depth, "should return 0 depth on error")

	path = "sequences_section.lists_section"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.ErrorIs(t, PathNotFoundError{Path: path}, err, "should return error for path keys with wrong nesting")
	assert.Exactly(t, 0, index, "should return 0 index on error")
	assert.Exactly(t, 0, depth, "should return 0 depth on error")

	// Regular behavior
	path = ""
	index, depth, err = insertIndex([]rune{}, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 0, index, "should return 0 index for empty input")
	assert.Exactly(t, 0, depth, "should return 0 depth for empty input")

	index, depth, err = insertIndex([]rune{}, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 0, index, "should return 0 index for empty input")
	assert.Exactly(t, 0, depth, "should return 0 depth for empty input")

	path = ""
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1128, index, "should return last index")
	assert.Exactly(t, 0, depth, "should return 0 depth")
	assert.Exactly(t, "e\"\r\n", string(input[index-4:]), "should be last 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1128, index, "should return last index")
	assert.Exactly(t, 0, depth, "should return 0 depth")
	assert.Exactly(t, "e\"\r\n", string(input[index-4:]), "should be last 4 characters")

	path = "key"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 29, index, "should return that index")
	assert.Exactly(t, 0, depth, "should return 0 depth as key is not a folder")
	assert.Exactly(t, "\r\n\r\nkey_", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 27, index, "should return that index")
	assert.Exactly(t, 0, depth, "should return 0 depth as key is not a folder")
	assert.Exactly(t, "1'\r\n\r\nke", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "nested_section.nested_section_2.nested_section_3.key"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 227, index, "should return that index")
	assert.Exactly(t, 2, depth, "should return that depth")
	assert.Exactly(t, "1'\r\n    ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 227, index, "should return that index")
	assert.Exactly(t, 3, depth, "should return that depth")
	assert.Exactly(t, "1'\r\n    ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "sequences_section.sequence_with_comments"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 629, index, "should return that index")
	assert.Exactly(t, 1, depth, "should return that depth")
	assert.Exactly(t, "\r\n\r\n  # ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 531, index, "should return that index")
	assert.Exactly(t, 2, depth, "should return that depth")
	assert.Exactly(t, "s:\r\n    ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "lists_section"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1069, index, "should return that index")
	assert.Exactly(t, 0, depth, "should return that depth")
	assert.Exactly(t, "5'\r\nscal", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 783, index, "should return that index")
	assert.Exactly(t, 1, depth, "should return that depth")
	assert.Exactly(t, "n:\r\n\r\n  ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "lists_section.empty_list_with_comments"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 963, index, "should return that index")
	assert.Exactly(t, 1, depth, "should return that depth")
	assert.Exactly(t, "1'\r\n  ne", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 945, index, "should return that index")
	assert.Exactly(t, 2, depth, "should return that depth")
	assert.Exactly(t, "s:\r\n    ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "lists_section.nested_list"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1069, index, "should return that index")
	assert.Exactly(t, 1, depth, "should return that depth")
	assert.Exactly(t, "5'\r\nscal", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 979, index, "should return that index")
	assert.Exactly(t, 2, depth, "should return that depth")
	assert.Exactly(t, "t:\r\n    ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "scalar_section"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1128, index, "should return that index")
	assert.Exactly(t, 0, depth, "should return that depth")
	assert.Exactly(t, "e\"\r\n", string(input[index-4:]), "should be last 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1086, index, "should return that index")
	assert.Exactly(t, 1, depth, "should return that depth")
	assert.Exactly(t, "n:\r\n  bo", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "scalar_section.bool_item"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1105, index, "should return that index")
	assert.Exactly(t, 0, depth, "should return that depth")
	assert.Exactly(t, "ue\r\n  st", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1105, index, "should return that index")
	assert.Exactly(t, 1, depth, "should return that depth")
	assert.Exactly(t, "ue\r\n  st", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "scalar_section.str_item"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1128, index, "should return that index")
	assert.Exactly(t, 0, depth, "should return that depth")
	assert.Exactly(t, "e\"\r\n", string(input[index-4:]), "should be last 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1128, index, "should return that index")
	assert.Exactly(t, 1, depth, "should return that depth")
	assert.Exactly(t, "e\"\r\n", string(input[index-4:]), "should be last 4 characters")
}
