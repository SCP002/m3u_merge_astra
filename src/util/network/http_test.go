package network

import (
	"m3u_merge_astra/util/logger"
	"net/http"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestNewHttpServer(t *testing.T) {
	log := logger.New(logrus.DebugLevel)

	mux := http.NewServeMux()
	mux.HandleFunc("/ok/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(200)
	})
	mux.HandleFunc("/not_found/", func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
	})
	mux.HandleFunc("/timeout/", func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(time.Second * 5)
	})

	// Run http & https servers as subset of current test to be able to fail it from another goroutines (servers) if any
	// server returns error
	var httpSrv, httpsSrv *http.Server
	t.Run("http_server", func(t *testing.T) {
		httpSrv, httpsSrv = NewHttpServer(mux, 7878, 9090, func(err error) {
			if !errors.Is(err, http.ErrServerClosed) {
				// Not using logging from testing.T or else message will not be displayed
				log.Errorf("Test server stopped with non-standard error: %v", err)
				t.FailNow()
			}
		})
	})
	defer httpSrv.Close()
	defer httpsSrv.Close()

	client := NewHttpClient(false, time.Second*3)

	resp, err := client.Get("http://127.0.0.1:7878/ok/1")
	assert.Exactly(t, 200, resp.StatusCode, "should return OK status")
	assert.NoError(t, err, "should not return error")

	resp, err = client.Get("https://127.0.0.1:9090/ok/1")
	assert.Exactly(t, 200, resp.StatusCode, "should return OK status")
	assert.NoError(t, err, "should not return error")

	resp, err = client.Get("http://127.0.0.1:1010/ok/1")
	assert.Nil(t, resp, "should return empty reposnse")
	assert.Exactly(t, Refused, GetErrType(err), "should refuse connections to the wrong port")

	resp, err = client.Get("https://127.0.0.1:1010/ok/1")
	assert.Nil(t, resp, "should return empty reposnse")
	assert.Exactly(t, Refused, GetErrType(err), "should refuse connections to the wrong port")

	resp, err = client.Get("http://127.0.0.1:9090/ok/1")
	assert.Exactly(t, 400, resp.StatusCode, "should return Bad Request status (HTTP request to HTTPS server)")
	assert.NoError(t, err, "should not return error")

	resp, err = client.Get("https://127.0.0.1:7878/ok/1")
	assert.Nil(t, resp, "should return empty reposnse")
	assert.Exactly(t, HTTPSClientHTTPServer, GetErrType(err), "should return HTTP response to HTTPS client error")

	resp, err = client.Get("http://127.0.0.1:7878/not_found/1")
	assert.Exactly(t, 404, resp.StatusCode, "should return Not Found status")
	assert.NoError(t, err, "should not return error")

	resp, err = client.Get("https://127.0.0.1:9090/not_found/1")
	assert.Exactly(t, 404, resp.StatusCode, "should return Not Found status")
	assert.NoError(t, err, "should not return error")

	resp, err = client.Get("http://127.0.0.1:1010/not_found/1")
	assert.Nil(t, resp, "should return empty reposnse")
	assert.Exactly(t, Refused, GetErrType(err), "should refuse connections to the wrong port")

	resp, err = client.Get("https://127.0.0.1:1010/not_found/1")
	assert.Nil(t, resp, "should return empty reposnse")
	assert.Exactly(t, Refused, GetErrType(err), "should refuse connections to the wrong port")

	resp, err = client.Get("http://127.0.0.1:9090/not_found/1")
	assert.Exactly(t, 400, resp.StatusCode, "should return Bad Request status (HTTP request to HTTPS server)")
	assert.NoError(t, err, "should not return error")

	resp, err = client.Get("https://127.0.0.1:7878/not_found/1")
	assert.Nil(t, resp, "should return empty reposnse")
	assert.Exactly(t, HTTPSClientHTTPServer, GetErrType(err), "should return HTTP response to HTTPS client error")

	resp, err = client.Get("http://dead/no_such_host/1")
	assert.Nil(t, resp, "should return empty reposnse")
	assert.Exactly(t, NoSuchHost, GetErrType(err), "dead URL's should return DNS lookup error")

	resp, err = client.Get("https://dead/no_such_host/1")
	assert.Nil(t, resp, "should return empty reposnse")
	assert.Exactly(t, NoSuchHost, GetErrType(err), "dead URL's should return DNS lookup error")

	resp, err = client.Get("http://127.0.0.1:7878/timeout/1")
	assert.Nil(t, resp, "should return empty reposnse")
	assert.Exactly(t, Timeout, GetErrType(err), "should return timeout error")

	resp, err = client.Get("https://127.0.0.1:9090/timeout/1")
	assert.Nil(t, resp, "should return empty reposnse")
	assert.Exactly(t, Timeout, GetErrType(err), "should return timeout error")

	client = NewHttpClient(true, time.Second*3)

	resp, err = client.Get("http://example.com:7878/ok/1")
	assert.Exactly(t, 200, resp.StatusCode, "should return OK status")
	assert.NoError(t, err, "should not return error")

	resp, err = client.Get("https://example.com:9090/ok/1")
	assert.Exactly(t, 200, resp.StatusCode, "should return OK status")
	assert.NoError(t, err, "should not return error")
}
