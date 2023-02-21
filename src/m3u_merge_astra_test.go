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
	m3uBytes := []byte(``)

	err = os.WriteFile(m3uPath, m3uBytes, 0644)
	assert.NoError(t, err, "should write m3u file")

	astraCfgInputPath := filepath.Join(t.TempDir(), "m3u_merge_astra_main_test.json")
	astraCfgInputBytes := []byte(`{}`)

	err = os.WriteFile(astraCfgInputPath, astraCfgInputBytes, 0644)
	assert.NoError(t, err, "should write input astra config file")

	os.Args = []string{"", "-c", programCfgPath, "-m", m3uPath, "-i", astraCfgInputPath, "-o", os.DevNull}
	main()
}
