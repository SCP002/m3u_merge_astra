package iter

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestTimes(t *testing.T) {
	count := 0
	Times(10, func(iteration int) bool {
		count++
		return true
	})
	assert.Equal(t, 10, count, "should iterate 10 times")

	count = 0
	Times(10, func(iteration int) bool {
		count++
		return iteration != 5
	})
	assert.Equal(t, 5, count, "should iterate 5 times")

	count = 0
	Times(10, func(iteration int) bool {
		count++
		return false
	})
	assert.Equal(t, 1, count, "should iterate once")

	count = 0
	Times(0, func(iteration int) bool {
		count++
		return true
	})
	assert.Equal(t, 0, count, "should not iterate")
}

func TestForEach(t *testing.T) {
	slice := []string{"a", "b", "c"}

	result := []string{}
	ForEach(slice, func(item string) {
		result = append(result, item)
	})

	expected := []string{"a", "b", "c"}
	assert.Exactly(t, expected, result, "should have these values")

	slice = []string{}

	result = []string{}
	ForEach(slice, func(_ string) {
		result = append(result, "a")
	})

	expected = []string{}
	assert.Exactly(t, expected, result, "should not run for empty input")
}
