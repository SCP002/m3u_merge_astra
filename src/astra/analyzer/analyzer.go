package analyzer

import (
	"context"
	"net/url"
	"time"

	json "github.com/SCP002/jsonexraw"
	"github.com/cockroachdb/errors"
	"github.com/gorilla/websocket"
	"github.com/samber/lo"
)

// request represets request to analyzer
type request struct {
	Cmd     string `json:"cmd"`
	Address string `json:"address"`
}

// response represents response from analyzer.
//
// Pointers are to distinguish between undefined and zero value.
type response struct {
	OnAir   *bool    `json:"on_air"`
	Cmd     *string  `json:"cmd"`
	Total   *total   `json:"total"`
	Streams []stream `json:"streams"`
}

// total represents aggregated information about stream since previous response
type total struct {
	BitrateLimit int  `json:"bitrate_limit"`
	CCErrors     int  `json:"cc_errors"`
	Scrambled    bool `json:"scrambled"`
	Packets      int  `json:"packets"`
	Bitrate      int  `json:"bitrate"`
	PCRErrors    int  `json:"pcr_errors"`
	SCErrors     int  `json:"sc_errors"`
	PESErrors    int  `json:"pes_errors"`
}

// stream respresents elementary stream (audio or video)
type stream struct {
	Descriptors []any  `json:"descriptors"`
	TypeID      int    `json:"type_id"`
	Pid         int    `json:"pid"`
	TypeName    string `json:"type_name"`
}

// Result represents check result containing averages of info such as bitrate and various errors
type Result struct {
	Bitrate   int // Kbit/s
	CCErrors  int
	PESErrors int
	Scrambled bool
	HasAudio  bool
	HasVideo  bool
}

// Check returns check result of <urlToCheck> using astra analyzer at <analyzerAddr> in format of 'host:port' with
// context <ctx>.
//
// Does Not return error if <urlToCheck> is dead or invalid, rely on bitrate == 0.
func Check(ctx context.Context, analyzerAddr, urlToCheck string) (Result, error) {
	url := url.URL{Scheme: "ws", Host: analyzerAddr, Path: "/api/"}
	conn, _, err := websocket.DefaultDialer.Dial(url.String(), nil)
	if err != nil {
		return Result{}, errors.Wrap(err, "Dial")
	}
	defer conn.Close()

	// Read responses
	readErrCh := make(chan error)
	readRespCh := make(chan response)

	go func() {
		defer close(readErrCh)
		defer close(readRespCh)
		for {
			_, respBytes, err := conn.ReadMessage()
			if err != nil {
				readErrCh <- errors.Wrap(err, "Read response")
				return
			}
			var resp response
			err = json.Unmarshal(respBytes, &resp)
			if err != nil {
				readErrCh <- errors.Wrap(err, "Decode response")
				return
			}
			readRespCh <- resp
		}
	}()

	// Send start request
	startReqBytes, err := json.Marshal(request{Cmd: "start", Address: urlToCheck})
	if err != nil {
		return Result{}, errors.Wrap(err, "Encode start request")
	}
	err = conn.WriteMessage(websocket.TextMessage, startReqBytes)
	if err != nil {
		return Result{}, errors.Wrap(err, "Send start request")
	}

	// Collect, calculate and return the result when context deadline exceeded
	totalResponsesCount := 0
	result := Result{}
	for {
		select {
		case err := <-readErrCh:
			// Return on error
			return Result{}, err
		case resp := <-readRespCh:
			// Collect results
			if resp.Total != nil {
				totalResponsesCount++
				result.Scrambled = resp.Total.Scrambled
				// Build sums to calculate averages later
				result.Bitrate += resp.Total.Bitrate
				result.CCErrors += resp.Total.CCErrors
				result.PESErrors += resp.Total.PESErrors
			}
			if resp.Streams != nil {
				if !result.HasAudio {
					result.HasAudio = lo.ContainsBy(resp.Streams, func(s stream) bool {
						return s.TypeName == "AUDIO"
					})
				}
				if !result.HasVideo {
					result.HasVideo = lo.ContainsBy(resp.Streams, func(s stream) bool {
						return s.TypeName == "VIDEO"
					})
				}
			}
		case <-ctx.Done():
			// Deadline exceeded
			// Close the connection by sending a close message
			closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
			err := conn.WriteMessage(websocket.CloseMessage, closeMsg)
			if err != nil {
				return Result{}, errors.Wrap(err, "Send close request")
			}
			// Wait (with timeout) for the server to close the connection
			select {
			case <-readErrCh:
			case <-time.After(time.Second):
			}
			// Calculate averages and return the result
			if totalResponsesCount != 0 {
				result.Bitrate = result.Bitrate / totalResponsesCount
				result.CCErrors = result.CCErrors / totalResponsesCount
				result.PESErrors = result.PESErrors / totalResponsesCount
			}
			return result, nil
		}
	}
}
