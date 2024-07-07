package api

import (
	"bytes"
	"io"
	"net/http"

	"m3u_merge_astra/astra"
	"m3u_merge_astra/util/logger"

	json "github.com/SCP002/jsonexraw"
	"github.com/cockroachdb/errors"
)

// basicReq respresents basic request to astra API
type basicReq struct {
	Cmd string `json:"cmd"`
}

// handler holds dependencies and credentials to access astra API
type handler struct {
	log        *logger.Logger
	httpClient *http.Client
	address    string
	user       string
	password   string
}

// NewHandler returns new astra API handler
func NewHandler(log *logger.Logger, httpClient *http.Client, address string, user string, password string) handler {
	return handler{log: log, httpClient: httpClient, address: address, user: user, password: password}
}

// FetchCfg makes a request to API and returns astra config
func (h handler) FetchCfg() (astra.Cfg, error) {
	resp, err := h.request("POST", "/control/", basicReq{Cmd: "load"})
	if err != nil {
		return astra.Cfg{}, errors.Wrap(err, "Fetch astra config")
	}

	var cfg astra.Cfg
	err = json.Unmarshal(resp, &cfg)
	if err != nil {
		return astra.Cfg{}, errors.Wrap(err, "Decode astra config")
	}

	return cfg, nil
}

// request makes a request to astra API sending struct <cmd> in reqest body and returns response body as bytes
func (h handler) request(method string, path string, cmd any) ([]byte, error) {
	reqBody, err := json.Marshal(cmd)
	if err != nil {
		return nil, errors.Wrap(err, "Encode request to API")
	}

	req, err := http.NewRequest(method, h.address + path, bytes.NewBuffer(reqBody))
	if err != nil {
		return nil, errors.Wrap(err, "Create HTTP request instance to API")
	}
	req.Header.Add("Content-Type", "application/json")
	req.SetBasicAuth(h.user, h.password)

	resp, err := h.httpClient.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "Send HTTP request to API")
	}

	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "Read API response body")
	}

	return respBody, nil
}
