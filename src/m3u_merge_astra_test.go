package main

import (
	"m3u_merge_astra/util/file"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestMainCrash(t *testing.T) {
	programCfgPath := filepath.Join(t.TempDir(), "m3u_merge_astra_main_test.yaml")

	err := file.Copy(filepath.Join("cfg", "default.yaml"), programCfgPath)
	assert.NoError(t, err, "should copy default program config")

	m3uPath := filepath.Join(t.TempDir(), "m3u_merge_astra_main_test.m3u")
	m3uBytes := []byte(`#EXTM3U
	#EXTINF:-1 group-title="Group 3",Channel 3
	http://url/3
	#EXTINF:-1 group-title="Group 4",Channel 4
	http://url/4`)

	err = os.WriteFile(m3uPath, m3uBytes, 0644)
	assert.NoError(t, err, "should write m3u file")

	astraCfgInputPath := filepath.Join(t.TempDir(), "m3u_merge_astra_main_test.json")
	astraCfgInputBytes := []byte(`{
		"categories": [
		  {
			"name": "Category 1",
			"groups": [{ "name": "Group 1" }, { "name": "Group 2" }]
		  }
		],
		"make_stream": [
		  {
			"enable": true,
			"groups": { "Category 1": "Group 1" },
			"id": "0001",
			"input": ["http://url/1"],
			"name": "Channel 1",
			"type": "spts"
		  },
		  {
			"enable": true,
			"groups": { "Category 1": "Group 2" },
			"id": "0002",
			"input": ["http://url/2"],
			"name": "Channel 2",
			"type": "spts"
		  }
		]
	  }`)

	err = os.WriteFile(astraCfgInputPath, astraCfgInputBytes, 0644)
	assert.NoError(t, err, "should write input astra config file")

	os.Args = []string{"", "-c", programCfgPath, "-m", m3uPath, "-i", astraCfgInputPath, "-o", os.DevNull}
	main()
}
