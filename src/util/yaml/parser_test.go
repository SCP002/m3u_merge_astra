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
	assert.Exactly(t, PathNotFoundError{Path: afterPath}, errors.UnwrapAll(err), "should return unknown path error")
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
	assert.Exactly(t, PathNotFoundError{Path: afterPath}, errors.UnwrapAll(err), "should return unknown path error")
	assert.Exactly(t, input, output, "on error, output should stay the same as input")

	afterPath = "lists_section.list.item_2"
	node = Node{Data: Scalar{Key: "new_key", Value: "true"}}
	output, err = Insert(input, afterPath, false, node)
	assert.Exactly(t, PathNotFoundError{Path: afterPath}, errors.UnwrapAll(err),
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
							},
							{
								Value: Value{Value: "'item_3'"},
							},
						},
					},
					{
						Value: Value{Value: "'item_4'", Commented: true},
						Children: []ValueTree{
							{
								Value: Value{Value: "'item_5'", Commented: true},
							},
							{
								Value: Value{Value: "'item_6'", Commented: true},
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

	node = Node{
		Data: NestedList{
			Key: "new_nested_list_2",
			Tree: ValueTree{
				Children: []ValueTree{
					{
						Value: Value{Value: "'item_1'"},
						Children: []ValueTree{
							{
								Value: Value{Value: "'item_2'"},
								Children: []ValueTree{
									{
										Value: Value{Value: "'item_3'"},
										Children: []ValueTree{
											{
												Value: Value{Value: "'item_4'"},
											},
										},
									},
								},
							},
						},
					},
					{
						Value: Value{Value: "'item_5'"},
					},
				},
			},
		},
		EndNewline: true,
	}
	output, err = Insert(output, "lists_section.new_nested_list", true, node)
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
	assert.Exactly(t, PathNotFoundError{Path: path}, err, "should return error for unexisting paths")
	assert.Exactly(t, 0, index, "should return 0 index on error")
	assert.Exactly(t, 0, depth, "should return 0 depth on error")

	path = "nested_section.nested_section_3"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.Exactly(t, PathNotFoundError{Path: path}, err, "should return error for not full paths")
	assert.Exactly(t, 0, index, "should return 0 index on error")
	assert.Exactly(t, 0, depth, "should return 0 depth on error")

	path = "sequence"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.Exactly(t, PathNotFoundError{Path: path}, err, "should return error for not full paths")
	assert.Exactly(t, 0, index, "should return 0 index on error")
	assert.Exactly(t, 0, depth, "should return 0 depth on error")

	path = "nested_section.nested_section_2.key"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.Exactly(t, PathNotFoundError{Path: path}, err, "should return error for path keys with wrong nesting")
	assert.Exactly(t, 0, index, "should return 0 index on error")
	assert.Exactly(t, 0, depth, "should return 0 depth on error")

	path = "nested_section.sequence"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.Exactly(t, PathNotFoundError{Path: path}, err, "should return error for path keys with wrong nesting")
	assert.Exactly(t, 0, index, "should return 0 index on error")
	assert.Exactly(t, 0, depth, "should return 0 depth on error")

	path = "sequences_section.lists_section"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.Exactly(t, PathNotFoundError{Path: path}, err, "should return error for path keys with wrong nesting")
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
	assert.Exactly(t, 1192, index, "should return last index")
	assert.Exactly(t, 0, depth, "should return 0 depth")
	assert.Exactly(t, "e 2\n", string(input[index-4:]), "should be last 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1192, index, "should return last index")
	assert.Exactly(t, 0, depth, "should return 0 depth")
	assert.Exactly(t, "e 2\n", string(input[index-4:]), "should be last 4 characters")

	path = "key"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 26, index, "should return that index")
	assert.Exactly(t, 0, depth, "should return 0 depth as key is not a folder")
	assert.Exactly(t, "1'\n\nkey_", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 25, index, "should return that index")
	assert.Exactly(t, 0, depth, "should return 0 depth as key is not a folder")
	assert.Exactly(t, "_1'\n\nkey", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "nested_section.nested_section_2.nested_section_3.key"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 213, index, "should return that index")
	assert.Exactly(t, 2, depth, "should return that depth")
	assert.Exactly(t, "_1'\n    ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 213, index, "should return that index")
	assert.Exactly(t, 3, depth, "should return that depth")
	assert.Exactly(t, "_1'\n    ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "sequences_section.sequence_with_comments"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 593, index, "should return that index")
	assert.Exactly(t, 1, depth, "should return that depth")
	assert.Exactly(t, "'\n\n\n  # ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 501, index, "should return that index")
	assert.Exactly(t, 2, depth, "should return that depth")
	assert.Exactly(t, "ts:\n    ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "lists_section"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1073, index, "should return that index")
	assert.Exactly(t, 0, depth, "should return that depth")
	assert.Exactly(t, "_8'\nscal", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 738, index, "should return that index")
	assert.Exactly(t, 1, depth, "should return that depth")
	assert.Exactly(t, "on:\n\n  l", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "lists_section.empty_list_with_comments"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 906, index, "should return that index")
	assert.Exactly(t, 1, depth, "should return that depth")
	assert.Exactly(t, "_1'\n  ne", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 889, index, "should return that index")
	assert.Exactly(t, 2, depth, "should return that depth")
	assert.Exactly(t, "ts:\n    ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "lists_section.nested_list"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1073, index, "should return that index")
	assert.Exactly(t, 1, depth, "should return that depth")
	assert.Exactly(t, "_8'\nscal", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 921, index, "should return that index")
	assert.Exactly(t, 2, depth, "should return that depth")
	assert.Exactly(t, "st:\n    ", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "scalar_section"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1130, index, "should return that index")
	assert.Exactly(t, 0, depth, "should return that depth")
	assert.Exactly(t, "e\"\n\n# Co", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1089, index, "should return that index")
	assert.Exactly(t, 1, depth, "should return that depth")
	assert.Exactly(t, "on:\n  bo", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "scalar_section.bool_item"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1107, index, "should return that index")
	assert.Exactly(t, 0, depth, "should return that depth")
	assert.Exactly(t, "rue\n  st", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1107, index, "should return that index")
	assert.Exactly(t, 1, depth, "should return that depth")
	assert.Exactly(t, "rue\n  st", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	path = "scalar_section.str_item"
	index, depth, err = insertIndex(input, path, true, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1130, index, "should return that index")
	assert.Exactly(t, 0, depth, "should return that depth")
	assert.Exactly(t, "e\"\n\n# Co", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")

	index, depth, err = insertIndex(input, path, false, 2)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, 1129, index, "should return that index")
	assert.Exactly(t, 1, depth, "should return that depth")
	assert.Exactly(t, "ue\"\n\n# C", string(input[index-4:index+4]), "should be from last 4 to next 4 characters")
}

func TestFlatten(t *testing.T) {
	tree := ValueTree{
		Children: []ValueTree{
			{
				Value: Value{Value: "'item_1'"},
				Children: []ValueTree{
					{Value: Value{Value: "'item_2'"}},
				},
			},
			{
				Value: Value{Value: "'item_3'"},
				Children: []ValueTree{
					{Value: Value{Value: "'item_4'"}},
				},
			},
			{
				Value: Value{Value: "'item_5'"},
				Children: []ValueTree{
					{Value: Value{Value: "'item_6'"}},
				},
			},
		},
	}
	treeOriginal := copier.TestDeep(t, tree)

	maxDepth := 0
	actual := flatten(tree, &maxDepth)

	assert.Exactly(t, treeOriginal, tree, "should not modify the source tree")

	expected := []ValueTree{
		{
			Value: Value{Value: "'item_1'"},
			depth: 1,
			Children: []ValueTree{
				{
					Value: Value{Value: "'item_2'"},
					depth: 0,
				},
			},
		},
		{
			Value: Value{Value: "'item_2'"},
			depth: 2,
		},
		{
			Value: Value{Value: "'item_3'"},
			depth: 1,
			Children: []ValueTree{
				{
					Value: Value{Value: "'item_4'"},
					depth: 0,
				},
			},
		},
		{
			Value: Value{Value: "'item_4'"},
			depth: 2,
		},
		{
			Value: Value{Value: "'item_5'"},
			depth: 1,
			Children: []ValueTree{
				{
					Value: Value{Value: "'item_6'"},
					depth: 0,
				},
			},
		},
		{
			Value: Value{Value: "'item_6'"},
			depth: 2,
		},
	}
	assert.Exactly(t, expected, actual, "should return that flat tree")
	assert.Exactly(t, 2, maxDepth, "should return that maximum depth")

	tree = ValueTree{
		Children: []ValueTree{
			{
				Value: Value{Value: "'item_1'"},
				Children: []ValueTree{
					{
						Value: Value{Value: "'item_2'"},
						Children: []ValueTree{
							{
								Value: Value{Value: "'item_3'"},
								Children: []ValueTree{
									{
										Value: Value{Value: "'item_4'"},
									},
								},
							},
						},
					},
				},
			},
			{
				Value: Value{Value: "'item_5'"},
			},
		},
	}
	treeOriginal = copier.TestDeep(t, tree)

	actual = flatten(tree, &maxDepth)

	assert.Exactly(t, treeOriginal, tree, "should not modify the source tree")

	expected = []ValueTree{
		{
			Value: Value{Value: "'item_1'"},
			depth: 1,
			Children: []ValueTree{
				{
					Value: Value{Value: "'item_2'"},
					depth: 0,
					Children: []ValueTree{
						{
							Value: Value{Value: "'item_3'"},
							depth: 0,
							Children: []ValueTree{
								{
									Value: Value{Value: "'item_4'"},
									depth: 0,
								},
							},
						},
					},
				},
			},
		},
		{
			Value: Value{Value: "'item_2'"},
			depth: 2,
			Children: []ValueTree{
				{
					Value: Value{Value: "'item_3'"},
					depth: 0,
					Children: []ValueTree{
						{
							Value: Value{Value: "'item_4'"},
							depth: 0,
						},
					},
				},
			},
		},
		{
			Value: Value{Value: "'item_3'"},
			depth: 3,
			Children: []ValueTree{
				{
					Value: Value{Value: "'item_4'"},
					depth: 0,
				},
			},
		},
		{
			Value: Value{Value: "'item_4'"},
			depth: 4,
		},
		{
			Value: Value{Value: "'item_5'"},
			depth: 1,
		},
	}
	assert.Exactly(t, expected, actual, "should return that flat tree")
	assert.Exactly(t, 4, maxDepth, "should return that maximum depth")
}
