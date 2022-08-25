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

func TestRemoveLast(t *testing.T) {
	ol1 := []string{"C", "A", "", "A", "B"}
	ol1Original := copier.TDeep(t, ol1)

	ol2 := RemoveLast(ol1, "A")
	assert.NotSame(t, &ol1, &ol2, "should return copy of objects")
	assert.Exactly(t, ol1Original, ol1, "should not modify the source")

	expected := []string{"C", "A", "", "B"}
	assert.Exactly(t, expected, ol2, "should remove last object for which predicate returns true")
}
