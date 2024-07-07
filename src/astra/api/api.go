package api

import (
	"bytes"
	"fmt"
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

// setStreamReq represents request to set stream
type setStreamReq struct {
	Cmd    string       `json:"cmd"`
	ID     string       `json:"id"`
	Stream astra.Stream `json:"stream"`
}

// setStreamResp represents response to setting stream
type setStreamResp struct {
	Status string `json:"set-stream"`
}

// setCategoryReq represents request to set category
type setCategoryReq struct {
	Cmd      string         `json:"cmd"`
	Category astra.Category `json:"category"`
}

// setCategoryResp represents response to setting category
type setCategoryResp struct {
	Status string `json:"set-category"`
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

// SetCategory makes a request to API setting <category>
func (h handler) SetCategory(category astra.Category) error {
	respBytes, err := h.request("POST", "/control/", setCategoryReq{Cmd: "set-category", Category: category})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Set category %v", category.Name))
	}

	var resp setCategoryResp
	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Invalid response while setting category %v", category.Name))
	}
	if resp.Status != "ok" {
		return errors.Wrap(err, fmt.Sprintf("Bad response while setting category %v (%v)", category.Name, resp.Status))
	}

	return nil
}

// SetStream makes a request to API setting stream with <id> to <stream>
func (h handler) SetStream(id string, stream astra.Stream) error {
	respBytes, err := h.request("POST", "/control/", setStreamReq{Cmd: "set-stream", ID: id, Stream: stream})
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Set stream with ID %v", id))
	}

	var resp setStreamResp
	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		return errors.Wrap(err, fmt.Sprintf("Invalid response while setting stream with ID %v", id))
	}
	if resp.Status != "ok" {
		return errors.Wrap(err, fmt.Sprintf("Bad response while setting stream with ID %v (%v)", id, resp.Status))
	}

	return nil
}

// FetchCfg makes a request to API and returns astra config
func (h handler) FetchCfg() (astra.Cfg, error) {
	respBytes, err := h.request("POST", "/control/", basicReq{Cmd: "load"})
	if err != nil {
		return astra.Cfg{}, errors.Wrap(err, "Fetch astra config")
	}

	var cfg astra.Cfg
	err = json.Unmarshal(respBytes, &cfg)
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

	req, err := http.NewRequest(method, h.address+path, bytes.NewBuffer(reqBody))
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
