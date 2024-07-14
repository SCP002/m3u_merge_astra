package main

import (
	"m3u_merge_astra/util/file"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

// Requires running astra
func TestMainCrash(t *testing.T) { // TODO: Finish this test
	programCfgPath := filepath.Join(t.TempDir(), "m3u_merge_astra_main_test.yaml")

	// Create program config file
	err := file.Copy(filepath.Join("cfg", "default.yaml"), programCfgPath)
	assert.NoError(t, err, "should copy default program config")

	// Create M3U file
	m3uPath := filepath.Join(t.TempDir(), "m3u_merge_astra_main_test.m3u")
	m3uBytes := []byte(`#EXTM3U
	#EXTINF:-1 group-title="Group 3",Channel 3
	http://url/3
	#EXTINF:-1 group-title="Group 4",Channel 4
	http://url/4`)
	err = os.WriteFile(m3uPath, m3uBytes, 0644)
	assert.NoError(t, err, "should write m3u file")

	// Read initial astra config
	// ...

	// Clean existing streams
	// ...

	// Clean existing categories
	// ...

	// Add streams
	// ...

	// Add categories
	// ...

	// Run program
	os.Args = []string{"", "-l", "6", "-c", programCfgPath, "-m", m3uPath, "-u", "admin", "-p", "admin"}
	main()

	// Read modified config
	// ...

	// Check modified config
	// ...
}
