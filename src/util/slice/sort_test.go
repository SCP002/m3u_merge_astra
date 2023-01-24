package slice

import (
	"m3u_merge_astra/util/copier"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSort(t *testing.T) {

	ol1 := []TestNamedStruct{
		{Name: "C"}, {Name: "A"}, {}, {Name: "B"},
	}
	ol1Original := copier.TestDeep(t, ol1)

	ol2 := Sort(ol1)
	assert.NotSame(t, &ol1, &ol2, "should return copy of objects")
	assert.Exactly(t, ol1Original, ol1, "should not modify the source")

	expected := []TestNamedStruct{{Name: ""}, {Name: "A"}, {Name: "B"}, {Name: "C"}}
	assert.Exactly(t, expected, ol2, "should sort objects by name")
}
