package api

import (
	"fmt"
	"m3u_merge_astra/astra"
	"m3u_merge_astra/cfg"
	"m3u_merge_astra/util/logger"
	"m3u_merge_astra/util/network"
	"m3u_merge_astra/util/rnd"
	"testing"
	"time"

	"github.com/google/go-cmp/cmp"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
	"github.com/zenizh/go-capturer"
)

func TestNewHandler(t *testing.T) {
	log := logger.New(logrus.DebugLevel)
	httpClient := network.NewHttpClient(time.Second * 3)

	expected := handler{
		log:        log,
		httpClient: httpClient,
		address:    "127.0.0.1",
		user:       "user",
		password:   "pass",
	}
	assert.Exactly(t, expected, NewHandler(log, httpClient, "127.0.0.1", "user", "pass"), "should initialize handler")
}

// Requires a running astra
func TestSetCategories(t *testing.T) {
	log := logger.New(logrus.DebugLevel)
	httpClient := network.NewHttpClient(time.Second * 3)
	apiHandler := NewHandler(log, httpClient, "http://127.0.0.1:8000", "admin", "admin")

	// Remove existing categories
	config, err := apiHandler.FetchCfg()
	assert.NoError(t, err, "should not return error")
	for _, category := range config.Categories {
		category.Remove = true
		err := apiHandler.SetCategory(0, category)
		assert.NoError(t, err, "should not return error")
	}

	// Set
	idxCategoryMap := []lo.Entry[int, astra.Category]{
		{ // 0
			Key:   -1,
			Value: astra.Category{Name: "Category 1", Groups: []astra.Group{{Name: "Group 1"}, {Name: "Group 2"}}},
		},
		{ // 1
			Key:   -1,
			Value: astra.Category{Name: "Category 2", Groups: []astra.Group{{Name: "Group 3"}, {Name: "Group 4"}}},
		},
		{ // 2
			Key:   -1,
			Value: astra.Category{Name: "Category 3", Groups: []astra.Group{{Name: "Group 5"}, {Name: "Group 6"}}},
		},
		{ // 3
			Key:   -1,
			Value: astra.Category{Name: "Category 4", Groups: []astra.Group{{Name: "Group 7"}, {Name: "Group 8"}}},
		},
		{ // 4
			Key:   -1,
			Value: astra.Category{Name: "Category 5", Groups: []astra.Group{{Name: "Group 9"}, {Name: "Group 10"}}},
		},
		{ // 5
			Key:   1,
			Value: astra.Category{Name: "Category 2*", Groups: []astra.Group{{Name: "Group 3*"}, {Remove: true}}},
		},
		{ // 4
			Key:   -1,
			Value: astra.Category{Name: "Category 6", Groups: []astra.Group{{Name: "Group 11"}, {Name: "Group 12"}}},
		},
		{ // 6
			Key:   4,
			Value: astra.Category{Remove: true},
		},
		{ // 7
			Key:   3,
			Value: astra.Category{Remove: true},
		},
	}
	apiHandler.SetCategories(idxCategoryMap)

	// Check
	config, err = apiHandler.FetchCfg()
	assert.NoError(t, err, "should not return error")
	expected := []astra.Category{
		{Name: "Category 1", Groups: []astra.Group{{Name: "Group 1"}, {Name: "Group 2"}}},
		{Name: "Category 2*", Groups: []astra.Group{{Name: "Group 3*"}, {Remove: true}}},
		{Name: "Category 3", Groups: []astra.Group{{Name: "Group 5"}, {Name: "Group 6"}}},
		{Name: "Category 6", Groups: []astra.Group{{Name: "Group 11"}, {Name: "Group 12"}}},
	}
	assert.Equal(t, expected, config.Categories, "returned config should consist of categories set")

	// Test log output
	out := capturer.CaptureStderr(func() {
		log := logger.New(logrus.DebugLevel)
		httpClient := network.NewHttpClient(time.Second * 3)
		apiHandler := NewHandler(log, httpClient, "http://127.0.0.1:8000", "admin", "admin")
		idxCategoryMap = []lo.Entry[int, astra.Category]{
			{
				Key:   -1,
				Value: astra.Category{Name: "Category 0", Groups: []astra.Group{{Name: "Group 0"}, {Name: "Group 01"}}},
			},
		}
		apiHandler.SetCategories(idxCategoryMap)
	})
	assert.Contains(t, out, `Successfully set category: name "Category 0", groups "[{Name:Group 0 Remove:false} `+
		`{Name:Group 01 Remove:false}]", remove "false"`)
	assert.NotContains(t, out, "Failed")
}

