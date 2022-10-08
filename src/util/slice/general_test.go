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
	ol1Original := copier.TDeep(t, ol1)

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
	ol1Original := copier.TDeep(t, ol1)

	ol2 := RemoveLast(ol1, "A")
	assert.NotSame(t, &ol1, &ol2, "should return copy of objects")
	assert.Exactly(t, ol1Original, ol1, "should not modify the source")

	expected := []string{"C", "A", "", "B"}
	assert.Exactly(t, expected, ol2, "should remove last object for which predicate returns true")
}
