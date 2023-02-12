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

func TestBadDataError(t *testing.T) {
	err := error(BadDataError{Reason: "unknown"})
	assert.Exactly(t, "unknown", err.Error())
}

func TestInsert(t *testing.T) {
	input, err := os.ReadFile("insert_input_test.yaml")
	assert.NoError(t, err, "should read input file")
	inputOriginal := copier.TestDeep(t, input)

	// Error cases (bad data or path can not be found)
	afterPath := ""
	node := Node{Data: Scalar{Value: "a"}}
	output, err := Insert(input, afterPath, false, node)
	assert.ErrorAs(t, err, &BadDataError{}, "should return bad data error if key is not specified")
	assert.Exactly(t, input, output, "on error, output should stay the same as input")

	afterPath = "key"
	node = Node{Data: Sequence{
		Key: "new_key",
		Sets: [][]Pair{
			{
				{Key: "key_1", Value: "'val_1'"},
				{Key: "key_2", Commented: true}, // <- Missing value here
			},
		},
	}}
	output, err = Insert(input, afterPath, false, node)
	assert.ErrorAs(t, err, &BadDataError{}, "should return bad data error if value is not specified")
	assert.Exactly(t, input, output, "on error, output should stay the same as input")

	afterPath = "nested_section"
	node = Node{Data: errors.EncodedError{}} // Random invalid struct
	output, err = Insert(input, afterPath, false, node)
	assert.ErrorAs(t, err, &BadDataError{}, "should return bad data error if data type is invalid")
	assert.Exactly(t, input, output, "on error, output should stay the same as input")

	afterPath = "unknown_root_path"
	node = Node{Data: Scalar{Key: "new_key", Value: "true"}}
	output, err = Insert(input, afterPath, false, node)
	assert.ErrorIs(t, PathNotFoundError{Path: afterPath}, errors.UnwrapAll(err), "should return unknown path error")
	assert.Exactly(t, input, output, "on error, output should stay the same as input")

	afterPath = "sequences_section.unknown_subkey"
	node = Node{Data: Sequence{
		Key: "new_key",
		Sets: [][]Pair{
			{
				{Key: "a", Value: "'b'"},
				{Key: "c", Value: "'d'"},
			},
		},
	}}
	output, err = Insert(input, afterPath, false, node)
	assert.ErrorIs(t, PathNotFoundError{Path: afterPath}, errors.UnwrapAll(err), "should return unknown path error")
	assert.Exactly(t, input, output, "on error, output should stay the same as input")

	afterPath = "lists_section.list.item_2"
	node = Node{Data: Scalar{Key: "new_key", Value: "true"}}
	output, err = Insert(input, afterPath, false, node)
	assert.ErrorIs(t, PathNotFoundError{Path: afterPath}, errors.UnwrapAll(err),
		"should not resolve node value as proper path")
	assert.Exactly(t, input, output, "on error, output should stay the same as input")

	// Overflow check, should not panic
	node = Node{Data: List{Key: "new_key", Values: slice.Filled(Value{Value: "a"}, 10000)}}
	_, err = Insert(input, "", false, node)
	assert.NoError(t, err, "should not return error")

	// Regular behavior
	// First change
	node = Node{
		StartNewline: true,
		HeadComment:  []string{"New comment"},
		Data:         Scalar{Key: "new_key", Value: "'value_1'"},
	}
	output, err = Insert(input, "key", false, node)
	assert.NoError(t, err, "should not return error")

	assert.NotSame(t, &input, &output, "should return copy of input bytes")
	assert.Exactly(t, inputOriginal, input, "should not modify the source input bytes")

	// Futurer changes to sequences
	node = Node{
		HeadComment: []string{"New comment line 1", "New comment line 2"},
		Data: Sequence{
			Key: "new_sequence",
			Sets: [][]Pair{
				{
					{Key: "key_1", Value: "'value_1'"},
					{Key: "val_1", Value: "'value_2'"},
				},
				{
					{Key: "key_2", Value: "'value_1'"},
					{Key: "val_2", Value: "'value_2'"},
				},
			},
		},
		EndNewline: true,
	}
	output, err = Insert(output, "sequences_section", false, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		StartNewline: true,
		HeadComment:  []string{"New comment"},
		Data: Sequence{
			Key: "new_empty_sequence_with_comments",
			Sets: [][]Pair{
				{
					{Key: "key_1", Value: "'value_1'", Commented: true},
					{Key: "val_1", Value: "'value_2'", Commented: true},
				},
			},
		},
		EndNewline: true,
	}
	output, err = Insert(output, "sequences_section.sequence", true, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		Data: Sequence{
			Key: "new_sequence_with_comments",
			Sets: [][]Pair{
				{
					{Key: "key_1", Value: "'value_1'", Commented: true},
					{Key: "val_1", Value: "'value_2'", Commented: true},
				},
				{
					{Key: "key_1", Value: "'value_3'"},
					{Key: "val_1", Value: "'value_4'"},
				},
			},
		},
		EndNewline: true,
	}
	output, err = Insert(output, "sequences_section.sequence_with_comments", true, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		HeadComment: []string{"New comment line 1", "New comment line 2"},
		Data: Sequence{
			Key: "new_empty_sequence_with_comments_2",
			Sets: [][]Pair{
				{
					{Key: "key_1", Value: "'value_1'", Commented: true},
					{Key: "val_1", Value: "'value_2'", Commented: true},
				},
			},
		},
		EndNewline: true,
	}
	output, err = Insert(output, "sequences_section.empty_sequence_with_comments", true, node)
	assert.NoError(t, err, "should not return error")

	// Futurer changes to lists
	node = Node{
		HeadComment: []string{"New comment"},
		Data: List{
			Key: "new_list",
			Values: []Value{
				{Value: "0"},
			},
		},
	}
	output, err = Insert(output, "lists_section", false, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		StartNewline: true,
		Data: List{
			Key: "new_list_with_comments",
			Values: []Value{
				{Value: "'item_1'"}, {Value: "'item_2'"}, {Value: "'item_3'", Commented: true},
			},
		},
	}
	output, err = Insert(output, "lists_section.list_with_comments", true, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		StartNewline: true,
		Data: List{
			Key: "new_empty_list_with_comments",
			Values: []Value{
				{Value: "'item_1'", Commented: true},
			},
		},
		EndNewline: true,
	}
	output, err = Insert(output, "lists_section.new_list_with_comments", true, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		StartNewline: true,
		Data: List{
			Key: "new_empty_list_with_comments_2",
			Values: []Value{
				{Value: "'item_1'", Commented: true},
			},
		},
		EndNewline: true,
	}
	output, err = Insert(output, "lists_section.empty_list_with_comments", true, node)
	assert.NoError(t, err, "should not return error")

	// Futurer changes to nested lists
	node = Node{
		StartNewline: true,
		Data: NestedList{
			Key: "new_nested_list",
			Tree: ValueTree{
				Children: []ValueTree{
					{
						Value: Value{Value: "'item_1'"},
						Children: []ValueTree{
							{
								Value: Value{Value: "'item_2'"},
								Children: []ValueTree{
									{Value: Value{Value: "'item_3'"}},
									{Value: Value{Value: "'item_4'"}},
								},
							},
						},
					},
					{
						Value: Value{Value: "'item_5'"},
						Children: []ValueTree{
							{
								Value: Value{Value: "'item_6'"},
								Children: []ValueTree{
									{Value: Value{Value: "'item_7'"}},
									{Value: Value{Value: "'item_8'", Commented: true}},
								},
							},
						},
					},
				},
			},
		},
		EndNewline: true,
	}
	output, err = Insert(output, "lists_section.nested_list", true, node)
	assert.NoError(t, err, "should not return error")

	// Futurer changes scalars
	node = Node{
		HeadComment: []string{"Comment"},
		Data:        Scalar{Key: "new_int_item", Value: "1"},
	}
	output, err = Insert(output, "scalar_section.bool_item", false, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		Data: Scalar{Key: "new_bool_item", Value: "false"},
	}
	output, err = Insert(output, "scalar_section.str_item", false, node)
	assert.NoError(t, err, "should not return error")

	// Add new map section to the end
	node = Node{
		StartNewline: true,
		HeadComment:  []string{"New comment"},
		Data:         Key{Key: "new_map_section"},
	}
	output, err = Insert(output, "", false, node)
	assert.NoError(t, err, "should not return error")

	node = Node{
		Data: Map{
			Key: "new_map",
			Map: map[string]Value{
				"key_1": {Value: "'value_1'", Commented: true},
				"key_2": {Value: "'value_2'"},
				"key_3": {Value: "'value_3'"},
			},
		},
		EndNewline: true,
	}
	output, err = Insert(output, "new_map_section", false, node)
	assert.NoError(t, err, "should not return error")

	// Add new empty section to the end
	node = Node{
		Data: Key{Key: "new_empty_section"},
	}
	output, err = Insert(output, "", false, node)
	assert.NoError(t, err, "should not return error")

	// Add comment with no data
	node = Node{
		StartNewline: true,
		HeadComment:  []string{"New comment without data, line 1", "New comment without data, line 2"},
		Data:         nil,
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
	assert.Exactly(t, 1264, index, "should return last index")
	assert.Exactly(t, 0, depth, "should return 0 depth")
	assert.Exactly(t, " 2\r\n", string(input[index-4:]), "should be last 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1264, index, "should return last index")
	assert.Exactly(t, 0, depth, "should return 0 depth")
	assert.Exactly(t, " 2\r\n", string(input[index-4:]), "should be last 4 characters")

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
	assert.Exactly(t, 1139, index, "should return that index")
	assert.Exactly(t, 0, depth, "should return that depth")
	assert.Exactly(t, "8'\r\nscal", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

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
	assert.Exactly(t, 1139, index, "should return that index")
	assert.Exactly(t, 1, depth, "should return that depth")
	assert.Exactly(t, "8'\r\nscal", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 979, index, "should return that index")
	assert.Exactly(t, 2, depth, "should return that depth")
	assert.Exactly(t, "t:\r\n    ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "scalar_section"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1200, index, "should return that index")
	assert.Exactly(t, 0, depth, "should return that depth")
	assert.Exactly(t, "\r\n\r\n# Co", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1156, index, "should return that index")
	assert.Exactly(t, 1, depth, "should return that depth")
	assert.Exactly(t, "n:\r\n  bo", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "scalar_section.bool_item"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1175, index, "should return that index")
	assert.Exactly(t, 0, depth, "should return that depth")
	assert.Exactly(t, "ue\r\n  st", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1175, index, "should return that index")
	assert.Exactly(t, 1, depth, "should return that depth")
	assert.Exactly(t, "ue\r\n  st", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "scalar_section.str_item"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1200, index, "should return that index")
	assert.Exactly(t, 0, depth, "should return that depth")
	assert.Exactly(t, "\r\n\r\n# Co", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1198, index, "should return that index")
	assert.Exactly(t, 1, depth, "should return that depth")
	assert.Exactly(t, "e\"\r\n\r\n# ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")
}

func TestFlatten(t *testing.T) { // TODO: Make it pass
	tree := ValueTree{
		Children: []ValueTree{
			{
				Value: Value{Value: "'item_1'"},
				Children: []ValueTree{
					{Value: Value{Value: "'item_2'"}},
					{Value: Value{Value: "'item_3'"},
						Children: []ValueTree{
							{Value: Value{Value: "'item_4'"}},
							{Value: Value{Value: "'item_5'"}},
						},
					},
				},
			},
			{
				Value: Value{Value: "'item_6'"},
				Children: []ValueTree{
					{Value: Value{Value: "'item_7'"}},
				},
			},
		},
	}
	treeOriginal := copier.TestDeep(t, tree)

	actual, maxDepth := flatten(tree)

	assert.Exactly(t, treeOriginal, tree, "should not modify the source tree")

	expected := []ValueTree{
		{ // 1
			Value: Value{Value: "'item_1'"},
			Depth: 1,
			Children: []ValueTree{
				{
					Value: Value{Value: "'item_2'"},
					Depth: 2,
				},
				{
					Value: Value{Value: "'item_3'"},
					Depth: 2,
					Children: []ValueTree{
						{
							Value: Value{Value: "'item_4'"},
							Depth: 3,
						},
						{
							Value: Value{Value: "'item_5'"},
							Depth: 3,
						},
					},
				},
			},
		},
		{ // 2
			Value: Value{Value: "'item_2'"},
			Depth: 2,
		},
		{ // 3
			Value: Value{Value: "'item_3'"},
			Depth: 2,
			Children: []ValueTree{
				{
					Value: Value{Value: "'item_4'"},
					Depth: 3,
				},
				{
					Value: Value{Value: "'item_5'"},
					Depth: 3,
				},
			},
		},
		{ // 4
			Value: Value{Value: "'item_4'"},
			Depth: 0,
		},
		{ // 5
			Value: Value{Value: "'item_5'"},
			Depth: 0,
		},
		{ // 6
			Value: Value{Value: "'item_6'"},
			Depth: 1,
			Children: []ValueTree{
				{
					Value: Value{Value: "'item_7'"},
					Depth: 2,
				},
			},
		},
		{ // 7
			Value: Value{Value: "'item_7'"},
			Depth: 2,
		},
	}
	assert.Exactly(t, expected, actual, "should return that flat tree")
	assert.Exactly(t, 3, maxDepth, "should return that maximum depth")
}
