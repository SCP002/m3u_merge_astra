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

func TestFilled(t *testing.T) {
	assert.Exactly(t, []string{}, Filled("a", 0))
	assert.Exactly(t, []string{"", ""}, Filled("", 2))
	assert.Exactly(t, []string{"a", "a", "a"}, Filled("a", 3))
}

func TestContainsAny(t *testing.T) {
	assert.True(t, ContainsAny("some words", "some"), "should contain element")
	assert.True(t, ContainsAny("some words", "unknown", "some"), "should contain element")
	assert.False(t, ContainsAny("some words", "unknown", "unknown 2"), "should not contain any element")
}

func TestIsAllEmpty(t *testing.T) {
	type test struct {}
	assert.True(t, IsAllEmpty([]int{}, []int{}))
	assert.False(t, IsAllEmpty([]string{"something"}, []string{""}))
	assert.True(t, IsAllEmpty(make([]bool, 0), []bool{}))
	assert.False(t, IsAllEmpty([]bool{}, make([]bool, 1)))
	assert.True(t, IsAllEmpty([]test{}, nil))
	assert.False(t, IsAllEmpty(nil, []test{{}}))
}
