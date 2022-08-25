package slice

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRxAnyMatch(t *testing.T) {
	rxList := []regexp.Regexp{*regexp.MustCompile(`^a.*`), *regexp.MustCompile(`b`)}
	assert.True(t, RxAnyMatch(rxList, "a0"), "should match")
	assert.False(t, RxAnyMatch(rxList, "c"), "should not match")
}