// Requires a running astra
func TestSetCategory(t *testing.T) {
	log := logger.New(logrus.DebugLevel)
	httpClient := network.NewHttpClient(time.Second * 3)
	apiHandler := NewHandler(log, httpClient, "http://127.0.0.1:8000", "admin", "admin")

	astraCfg, err := apiHandler.FetchCfg()
	assert.NoError(t, err, "should not return error")

	// Remove old categories
	for _, category := range astraCfg.Categories {
		category.Remove = true
		err = apiHandler.SetCategory(0, category)
		if ok := assert.NoError(t, err, "should remove category"); !ok {
			t.FailNow()
		}
	}

	// Set new category
	err = apiHandler.SetCategory(-1, astra.Category{
		Name:   fmt.Sprintf("Category %v", rnd.String(4, false, true)),
		Groups: []astra.Group{{Name: "Group name 1"}, {Name: "Group name 2"}, {Name: "Group name 3"}},
	})
	assert.NoError(t, err, "should not return error")

	// Set new stream (test for crash when changing categories with existing streams)
	err = apiHandler.SetStream("0001", astra.Stream{
		ID:      "0001",
		Name:    "Name 1",
		Enabled: true,
		Type:    string(cfg.SPTS),
	})
	assert.NoError(t, err, "should not return error")

	astraCfg, err = apiHandler.FetchCfg()
	assert.NoError(t, err, "should not return error")

	modifiedCategory := astra.Category{
		Name: "Category modified",
		Groups: []astra.Group{
			{Name: "Group name 1", Remove: true},
			{Name: "Group modified"},
			{Name: "Group name 3", Remove: true},
		},
	}
	err = apiHandler.SetCategory(0, modifiedCategory)
	assert.NoError(t, err, "should not return error")

	astraCfg, err = apiHandler.FetchCfg()
	assert.NoError(t, err, "should not return error")
	assert.Equal(t, astraCfg.Categories, []astra.Category{modifiedCategory},
		"category in returned config should be category set")
}

func TestSetStreams(t *testing.T) {
	log := logger.New(logrus.DebugLevel)
	httpClient := network.NewHttpClient(time.Second * 3)
	apiHandler := NewHandler(log, httpClient, "http://127.0.0.1:8000", "admin", "admin")

	// Remove existing streams
	config, err := apiHandler.FetchCfg()
	assert.NoError(t, err, "should not return error")
	apiHandler.SetStreams(lo.Map(config.Streams, func(s astra.Stream, _ int) astra.Stream {
		s.Remove = true
		return s
	}))

	// Set
	streams := []astra.Stream{
		{ID: "0001", Name: "Name 1", Type: string(cfg.SPTS)},
		{ID: "0002", Name: "Name 2", Type: string(cfg.SPTS)},
		{ID: "0003", Name: "Name 3", Type: string(cfg.SPTS)},
	}
	apiHandler.SetStreams(streams)

	// Check
	config, err = apiHandler.FetchCfg()
	assert.NoError(t, err, "should not return error")
	assert.Equal(t, streams, config.Streams, "returned config should consist of streams set")

	// Test log output
	out := capturer.CaptureStderr(func() {
		log := logger.New(logrus.DebugLevel)
		httpClient := network.NewHttpClient(time.Second * 3)
		apiHandler := NewHandler(log, httpClient, "http://127.0.0.1:8000", "admin", "admin")
		streams := []astra.Stream{
			{ID: "0000", Name: "Name 0", Type: string(cfg.SPTS)},
		}
		apiHandler.SetStreams(streams)
	})
	assert.Contains(t, out, `Successfully set stream: ID "0000", name "Name 0"`)
	assert.NotContains(t, out, "Failed")
}

// Requires a running astra
func TestSetStream(t *testing.T) {
	log := logger.New(logrus.DebugLevel)
	httpClient := network.NewHttpClient(time.Second * 3)
	apiHandler := NewHandler(log, httpClient, "http://127.0.0.1:8000", "admin", "admin")

	// Set
	streamName := fmt.Sprintf("Stream %v", rnd.String(4, false, true))
	err := apiHandler.SetStream("0000", astra.Stream{
		Enabled: true,
		ID:      "0000",
		Inputs:  []string{"http://xxx/2", "http://xxx"},
		Groups:  map[string]string{"Category 1": "Group 1"},
		Name:    streamName,
		Type:    string(cfg.SPTS),
	})
	assert.NoError(t, err, "should not return error")

	// Change
	err = apiHandler.SetStream("0000", astra.Stream{
		Enabled: true,
		ID:      "0000",
		Inputs:  []string{"http://xxx"},
		Name:    streamName,
		Type:    string(cfg.SPTS),
	})
	assert.NoError(t, err, "should not return error")

	// Check
	cfg, err := apiHandler.FetchCfg()
	assert.NoError(t, err, "should not return error")
	assert.True(t, lo.ContainsBy(cfg.Streams, func(s astra.Stream) bool {
		return s.ID == "0000" && s.Name == streamName && cmp.Equal(s.Inputs, []string{"http://xxx"})
	}), "returned config should contain data from stream set")
}

// Requires a running astra
func TestFetchCfg(t *testing.T) {
	log := logger.New(logrus.DebugLevel)
	httpClient := network.NewHttpClient(time.Second * 3)
	apiHandler := NewHandler(log, httpClient, "http://127.0.0.1:8000", "admin", "admin")
	astraCfg, err := apiHandler.FetchCfg()
	assert.NotEmpty(t, astraCfg, "should return not empty config")
	assert.NoError(t, err, "should not return error")
}

// Requires a running astra
func TestRequest(t *testing.T) {
	log := logger.New(logrus.DebugLevel)
	httpClient := network.NewHttpClient(time.Second * 3)
	apiHandler := NewHandler(log, httpClient, "http://127.0.0.1:8000", "admin", "admin")
	resp, err := apiHandler.request("POST", "/control/", basicReq{Cmd: "sessions"})
	assert.Contains(t, string(resp), "sessions", "should return sessions list")
	assert.NoError(t, err, "should not return error")
}
