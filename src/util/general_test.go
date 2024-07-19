package util

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
