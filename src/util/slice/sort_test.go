package slice

import (
	"m3u_merge_astra/util/copier"
	"m3u_merge_astra/util/logger"
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestSort(t *testing.T) {
	log := logger.New(logrus.DebugLevel)

	ol1 := []TestNamedStruct{
		{Name: "C"}, {Name: "A"}, {}, {Name: "B"},
	}
	ol1Original := copier.TDeep(t, ol1)

	ol2 := Sort(log, ol1, "test objects")
	assert.NotSame(t, &ol1, &ol2, "should return copy of objects")
	assert.Exactly(t, ol1Original, ol1, "should not modify the source")

	expected := []TestNamedStruct{{Name: ""}, {Name: "A"}, {Name: "B"}, {Name: "C"}}
	assert.Exactly(t, expected, ol2, "should sort objects by name")
}
