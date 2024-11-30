package main

import (
	"m3u_merge_astra/astra"
	"m3u_merge_astra/astra/api"
	"m3u_merge_astra/cfg"
	"m3u_merge_astra/util/file"
	"m3u_merge_astra/util/logger"
	"m3u_merge_astra/util/network"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

// Requires running astra
func TestMain(t *testing.T) {
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
	log := logger.New(logrus.DebugLevel)
	apiHttpClient := network.NewHttpClient(time.Second * 10)
	apiHandler := api.NewHandler(log, apiHttpClient, "http://127.0.0.1:8000", "admin", "admin")
	astraCfg, err := apiHandler.FetchCfg()
	assert.NoError(t, err, "should fetch astra config")

	// Clean existing streams
	for _, stream := range astraCfg.Streams {
		stream.Remove = true
		err := apiHandler.SetStream(stream.ID, stream)
		if ok := assert.NoError(t, err, "should remove stream"); !ok {
			t.FailNow()
		}
	}

	// Clean existing categories
	for _, category := range astraCfg.Categories {
		category.Remove = true
		err := apiHandler.SetCategory(0, category)
		if ok := assert.NoError(t, err, "should remove category"); !ok {
			t.FailNow()
		}
	}

	// Add streams
	apiHandler.SetStreams([]astra.Stream{
		{
			ID:      "0001",
			Name:    "Channel 1",
			Type:    string(cfg.SPTS),
			Enabled: true,
			Inputs:  []string{"http://url/1"},
			Groups:  map[string]string{"Category 1": "Group 1"},
		},
		{
			ID:      "0002",
			Name:    "Channel 2",
			Type:    string(cfg.SPTS),
			Enabled: true,
			Inputs:  []string{"http://url/2"},
			Groups:  map[string]string{"Category 1": "Group 2"},
		},
	})

	// Run program
	os.Args = []string{"", "-n", "-l", "6", "-c", programCfgPath, "-m", m3uPath, "-u", "admin", "-p", "admin"}
	main()

	// Read modified config
	astraCfg, err = apiHandler.FetchCfg()
	assert.NoError(t, err, "should fetch astra config")

	// Check modified config
	expectedCategories := []astra.Category{
		{Name: "Category 1", Groups: []astra.Group{{Name: "Group 1"}, {Name: "Group 2"}}},
	}
	assert.Exactly(t, expectedCategories, astraCfg.Categories, "should return that categories")

	// Checking fields separately as new streams contains random ID
	assert.Exactly(t, "Channel 1", astraCfg.Streams[0].Name, "should have stream with this name")
	assert.Exactly(t, "Channel 2", astraCfg.Streams[1].Name, "should have stream with this name")
	assert.Exactly(t, "_ADDED: Channel 3", astraCfg.Streams[2].Name, "should have stream with this name")
	assert.Exactly(t, "_ADDED: Channel 4", astraCfg.Streams[3].Name, "should have stream with this name")
}
