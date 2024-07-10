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
	"github.com/samber/lo"
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
	Error  string `json:"error"`
}

// setCategoryReq represents request to set category
type setCategoryReq struct {
	Cmd      string         `json:"cmd"`
	ID       *int           `json:"id,omitempty"`
	Category astra.Category `json:"category"`
}

// setCategoryResp represents response to setting category
type setCategoryResp struct {
	Status string `json:"set-category"`
	Error  string `json:"error"`
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

// SetCategories makes a requests to API setting categories by indexes as defined in <idxCategoryMap> synchronously.
//
// Use negative key (index) in <idxCategoryMap> to create new category.
func (h handler) SetCategories(idxCategoryMap []lo.Entry[int, astra.Category]) {
	h.log.Info("Sending changed categories to astra")

	for _, entry := range idxCategoryMap {
		err := h.SetCategory(entry.Key, entry.Value)
		if err == nil {
			h.log.InfoCFi("Successfully set category", "name", entry.Value.Name, "groups", entry.Value.Groups)
		} else {
			h.log.ErrorCFi("Failed to set category", "name", entry.Value.Name, "groups", entry.Value.Groups, "error",
				err)
		}
	}
}

// SetCategory makes a request to API setting category with <idx> to <category>.
//
// To create new category, pass negative <idx>.
func (h handler) SetCategory(idx int, category astra.Category) error {
	req := setCategoryReq{Cmd: "set-category", ID: lo.Ternary(idx >= 0, &idx, nil), Category: category}

	respBytes, err := h.request("POST", "/control/", req)
	if err != nil {
		return err
	}

	var resp setCategoryResp
	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		return errors.Wrap(err, "Invalid API response")
	}
	if resp.Status != "ok" {
		return errors.Wrap(err, fmt.Sprintf("API responded with bad status (%v)", resp.Status))
	}
	if resp.Error != "" {
		return errors.Wrap(err, fmt.Sprintf("API responded with error (%v)", resp.Error))
	}

	return nil
}

// SetStreams makes a requests to API setting <streams> synchronously
func (h handler) SetStreams(streams []astra.Stream) {
	h.log.Info("Sending changed streams to astra")

	for _, stream := range streams {
		err := h.SetStream(stream.ID, stream)
		if err == nil {
			h.log.InfoCFi("Successfully set stream", "ID", stream.ID, "name", stream.Name)
		} else {
			h.log.ErrorCFi("Failed to set stream", "ID", stream.ID, "name", stream.Name, "error", err)
		}
	}
}

// SetStream makes a request to API setting stream with <id> to <stream>
func (h handler) SetStream(id string, stream astra.Stream) error {
	respBytes, err := h.request("POST", "/control/", setStreamReq{Cmd: "set-stream", ID: id, Stream: stream})
	if err != nil {
		return err
	}

	var resp setStreamResp
	err = json.Unmarshal(respBytes, &resp)
	if err != nil {
		return errors.Wrap(err, "Invalid API response")
	}
	if resp.Status != "ok" {
		return errors.Wrap(err, fmt.Sprintf("API responded with bad status (%v)", resp.Status))
	}
	if resp.Error != "" {
		return errors.Wrap(err, fmt.Sprintf("API responded with error (%v)", resp.Error))
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
