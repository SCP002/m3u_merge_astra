package simplify

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestName(t *testing.T) {
	assert.Exactly(t, "samplename", Name("Sample, Name!\r\n"), "should return simplified name")
	assert.NotEqual(t, "samplename2", Name("Sample Name (+2)"), "should not discard the + symbol")
}
