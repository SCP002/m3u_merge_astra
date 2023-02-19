package url

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestEqual(t *testing.T) {
	// Both good, with hash
	equal, err := Equal("http://url/1", "http://url/1", true)
	assert.True(t, equal, "should be equal")
	assert.NoError(t, err, "should not return error")

	equal, err = Equal("http://url/1#a", "http://url/1#b", true)
	assert.False(t, equal, "should not be equal")
	assert.NoError(t, err, "should not return error")

	equal, err = Equal("http://url/1#a", "http://url/2#a", true)
	assert.False(t, equal, "should not be equal")
	assert.NoError(t, err, "should not return error")

	// Both bad, with hash
	equal, err = Equal("http://{bad/url/1", "http://{bad/url/1", true)
	assert.True(t, equal, "should be equal")
	assert.NoError(t, err, "should not return error")

	equal, err = Equal("http://{bad/url/1#a", "http://{bad/url/1#b", true)
	assert.False(t, equal, "should not be equal")
	assert.NoError(t, err, "should not return error")

	equal, err = Equal("http://{bad/url/1#a", "http://{bad/url/2#a", true)
	assert.False(t, equal, "should not be equal")
	assert.NoError(t, err, "should not return error")

	// Both good, no hash
	equal, err = Equal("http://url/1", "http://url/1", false)
	assert.True(t, equal, "should be equal")
	assert.NoError(t, err, "should not return error")

	equal, err = Equal("http://url/1#a", "http://url/1#b", false)
	assert.True(t, equal, "should be equal")
	assert.NoError(t, err, "should not return error")

	equal, err = Equal("http://url/1#a", "http://url/2#a", false)
	assert.False(t, equal, "should not be equal")
	assert.NoError(t, err, "should not return error")

	// Right bad, no hash
	equal, err = Equal("http://url/1", "http://{bad/url/1", false)
	assert.False(t, equal, "should not be equal")
	assert.Error(t, err, "should return error")

	equal, err = Equal("http://url/1#a", "http://{bad/url/1#b", false)
	assert.False(t, equal, "should not be equal")
	assert.Error(t, err, "should return error")

	equal, err = Equal("http://url/1#a", "http://{bad/url/2#a", false)
	assert.False(t, equal, "should not be equal")
	assert.Error(t, err, "should return error")

	// Both bad, no hash
	equal, err = Equal("http://{bad/url/1", "http://{bad/url/1", false)
	assert.True(t, equal, "should be equal")
	assert.Error(t, err, "should return error")

	equal, err = Equal("http://{bad/url/1#a", "http://{bad/url/1#b", false)
	assert.False(t, equal, "should not be equal")
	assert.Error(t, err, "should return error")

	equal, err = Equal("http://{bad/url/1#a", "http://{bad/url/2#a", false)
	assert.False(t, equal, "should not be equal")
	assert.Error(t, err, "should return error")
}
