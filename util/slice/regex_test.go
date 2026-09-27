package slice

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRxMatchAny(t *testing.T) {
	rx := *regexp.MustCompile(`^a.*`)
	assert.True(t, RxMatchAny(rx, "b", "a0"), "should match")
	assert.False(t, RxMatchAny(rx, "b", "c"), "should not match")
}

func TestAnyRxMatch(t *testing.T) {
	rxList := []regexp.Regexp{*regexp.MustCompile(`^a.*`), *regexp.MustCompile(`b`)}
	assert.True(t, AnyRxMatch(rxList, "a0"), "should match")
	assert.False(t, AnyRxMatch(rxList, "c"), "should not match")
}

func TestAnyRxMatchAny(t *testing.T) {
	rxList := []regexp.Regexp{*regexp.MustCompile(`^a.*`), *regexp.MustCompile(`b`)}
	assert.True(t, AnyRxMatchAny(rxList, "a0"), "should match")
	assert.True(t, AnyRxMatchAny(rxList, "a0", "b"), "should match")
	assert.True(t, AnyRxMatchAny(rxList, "a0", "c"), "should match")
	assert.True(t, AnyRxMatchAny(rxList, "b", "c"), "should match")

	assert.False(t, AnyRxMatchAny(rxList, "c"), "should not match")
	assert.False(t, AnyRxMatchAny(rxList, "c", "d"), "should not match")
}
