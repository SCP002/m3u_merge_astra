package yaml

import (
	"m3u_merge_astra/util/copier"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInsertIndex(t *testing.T) {
	inputBytes, err := os.ReadFile("input_test.yaml")
	assert.NoError(t, err, "should read input file")
	input := []rune(string(inputBytes))

	// Error cases (path can not be found)
	path := "unknown_root_path:"
	index, err := insertIndex(input, path)
	assert.ErrorIs(t, PathNotFoundError{Path: path}, err, "should return error for unexisting paths")
	assert.Exactly(t, 0, index, "should return 0 index on error")

	path = "nested_section.nested_section_3:"
	index, err = insertIndex(input, path)
	assert.ErrorIs(t, PathNotFoundError{Path: path}, err, "should return error for not full paths")
	assert.Exactly(t, 0, index, "should return 0 index on error")

	path = "sequence:"
	index, err = insertIndex(input, path)
	assert.ErrorIs(t, PathNotFoundError{Path: path}, err, "should return error for not full paths")
	assert.Exactly(t, 0, index, "should return 0 index on error")

	path = "sequences_section.lists_section:"
	index, err = insertIndex(input, path)
	assert.ErrorIs(t, PathNotFoundError{Path: path}, err, "should return error for path keys with wrong nesting")
	assert.Exactly(t, 0, index, "should return 0 index on error")

	// Regular behavior
	path = ""
	index, err = insertIndex(input, path)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 935, index, "should return last index")
	assert.Exactly(t, "e\"\r\n", string(input[index-4:]), "should be last 4 characters")

	path = "key:"
	index, err = insertIndex(input, path)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 29, index, "should return that index")
	assert.Exactly(t, "\r\n\r\nkey_", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "sequences_section.sequence_with_comments:"
	index, err = insertIndex(input, path)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 536, index, "should return that index")
	assert.Exactly(t, "\r\n\r\n  # ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "lists_section:"
	index, err = insertIndex(input, path)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 872, index, "should return that index")
	assert.Exactly(t, "1'\r\nscal", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "lists_section.empty_list_with_comments:"
	index, err = insertIndex(input, path)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 872, index, "should return that index")
	assert.Exactly(t, "1'\r\nscal", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "scalar_section:"
	index, err = insertIndex(input, path)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 935, index, "should return that index")
	assert.Exactly(t, "e\"\r\n", string(input[index-4:]), "should be last 4 characters")

	path = "scalar_section.str_item:"
	index, err = insertIndex(input, path)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 935, index, "should return that index")
	assert.Exactly(t, "e\"\r\n", string(input[index-4:]), "should be last 4 characters")
}

