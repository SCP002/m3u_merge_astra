package yaml

import (
	"m3u_merge_astra/util/copier"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInsert(t *testing.T) {
	input, err := os.ReadFile("insert_input_test.yaml")
	assert.NoError(t, err, "should read input file")
	inputOriginal := copier.TDeep(t, input)

	// Error cases (path can not be found)
	afterPath := "unknown_root_path:"
	node := Node{HeadComment: []string{}, Key: "new_key", ValType: Scalar, Values: []string{"true"}}
	output, err := Insert(input, afterPath, false, node)
	assert.ErrorIs(t, PathNotFoundError{Path: afterPath}, err, "should return unknown path error")
	assert.Exactly(t, input, output, "on error, output should stay the same as input")

	afterPath = "sequences_section.unknown_subkey:"
	node = Node{HeadComment: []string{}, Key: "new_key", ValType: Sequence, Values: []string{"- a: 'b'", "c: 'd'"}}
	output, err = Insert(input, afterPath, false, node)
	assert.ErrorIs(t, PathNotFoundError{Path: afterPath}, err, "should return unknown path error")
	assert.Exactly(t, input, output, "on error, output should stay the same as input")

	afterPath = "lists_section.list.item_2:"
	node = Node{HeadComment: []string{}, Key: "new_key", ValType: Scalar, Values: []string{"true"}}
	output, err = Insert(input, afterPath, false, node)
	assert.ErrorIs(t, PathNotFoundError{Path: afterPath}, err, "should not resolve node value as proper path")
	assert.Exactly(t, input, output, "on error, output should stay the same as input")

	// Regular behavior
	// First change
	node = Node{HeadComment: []string{"New comment"}, Key: "new_key", ValType: Scalar, Values: []string{"'value_1'"}}
	output, err = Insert(input, "key:", false, node)
	assert.NoError(t, err, "should not return error")

	assert.NotSame(t, &input, &output, "should return copy of input bytes")
	assert.Exactly(t, inputOriginal, input, "should not modify the source input bytes")

	// Futurer changes to sequences
	node = Node{
		HeadComment: []string{"New comment line 1", "New comment line 2"},
		Key:         "new_seqence",
		ValType:     Sequence,
		Values:      []string{"- key_1: 'value_1'", "val_1: 'value_2'", "- key_2: 'value_1'", "val_2: 'value_2'"},
	}
	output, err = Insert(output, "sequences_section:", false, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		HeadComment: []string{"New comment"},
		Key:         "new_empty_sequence_with_comments",
		ValType:     Sequence,
		Values:      []string{"# - key_1: 'value_1'", "#   val_1: 'value_2'"},
	}
	output, err = Insert(output, "sequences_section.sequence:", true, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		HeadComment: []string{},
		Key:         "new_sequence_with_comments",
		ValType:     Sequence,
		Values:      []string{"# - key_1: 'value_1'", "#   val_1: 'value_2'", "- key_1: 'value_3'", "val_1: 'value_4'"},
	}
	output, err = Insert(output, "sequences_section.sequence_with_comments:", true, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		HeadComment: []string{"New comment line 1", "New comment line 2"},
		Key:         "new_empty_sequence_with_comments_2",
		ValType:     Sequence,
		Values:      []string{"# - key_1: 'value_1'", "#   val_1: 'value_2'"},
	}
	output, err = Insert(output, "sequences_section.empty_sequence_with_comments:", true, node)
	assert.NoError(t, err, "should not return error")

	// Futurer changes to lists
	node = Node{
		HeadComment: []string{"New comment"},
		Key:         "new_list",
		ValType:     List,
		Values:      []string{"- 0"},
	}
	output, err = Insert(output, "lists_section:", true, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		HeadComment: []string{},
		Key:         "new_list_with_comments",
		ValType:     List,
		Values:      []string{"- 'item_1'", "- 'item_2'", "# - 'item_3'"},
	}
	output, err = Insert(output, "lists_section.list_with_comments:", true, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		HeadComment: []string{},
		Key:         "new_empty_list_with_comments",
		ValType:     List,
		Values:      []string{"# - 'item_1'"},
	}
	output, err = Insert(output, "lists_section.new_list_with_comments:", true, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		HeadComment: []string{},
		Key:         "new_empty_list_with_comments_2",
		ValType:     List,
		Values:      []string{"# - 'item_1'"},
	}
	output, err = Insert(output, "lists_section.empty_list_with_comments:", true, node)
	assert.NoError(t, err, "should not return error")

	// Futurer changes scalars
	node = Node{
		HeadComment: []string{"Comment"},
		Key:         "new_int_item",
		ValType:     Scalar,
		Values:      []string{"1"},
	}
	output, err = Insert(output, "scalar_section.bool_item:", false, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		HeadComment: []string{},
		Key:         "new_bool_item",
		ValType:     Scalar,
		Values:      []string{"false"},
	}
	output, err = Insert(output, "scalar_section.str_item:", false, node)
	assert.NoError(t, err, "should not return error")

	// Add new map section to the end
	node = Node{
		HeadComment: []string{"New comment"},
		Key:         "new_map_section",
		ValType:     Scalar,
		Values:      []string{},
	}
	output, err = Insert(output, "", false, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		HeadComment: []string{},
		Key:         "new_map",
		ValType:     Map,
		Values:      []string{"# key_1: 'value_1'", "key_2: 'value_2'", "key_3: 'value_3'"},
	}
	output, err = Insert(output, "new_map_section:", false, node)
	assert.NoError(t, err, "should not return error")

	// Add new empty section to the end
	node = Node{
		HeadComment: []string{},
		Key:         "new_empty_section",
		ValType:     Scalar,
		Values:      []string{},
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
	inputOriginal := copier.TDeep(t, input)

	output := setIndent(input, 2)

	assert.NotSame(t, &input, &output, "should return copy of input")
	assert.Exactly(t, inputOriginal, input, "should not modify the source")

	expected, err := os.ReadFile("set_indent_expected_test.yaml")
	assert.NoError(t, err, "should read expected file")
	assert.Exactly(t, string(expected), string(output), "should produce the following YAML config")
}

func TestInsertIndex(t *testing.T) {
	inputBytes, err := os.ReadFile("insert_input_test.yaml")
	assert.NoError(t, err, "should read input file")
	input := []rune(string(inputBytes))

	// Error cases (path can not be found)
	path := "unknown_root_path:"
	index, err := insertIndex(input, path, true, 2)
	assert.ErrorIs(t, PathNotFoundError{Path: path}, err, "should return error for unexisting paths")
	assert.Exactly(t, 0, index, "should return 0 index on error")

	path = "nested_section.nested_section_3:"
	index, err = insertIndex(input, path, true, 2)
	assert.ErrorIs(t, PathNotFoundError{Path: path}, err, "should return error for not full paths")
	assert.Exactly(t, 0, index, "should return 0 index on error")

	path = "sequence:"
	index, err = insertIndex(input, path, true, 2)
	assert.ErrorIs(t, PathNotFoundError{Path: path}, err, "should return error for not full paths")
	assert.Exactly(t, 0, index, "should return 0 index on error")

	path = "nested_section.nested_section_2.key:"
	index, err = insertIndex(input, path, true, 2)
	assert.ErrorIs(t, PathNotFoundError{Path: path}, err, "should return error for path keys with wrong nesting")
	assert.Exactly(t, 0, index, "should return 0 index on error")

	path = "nested_section.sequence:"
	index, err = insertIndex(input, path, true, 2)
	assert.ErrorIs(t, PathNotFoundError{Path: path}, err, "should return error for path keys with wrong nesting")
	assert.Exactly(t, 0, index, "should return 0 index on error")

	path = "sequences_section.lists_section:"
	index, err = insertIndex(input, path, true, 2)
	assert.ErrorIs(t, PathNotFoundError{Path: path}, err, "should return error for path keys with wrong nesting")
	assert.Exactly(t, 0, index, "should return 0 index on error")

	// Regular behavior
	path = ""
	index, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1016, index, "should return last index")
	assert.Exactly(t, "e\"\r\n", string(input[index-4:]), "should be last 4 characters")

	index, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1016, index, "should return last index")
	assert.Exactly(t, "e\"\r\n", string(input[index-4:]), "should be last 4 characters")

	path = "key:"
	index, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 29, index, "should return that index")
	assert.Exactly(t, "\r\n\r\nkey_", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 27, index, "should return that index")
	assert.Exactly(t, "1'\r\n\r\nke", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "nested_section.nested_section_2.nested_section_3.key:"
	index, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 227, index, "should return that index")
	assert.Exactly(t, "1'\r\n    ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 227, index, "should return that index")
	assert.Exactly(t, "1'\r\n    ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "sequences_section.sequence_with_comments:"
	index, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 623, index, "should return that index")
	assert.Exactly(t, "\r\n\r\n  # ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 525, index, "should return that index")
	assert.Exactly(t, "s:\r\n    ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "lists_section:"
	index, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 957, index, "should return that index")
	assert.Exactly(t, "1'\r\nscal", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 777, index, "should return that index")
	assert.Exactly(t, "n:\r\n\r\n  ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "lists_section.empty_list_with_comments:"
	index, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 957, index, "should return that index")
	assert.Exactly(t, "1'\r\nscal", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 939, index, "should return that index")
	assert.Exactly(t, "s:\r\n    ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "scalar_section:"
	index, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1016, index, "should return that index")
	assert.Exactly(t, "e\"\r\n", string(input[index-4:]), "should be last 4 characters")

	index, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 974, index, "should return that index")
	assert.Exactly(t, "n:\r\n  bo", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "scalar_section.bool_item:"
	index, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 993, index, "should return that index")
	assert.Exactly(t, "ue\r\n  st", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 993, index, "should return that index")
	assert.Exactly(t, "ue\r\n  st", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "scalar_section.str_item:"
	index, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1016, index, "should return that index")
	assert.Exactly(t, "e\"\r\n", string(input[index-4:]), "should be last 4 characters")

	index, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1016, index, "should return that index")
	assert.Exactly(t, "e\"\r\n", string(input[index-4:]), "should be last 4 characters")
}
