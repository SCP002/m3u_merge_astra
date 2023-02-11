package yaml

// Pair represents key and value pair with comment boolean flag
type Pair struct {
	Key       string
	Value     string
	Commented bool
}

// Value represents value with comment boolean flag
type Value struct {
	Value     string
	Commented bool
}

// ValueTree represents tree of values with children
type ValueTree struct {
	Values   []Value
	Children *ValueTree
}

// Key represents YAML node value type. Use to create keys without values.
type Key struct {
	Key       string
	Commented bool
}

// Scalar represents YAML node value type
type Scalar struct {
	Key       string
	Value     string
	Commented bool
}

// Sequence represents YAML node value type
type Sequence struct {
	Key  string
	Sets [][]Pair
}

// List represents YAML node value type
type List struct {
	Key    string
	Values []Value
}

// NestedList represents YAML node value type
type NestedList struct {
	Key  string
	Tree ValueTree
}

// Map represents YAML node value type
type Map struct {
	Key string
	Map map[string]Value // Keep it as map type to prevent key duplication
}
