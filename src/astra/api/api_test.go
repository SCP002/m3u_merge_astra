package api

import (
	"m3u_merge_astra/astra"
	"m3u_merge_astra/util/logger"
	"m3u_merge_astra/util/network"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
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
func TestSetStream(t *testing.T) {
	log := logger.New(logrus.DebugLevel)
	httpClient := network.NewHttpClient(time.Second * 3)
	apiHandler := NewHandler(log, httpClient, "http://127.0.0.1:8000", "admin", "admin")
	err := apiHandler.SetStream("0000", astra.Stream{
		Enabled: true,
		ID: "0000",
		Inputs: []string{"http://tv.lan:8000/play/cgnj", "http://xxx"},
		Name: "NAME",
		Type: "spts",
	})
	assert.NoError(t, err, "should not return error")
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
