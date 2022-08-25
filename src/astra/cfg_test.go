package astra

import (
	"os"
	"path/filepath"
	"testing"

	"m3u_merge_astra/cli"

	"github.com/stretchr/testify/assert"
)

func TestWriteReadCfg(t *testing.T) {
	c1 := Cfg{
		Streams: []Stream{{ID: "0000"}},
		Unknown: map[string]any{
			"users": map[string]any{
				"user1": map[string]any{
					"enable": true,
				},
			},
			"gid": float64(111111),
		},
	}

	path := filepath.Join(os.TempDir(), "m3u_merge_astra_test_write_cfg.json")
	defer os.Remove(path)

	// Write to file
	err := WriteCfg(c1, path)
	assert.NoError(t, err, "should not return error")

	// Read from file
	c2, err := ReadCfg(path)
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, c1, c2, "config should stay the same")

	// Write to clipboard
	err = WriteCfg(c1, string(cli.Clipboard))
	assert.NoError(t, err, "should not return error")

	// Read from clipboard
	c2, err = ReadCfg(string(cli.Clipboard))
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, c1, c2, "config should stay the same")

	// Redirect stdout to stdin for testing
	r, w, err := os.Pipe()
	assert.NoError(t, err, "should not return error")
	os.Stdin = r
	os.Stdout = w

	// Write to stdout
	err = WriteCfg(c1, string(cli.Stdio))
	w.Close()
	assert.NoError(t, err, "should not return error")

	// Read from stdin
	c2, err = ReadCfg(string(cli.Stdio))
	r.Close()
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, c1, c2, "config should stay the same")
}
