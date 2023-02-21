package file

import (
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestCopy(t *testing.T) {
	path := filepath.Join(t.TempDir(), "m3u_merge_astra_copy_test.txt")
	
	err := Copy("copy_test.txt", path)
	assert.NoError(t, err, "should not return error")

	// Test overwrite
	err = Copy("copy_test.txt", path)
	assert.NoError(t, err, "should not return error")

	assert.FileExists(t, path, "should copy file")
}
