package slice

import (
	"m3u_merge_astra/util/copier"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPrepend(t *testing.T) {
	assert.Exactly(t, []int{0}, Prepend([]int{}, 0), "should add 0")
	assert.Exactly(t, []int{0, 1}, Prepend([]int{1}, 0), "should add 0 to the beginning")
}

func TestAppendNew(t *testing.T) {
	ol1 := []int{0, 1}
	ol1Original := copier.TestDeep(t, ol1)

	cbCounter := 0
	ol2 := AppendNew(ol1, func(_ int) {
		cbCounter++
	}, []int{1, 2, 3}...)

	assert.NotSame(t, &ol1, &ol2, "should return copy of objects")
	assert.Exactly(t, ol1Original, ol1, "should not modify the source objects")

	assert.Exactly(t, []int{0, 1, 2, 3}, ol2, "should add unique numbers")
	assert.Exactly(t, 2, cbCounter, "callback should be called that amount of times")

	assert.NotPanics(t, func() { AppendNew(ol1, nil, []int{4}...) }, "should not panic if callback is nil")
}

func TestRemoveLast(t *testing.T) {
	ol1 := []string{"C", "A", "", "A", "B"}
	ol1Original := copier.TestDeep(t, ol1)

	ol2 := RemoveLast(ol1, "A")
	assert.NotSame(t, &ol1, &ol2, "should return copy of objects")
	assert.Exactly(t, ol1Original, ol1, "should not modify the source")

	expected := []string{"C", "A", "", "B"}
	assert.Exactly(t, expected, ol2, "should remove last object for which predicate returns true")
}

func TestRemoveFirst(t *testing.T) {
	ol1 := []string{"C", "A", "", "A", "B"}
	ol1Original := copier.TestDeep(t, ol1)

	ol2 := RemoveFirst(ol1, "A")
	assert.NotSame(t, &ol1, &ol2, "should return copy of objects")
	assert.Exactly(t, ol1Original, ol1, "should not modify the source")

	expected := []string{"C", "", "A", "B"}
	assert.Exactly(t, expected, ol2, "should remove first object for which predicate returns true")
}

func TestFilled(t *testing.T) {
	assert.Exactly(t, []string{}, Filled("a", 0))
	assert.Exactly(t, []string{"", ""}, Filled("", 2))
	assert.Exactly(t, []string{"a", "a", "a"}, Filled("a", 3))
}

func TestContainsAny(t *testing.T) {
	assert.True(t, ContainsAny("some words", "some"), "should contain element")
	assert.True(t, ContainsAny("some words", "words"), "should contain element")
	assert.True(t, ContainsAny("some words", "unknown", "some"), "should contain element")
	assert.False(t, ContainsAny("some words", "unknown", "unknown 2"), "should not contain any element")
}

func TestHasAnyPrefix(t *testing.T) {
	assert.True(t, HasAnyPrefix("some words", "some"), "should have prefix")
	assert.False(t, HasAnyPrefix("some words", "words"), "should not have prefix")
	assert.True(t, HasAnyPrefix("some words", "unknown", "some"), "should have one of prefixes")
	assert.False(t, HasAnyPrefix("some words", "unknown", "unknown 2"), "should not contain any prefix")
}

func TestIsAllEmpty(t *testing.T) {
	type test struct{}
	assert.True(t, IsAllEmpty([]int{}, []int{}))
	assert.False(t, IsAllEmpty([]string{"something"}, []string{""}))
	assert.True(t, IsAllEmpty(make([]bool, 0), []bool{}))
	assert.False(t, IsAllEmpty([]bool{}, make([]bool, 1)))
	assert.True(t, IsAllEmpty([]test{}, nil))
	assert.False(t, IsAllEmpty(nil, []test{{}}))
}

func TestMapFindDuplBy(t *testing.T) {
	type Struct struct {
		Str string
		Int int
	}

	ol1 := []Struct{{Str: "A", Int: 0}, {Str: "B", Int: 0}, {Str: "A", Int: 0}, {Str: "B", Int: 0}, {Str: "C", Int: 0}}
	ol1Original := copier.TestDeep(t, ol1)

	// Map to same type
	ol2Structs := MapFindDuplBy(ol1, func(elm Struct) string {
		return elm.Str
	}, func(elm Struct, idx int, dupl bool) Struct {
		if dupl {
			elm.Int = 1
		}
		return elm
	})
	assert.NotSame(t, &ol1, &ol2Structs, "should return copy of objects")
	assert.Exactly(t, ol1Original, ol1, "should not modify the source")

	expectedStructs := []Struct{
		{Str: "A", Int: 0},
		{Str: "B", Int: 0},
		{Str: "A", Int: 1},
		{Str: "B", Int: 1},
		{Str: "C", Int: 0},
	}
	assert.Exactly(t, expectedStructs, ol2Structs, "should set Int field for duplicates to 1")

	// Map to different type
	ol2Ints := MapFindDuplBy(ol1, func(elm Struct) string {
		return elm.Str
	}, func(elm Struct, idx int, dupl bool) int {
		if dupl {
			return 1
		}
		return 0
	})
	assert.NotSame(t, &ol1, &ol2Ints, "should return copy of objects")
	assert.Exactly(t, ol1Original, ol1, "should not modify the source")

	expectedInts := []int{0, 0, 1, 1, 0}
	assert.Exactly(t, expectedInts, ol2Ints, "should return that integer array")
}