func TestInsert(t *testing.T) {
	input, err := os.ReadFile("input_test.yaml")
	assert.NoError(t, err, "should read input file")
	inputOriginal := copier.TDeep(t, input)

	// Error cases (path can not be found)
	afterPath := "unknown_root_path:"
	output, err := Insert(input, afterPath, []string{}, "new_key", Scalar, "true")
	assert.ErrorIs(t, PathNotFoundError{Path: afterPath}, err, "should return unknown path error")
	assert.Exactly(t, input, output, "on error, output should stay the same as input")

	afterPath = "sequences_section.unknown_subkey:"
	output, err = Insert(input, afterPath, []string{}, "new_key", Sequence, []string{"- a: 'b'", "c: 'd'"}...)
	assert.ErrorIs(t, PathNotFoundError{Path: afterPath}, err, "should return unknown path error")
	assert.Exactly(t, input, output, "on error, output should stay the same as input")

	afterPath = "lists_section.list.item_2:"
	output, err = Insert(input, afterPath, []string{}, "new_key", Scalar, "true")
	assert.ErrorIs(t, PathNotFoundError{Path: afterPath}, err, "should not resolve node value as proper path")
	assert.Exactly(t, input, output, "on error, output should stay the same as input")

	// Regular behavior
	// First change
	output, err = Insert(input, "key:", []string{"New comment"}, "new_key", Scalar, "'value_1'")
	assert.NoError(t, err, "should not return error")

	assert.NotSame(t, &input, &output, "should return copy of input bytes")
	assert.Exactly(t, inputOriginal, input, "should not modify the source input bytes")

	// Futurer changes to sequences
	output, err = Insert(
		output,
		"sequences_section:",
		[]string{"New comment line 1", "New comment line 2"},
		"new_seqence",
		Sequence,
		[]string{"- key_1: 'value_1'", "val_1: 'value_2'", "- key_2: 'value_1'", "val_2: 'value_2'"}...,
	)
	assert.NoError(t, err, "should not return error")

	output, err = Insert(
		output,
		"sequences_section.sequence:",
		[]string{"New comment"},
		"new_empty_sequence_with_comments",
		Sequence,
		[]string{"# - key_1: 'value_1'", "#   val_1: 'value_2'"}...,
	)
	assert.NoError(t, err, "should not return error")

	output, err = Insert(
		output,
		"sequences_section.sequence_with_comments:",
		[]string{},
		"new_sequence_with_comments",
		Sequence,
		[]string{"# - key_1: 'value_1'", "#   val_1: 'value_2'", "- key_1: 'value_3'", "val_1: 'value_4'"}...,
	)
	assert.NoError(t, err, "should not return error")

	output, err = Insert(
		output,
		"sequences_section.empty_sequence_with_comments:",
		[]string{"New comment line 1", "New comment line 2"},
		"new_empty_sequence_with_comments_2",
		Sequence,
		[]string{"# - key_1: 'value_1'", "#   val_1: 'value_2'"}...,
	)
	assert.NoError(t, err, "should not return error")

	// Futurer changes to lists
	output, err = Insert(
		output,
		"lists_section:",
		[]string{"New comment"},
		"new_list",
		List,
		[]string{"- 0"}...,
	)
	assert.NoError(t, err, "should not return error")

	output, err = Insert(
		output,
		"lists_section.list_with_comments:",
		[]string{},
		"new_list_with_comments",
		List,
		[]string{"- 'item_1'", "- 'item_2'", "# - 'item_3'"}...,
	)
	assert.NoError(t, err, "should not return error")

	output, err = Insert(
		output,
		"lists_section.new_list_with_comments:",
		[]string{},
		"new_empty_list_with_comments",
		List,
		[]string{"# - 'item_1'"}...,
	)
	assert.NoError(t, err, "should not return error")

	output, err = Insert(
		output,
		"lists_section.empty_list_with_comments:",
		[]string{},
		"new_empty_list_with_comments_2",
		List,
		[]string{"# - 'item_1'"}...,
	)
	assert.NoError(t, err, "should not return error")

	// Futurer changes scalars
	output, err = Insert(
		output,
		"scalar_section.bool_item:",
		[]string{"Comment"},
		"new_int_item",
		Scalar,
		[]string{"1"}...,
	)
	assert.NoError(t, err, "should not return error")

	output, err = Insert(
		output,
		"scalar_section.str_item:",
		[]string{},
		"new_bool_item",
		Scalar,
		[]string{"false"}...,
	)
	assert.NoError(t, err, "should not return error")

	// Add new map section to the end
	output, err = Insert(
		output,
		"",
		[]string{"New comment"},
		"new_map_section",
		Scalar,
		[]string{}...,
	)
	assert.NoError(t, err, "should not return error")

	output, err = Insert(
		output,
		"new_map_section:",
		[]string{},
		"new_map",
		Map,
		[]string{"# key_1: 'value_1'", "key_2: 'value_2'", "key_3: 'value_3'"}...,
	)
	assert.NoError(t, err, "should not return error")

	// Add new empty section to the end
	output, err = Insert(
		output,
		"",
		[]string{},
		"new_empty_section",
		Scalar,
		[]string{}...,
	)
	assert.NoError(t, err, "should not return error")

	// Compare result with the expected one
	expected, err := os.ReadFile("expected_test.yaml")
	assert.NoError(t, err, "should read expected file")
	assert.Exactly(t, string(expected), string(output), "should produce the following YAML config")
}
