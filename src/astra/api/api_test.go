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

func TestAddCategory(t *testing.T) {
	log := logger.New(logrus.DebugLevel)
	httpClient := network.NewHttpClient(time.Second * 3)
	apiHandler := NewHandler(log, httpClient, "http://127.0.0.1:8000", "admin", "admin")

	categoryName := fmt.Sprintf("Category %v", rnd.String(4, false, true))
	err := apiHandler.AddCategory(astra.Category{
		Name:   categoryName,
		Groups: []astra.Group{{Name: "Group name 1"}, {Name: "Group name 2"}},
	})
	assert.NoError(t, err, "should not return error")

	cfg, err := apiHandler.FetchCfg()
	assert.NoError(t, err, "should not return error")
	assert.True(t, lo.ContainsBy(cfg.Categories, func(c astra.Category) bool {
		return c.Name == categoryName
	}), "returned config should contain data from category set")
}

// Requires a running astra
func TestSetCategory(t *testing.T) {
	log := logger.New(logrus.DebugLevel)
	httpClient := network.NewHttpClient(time.Second * 3)
	apiHandler := NewHandler(log, httpClient, "http://127.0.0.1:8000", "admin", "admin")

	err := apiHandler.AddCategory(astra.Category{
		Name:   fmt.Sprintf("Category %v", rnd.String(4, false, true)),
		Groups: []astra.Group{{Name: "Group name 1"}, {Name: "Group name 2"}},
	})
	assert.NoError(t, err, "should not return error")

	cfg, err := apiHandler.FetchCfg()
	assert.NoError(t, err, "should not return error")

	modifiedCategory := astra.Category{
		Name:   "Category modified",
		Groups: []astra.Group{{Name: "Group modified"}},
	}
	err = apiHandler.SetCategory(len(cfg.Categories) - 1, modifiedCategory)
	assert.NoError(t, err, "should not return error")

	cfg, err = apiHandler.FetchCfg()
	assert.NoError(t, err, "should not return error")
	assert.Equal(t, cfg.Categories[len(cfg.Categories) - 1], modifiedCategory,
		"last category in returned config be category set")
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
	streamName := fmt.Sprintf("Stream %v", rnd.String(4, false, true))
	err := apiHandler.SetStream("0000", astra.Stream{
		Enabled: true,
		ID:      "0000",
		Inputs:  []string{"http://xxx/2", "http://xxx"},
		Name:    streamName,
		Type:    string(cfg.SPTS),
	})
	assert.NoError(t, err, "should not return error")

	cfg, err := apiHandler.FetchCfg()
	assert.NoError(t, err, "should not return error")
	assert.True(t, lo.ContainsBy(cfg.Streams, func(s astra.Stream) bool {
		return s.ID == "0000" && s.Name == streamName
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
