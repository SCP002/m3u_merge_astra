package slice

import (
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestAnyRxMatch(t *testing.T) {
	rxList := []regexp.Regexp{*regexp.MustCompile(`^a.*`), *regexp.MustCompile(`b`)}
	assert.True(t, AnyRxMatch(rxList, "a0"), "should match")
	assert.False(t, AnyRxMatch(rxList, "c"), "should not match")
}
