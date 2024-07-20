package analyzer

import (
	"context"
	"net/http"
	"net/url"
	"time"

	"m3u_merge_astra/util/iter"
	"m3u_merge_astra/util/logger"

	json "github.com/SCP002/jsonexraw"
	"github.com/cockroachdb/errors"
	"github.com/gorilla/websocket"
	"github.com/samber/lo"
)

// startReq represets start request to analyzer
type startReq struct {
	Cmd     string `json:"cmd"`
	Address string `json:"address"`
}

// startResp represents response from analyzer start command.
//
// Pointers are to distinguish between undefined and zero value.
type startResp struct {
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
	PCRErrors int
	PESErrors int
	Scrambled bool
	HasAudio  bool
	HasVideo  bool
}

// stopReq represets stop request to analyzer
type stopReq struct {
	Cmd string `json:"cmd"`
}

// Analyzer represents astra analyzer client interface
type Analyzer interface {
	Check(watchTime time.Duration, maxAttempts int, urlToCheck string) (Result, error)
}

// analyzer represents astra analyzer client
type analyzer struct {
	url    string
	dialer *websocket.Dialer
	log    *logger.Logger
}

// New returns new configured astra analyzer client which connects to <address> in format of 'host:port' with
// <handshakeTimeout>.
func New(log *logger.Logger, address string, handshakeTimeout time.Duration) *analyzer {
	url := url.URL{Scheme: "ws", Host: address, Path: "/api/"}

	return &analyzer{
		url: url.String(),
		dialer: &websocket.Dialer{
			Proxy:            http.ProxyFromEnvironment,
			HandshakeTimeout: handshakeTimeout,
		},
		log: log,
	}
}

// Check returns check result of <urlToCheck> using astra analyzer.
//
// Returns when <watchTime> is up, up to <maxAttempts> times or earlier if average bitrate was > 0 during previous
// attempts.
//
// Does Not return error if <urlToCheck> is dead or invalid, rely on bitrate == 0.
func (a analyzer) Check(watchTime time.Duration, maxAttempts int, urlToCheck string) (Result, error) {
	// Does the same as it's parent but without retry logic and returns when <ctx> is done
	check := func(ctx context.Context, urlToCheck string) (Result, error) {
		conn, _, err := a.dialer.Dial(a.url, nil)
		if err != nil {
			return Result{}, errors.Wrap(err, "Dial")
		}
		defer conn.Close()

		// Read responses
		readErrCh := make(chan error)
		readRespCh := make(chan startResp)

		go func() {
			defer close(readErrCh)
			defer close(readRespCh)
			for {
				_, respBytes, err := conn.ReadMessage()
				if err != nil {
					readErrCh <- errors.Wrap(err, "Read response")
					return
				}
				var resp startResp
				err = json.Unmarshal(respBytes, &resp)
				if err != nil {
					readErrCh <- errors.Wrap(err, "Decode response")
					return
				}
				readRespCh <- resp
			}
		}()

		// Send start request
		startReqBytes, err := json.Marshal(startReq{Cmd: "start", Address: urlToCheck})
		if err != nil {
			return Result{}, errors.Wrap(err, "Encode start request")
		}
		err = conn.WriteMessage(websocket.TextMessage, startReqBytes)
		if err != nil {
			return Result{}, errors.Wrap(err, "Send start request")
		}

		// Collect, calculate and return the result when context deadline exceeded
		totalResponsesCount := 0
		var result Result
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
					result.PCRErrors += resp.Total.PCRErrors
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
				// Send stop request
				stopReqBytes, err := json.Marshal(stopReq{Cmd: "stop"})
				if err != nil {
					return Result{}, errors.Wrap(err, "Encode stop request")
				}
				err = conn.WriteMessage(websocket.TextMessage, stopReqBytes)
				if err != nil {
					return Result{}, errors.Wrap(err, "Send stop request")
				}
				// Send close message
				closeMsg := websocket.FormatCloseMessage(websocket.CloseNormalClosure, "")
				err = conn.WriteMessage(websocket.CloseMessage, closeMsg)
				if err != nil {
					return Result{}, errors.Wrap(err, "Send close message")
				}
				// Wait (with timeout) for the server to close the connection (deferred close)
				select {
				case <-readErrCh:
				case <-time.After(time.Second):
				}
				// Calculate averages and return the result
				if totalResponsesCount != 0 {
					result.Bitrate = result.Bitrate / totalResponsesCount
					result.CCErrors = result.CCErrors / totalResponsesCount
					result.PCRErrors = result.PCRErrors / totalResponsesCount
					result.PESErrors = result.PESErrors / totalResponsesCount
				}
				return result, nil
			}
		}
	}

	// Make attempts
	var result Result
	var err error
	iter.Times(maxAttempts, func(attempt int) bool {
		a.log.DebugCFi("Analyzing", "url", urlToCheck, "attempt", attempt)
		ctx, cancel := context.WithTimeout(context.Background(), watchTime)
		defer cancel()
		result, err = check(ctx, urlToCheck)
		if result.Bitrate > 0 || err != nil {
			return false // Stop trying
		}
		return true
	})

	return result, err
}

// fakeAnalyzer represents fake astra analyzer client
type fakeAnalyzer struct {
	urlResultMap map[string]Result
}

// NewFake returns new fake astra analyzer client
func NewFake() *fakeAnalyzer {
	return &fakeAnalyzer{
		urlResultMap: map[string]Result{},
	}
}

// AddResult adds new <result> to return when checking <url>
func (a fakeAnalyzer) AddResult(url string, result Result) {
	a.urlResultMap[url] = result
}

// Check returns fake result for <urlToCheck> and nil error
func (a fakeAnalyzer) Check(watchTime time.Duration, maxAttempts int, urlToCheck string) (Result, error) {
	return a.urlResultMap[urlToCheck], nil
}
