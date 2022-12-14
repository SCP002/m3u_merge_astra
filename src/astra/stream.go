package astra

import (
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"sync"

	"m3u_merge_astra/cfg"
	"m3u_merge_astra/deps"
	"m3u_merge_astra/util/conv"
	"m3u_merge_astra/util/copier"
	"m3u_merge_astra/util/network"
	"m3u_merge_astra/util/slice"

	"github.com/alitto/pond"
	"github.com/cockroachdb/errors"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/samber/lo"
	"github.com/schollz/progressbar/v3"
)

// Stream represents astra stream object
type Stream struct {
	DisabledInputs []string       `json:"_input,omitempty"`
	Enabled        bool           `json:"enable"`
	Groups         map[string]any `json:"groups,omitempty"`
	HTTPKeepActive string         `json:"http_keep_active,omitempty"`
	ID             string         `json:"id,omitempty"`
	Inputs         []string       `json:"input,omitempty"`
	Name           string         `json:"name,omitempty"`
	Type           string         `json:"type,omitempty"`
	Unknown        map[string]any `json:"-" jsonex:"true"` // All unknown fields go here.
}

// NewStream returns new stream with default config
func NewStream(cfg cfg.Streams, id, name, group string, inputs []string) Stream {
	return Stream{
		DisabledInputs: []string{},
		Enabled:        cfg.MakeNewEnabled,
		Groups:         lo.Ternary(cfg.AddGroupsToNew, map[string]any{cfg.GroupsCategoryForNew: group}, nil),
		// HTTPKeepActive: "5",
		ID:     id,
		Inputs: inputs,
		Name:   cfg.AddedPrefix + name,
		Type:   string(cfg.NewType),
	}
}

// GetName used to satisfy util/slice.Named interface
func (s Stream) GetName() string {
	return s.Name
}

// FirstGroup returns first "category: group" pair or empty string if not found
func (s Stream) FirstGroup() string {
	if len(s.Groups) > 0 {
		return fmt.Sprintf("%v: %v", lo.Keys(s.Groups)[0], lo.Values(s.Groups)[0])
	}
	return ""
}

// UpdateInput updates first encountered input if both it and <newURL> match the InputUpdateMap from config in <r>.
//
// If KeepInputHash is enabled in config, it also adds old input URL hash to <newURL>.
func (s Stream) UpdateInput(r deps.Global, newURL string) Stream {
	s = copier.PDeep(s)
	for inpIdx, oldURL := range s.Inputs {
		for _, updRec := range r.Cfg().Streams.InputUpdateMap {
			if updRec.From.MatchString(oldURL) && updRec.To.MatchString(newURL) {
				// Append old hash to new URL.
				if r.Cfg().Streams.KeepInputHash {
					oldHash, err := conv.GetHash(oldURL)
					if err != nil {
						r.Log().Debug(err)
					}
					newURL, _, err = conv.AddHash(oldHash, newURL)
					if err != nil {
						r.Log().Debug(err)
					}
				}
				if oldURL == newURL {
					continue
				}
				// Update first encountered matching input.
				r.TW().AppendRow(table.Row{s.Name, oldURL, newURL})
				s.Inputs[inpIdx] = newURL
				return s
			}
		}
	}
	return s
}

// HasInput returns true if stream inputs contain <tURLStr> parameter.
//
// If <withHash> is false, return true even if stream input and <tURLStr> are the same but have different hashes.
func (s Stream) HasInput(r deps.Global, tURLStr string, withHash bool) bool {
	return lo.ContainsBy(s.Inputs, func(cURLStr string) bool {
		equal, err := conv.LinksEqual(tURLStr, cURLStr, withHash)
		if err != nil {
			r.Log().Debug(err)
		}
		return equal
	})
}

// AddInput adds new <url> to stream inputs.
//
// If <print> is true, print added input.
func (s Stream) AddInput(r deps.Global, url string, print bool) Stream {
	if print {
		note := lo.Ternary(s.Enabled, "", "Stream is disabled")
		r.TW().AppendRow(table.Row{s.Name, s.FirstGroup(), url, note})
	}
	s.Inputs = slice.Prepend(s.Inputs, url)
	return s
}

// KnownInputs returns all inputs matching InputUpdateMap.From expression from config.
func (s Stream) KnownInputs(r deps.Global) []string {
	return lo.Filter(s.Inputs, func(inp string, _ int) bool {
		return lo.ContainsBy(r.Cfg().Streams.InputUpdateMap, func(updRec cfg.UpdateRecord) bool {
			return updRec.From.MatchString(inp)
		})
	})
}

// RemoveInputs removes all stream inputs equal <tInp>.
//
// If <print> is true, print removed inputs.
func (s Stream) RemoveInputs(r deps.Global, tInp string, print bool) Stream {
	rejectFn := func(cInp string, _ int) bool {
		if print && cInp == tInp {
			r.TW().AppendRow(table.Row{s.Name, s.FirstGroup(), tInp})
		}
		return cInp == tInp
	}
	s.Inputs = lo.Reject(s.Inputs, rejectFn)
	s.DisabledInputs = lo.Reject(s.DisabledInputs, rejectFn)
	return s
}

// Disable disables stream and adds name prefix if not already contain any
func (s Stream) disable(r deps.Global) Stream {
	newName := s.Name
	updated := false
	if !conv.ContainsAny(s.Name, r.Cfg().Streams.AddedPrefix, r.Cfg().Streams.DisabledPrefix) {
		newName = r.Cfg().Streams.DisabledPrefix + s.Name
		updated = true
	}
	if s.Enabled {
		s.Enabled = false
		updated = true
	}
	if updated {
		r.TW().AppendRow(table.Row{s.Name, s.FirstGroup()})
	}
	s.Name = newName
	return s
}

// enable enables stream and removes name prefix.
//
// If <onlyPrefixed> is true, enable stream only if it's name contains DisabledPrefix in config from <r>.
func (s Stream) enable(r deps.Global, onlyPrefixed bool) Stream {
	newName := s.Name
	updated := false
	if strings.Contains(s.Name, r.Cfg().Streams.DisabledPrefix) && r.Cfg().Streams.DisabledPrefix != "" {
		newName = strings.ReplaceAll(s.Name, r.Cfg().Streams.DisabledPrefix, "")
		updated = true
	} else if onlyPrefixed {
		return s
	}
	if !s.Enabled {
		s.Enabled = true
		updated = true
	}
	if updated {
		r.TW().AppendRow(table.Row{s.Name, newName, s.FirstGroup()})
	}
	s.Name = newName
	return s
}

// removeBlockedInputs removes blocked inputs from stream
func (s Stream) removeBlockedInputs(r deps.Global) Stream {
	rejectFn := func(input string, _ int) bool {
		reject := slice.RxAnyMatch(r.Cfg().Streams.InputBlacklist, input)
		if reject {
			r.TW().AppendRow(table.Row{s.Name, s.FirstGroup(), input})
		}
		return reject
	}

	s.Inputs = lo.Reject(s.Inputs, rejectFn)
	s.DisabledInputs = lo.Reject(s.DisabledInputs, rejectFn)
	return s
}

// hasNoInputs reurns true if stream has no inputs
func (s Stream) hasNoInputs() bool {
	return len(s.Inputs) == 0
}

// HasInput returns true if any input of <streams> contains <inp>.
//
// If <withHash> is false, return true even if stream input and <inp> are the same but have different hashes.
func (r repo) HasInput(streams []Stream, inp string, withHash bool) bool {
	return lo.ContainsBy(streams, func(s Stream) bool {
		return s.HasInput(r, inp, withHash)
	})
}

// Enable returns copy of <streams> with all prefixed streams enabled and renamed
func (r repo) Enable(streams []Stream) (out []Stream) {
	r.log.Info("Enabling and renaming prefixed streams\n")
	r.tw.AppendHeader(table.Row{"Old name", "New name", "Group"})

	for _, s := range streams {
		out = append(out, s.enable(r, true))
	}

	r.tw.Render()
	fmt.Fprint(os.Stderr, "\n")
	return
}

// RemoveBlockedInputs returns copy of <streams> without blocked inputs
func (r repo) RemoveBlockedInputs(streams []Stream) (out []Stream) {
	r.log.Info("Removing blocked inputs from streams\n")
	r.tw.AppendHeader(table.Row{"Name", "Group", "Input"})

	for _, s := range streams {
		out = append(out, s.removeBlockedInputs(r))
	}

	r.tw.Render()
	fmt.Fprint(os.Stderr, "\n")
	return
}

// RemoveDuplicatedInputs returns copy of <streams> with only unique inputs
func (r repo) RemoveDuplicatedInputs(streams []Stream) (out []Stream) {
	r.log.Info("Removing duplicated inputs from streams\n")
	r.tw.AppendHeader(table.Row{"Name", "Group", "Input"})

	// inputsMap is used to check if input is the first one encountered in the list. Value of the map is not used.
	inputsMap := map[string]bool{}

	for _, s := range streams {
		for _, inp := range s.Inputs {
			if _, duplicate := inputsMap[inp]; duplicate {
				r.tw.AppendRow(table.Row{s.Name, s.FirstGroup(), inp})
				s.Inputs = slice.RemoveLast(s.Inputs, inp)
			} else {
				inputsMap[inp] = true
			}
		}
		out = append(out, s)
	}

	r.tw.Render()
	fmt.Fprint(os.Stderr, "\n")
	return
}

// UniteInputs returns copy of <streams> with inputs of every equally named stream moved to the first stream found
func (r repo) UniteInputs(streams []Stream) (out []Stream) {
	r.log.Info("Uniting inputs of streams\n")
	r.tw.AppendHeader(table.Row{"From ID", "From name", "Input", "To ID", "To name", "Note"})

	out = copier.PDeep(streams)
	for currIdx, currStream := range out {
		note := lo.Ternary(currStream.Enabled, "", "Stream is disabled")
		slice.EverySimilar(r.cfg.General, out, currStream.Name, currIdx+1, func(nextStream Stream, nextIdx int) {
			for _, nextInput := range nextStream.Inputs {
				r.tw.AppendRow(table.Row{nextStream.ID, nextStream.Name, nextInput, currStream.ID, currStream.Name,
					note})
				if !currStream.HasInput(r, nextInput, true) {
					currStream = currStream.AddInput(r, nextInput, false)
					out[currIdx] = currStream
				}
				nextStream = nextStream.RemoveInputs(r, nextInput, false)
				out[nextIdx] = nextStream
			}
		})
	}

	r.tw.Render()
	fmt.Fprint(os.Stderr, "\n")
	return
}

// SortInputs returns copy of <streams> with all inputs sorted by InputWeightToTypeMap in config
func (r repo) SortInputs(streams []Stream) (out []Stream) {
	r.log.Info("Sorting inputs of streams\n")

	out = copier.PDeep(streams)
	for _, s := range out {
		sort.SliceStable(s.Inputs, func(i, j int) bool {
			// Set default weight
			leftInpWeight := r.cfg.Streams.UnknownInputWeight
			rightInpWeight := r.cfg.Streams.UnknownInputWeight

			for weight, rx := range r.cfg.Streams.InputWeightToTypeMap {
				// Assign weight from map if match found
				leftInpWeight = lo.Ternary(rx.MatchString(s.Inputs[i]), weight, leftInpWeight)
				rightInpWeight = lo.Ternary(rx.MatchString(s.Inputs[j]), weight, rightInpWeight)
			}

			return leftInpWeight < rightInpWeight
		})
	}

	return
}

// RemoveDeadInputs returns copy of <streams> without inputs which do not respond in time or respond with status code
// >= 400.
//
// If <bar> is true, display progress bar.
//
// Not checking Content-Type header as server can return text/html but stream still will be playable.
//
// Not checking response body as some streams can periodically respond with no content but still be playable.
//
// Currently supports only HTTP(S).
func (r repo) RemoveDeadInputs(httpClient *http.Client, streams []Stream, bar bool) (out []Stream) {
	r.log.Info("Removing dead inputs from streams\n")
	r.tw.AppendHeader(table.Row{"Name", "Group", "Input", "Reason"})

	var progBar *progressbar.ProgressBar
	if bar {
		progBar = progressbar.Default(int64(getInputsAmount(streams)), "Done:")
	}

	canCheck := func(inp string) bool {
		if slice.RxAnyMatch(r.cfg.Streams.DeadInputsCheckBlacklist, inp) {
			return false
		}
		if strings.HasPrefix(inp, "http://") || strings.HasPrefix(inp, "https://") {
			return true
		}
		return false
	}

	getReason := func(inp string) string {
		resp, err := httpClient.Get(inp)
		if err == nil {
			defer resp.Body.Close()
		}
		reason := ""
		if err != nil {
			errType := network.GetErrType(err)
			reason = lo.Ternary(errType == network.Unknown, err.Error(), string(errType))
		} else if resp.StatusCode >= 400 {
			reason = fmt.Sprintf("Responded with: %v", resp.Status)
		}
		return reason
	}

	pool := pond.New(r.cfg.Streams.InputMaxConns, 0, pond.MinWorkers(0))
	var mut sync.Mutex

	out = copier.PDeep(streams)
	for sIdx, s := range out {
		for _, inp := range s.Inputs {
			s, sIdx, inp := s, sIdx, inp
			pool.Submit(func() {
				r.log.Debugf("Start task sIdx %v, inp %v", sIdx, inp)
				if canCheck(inp) {
					reason := getReason(inp)
					if reason != "" {
						mut.Lock()
						r.tw.AppendRow(table.Row{s.Name, s.FirstGroup(), inp, reason})
						out[sIdx].Inputs = slice.RemoveLast(out[sIdx].Inputs, inp)
						mut.Unlock()
					}
				}
				if bar {
					err := progBar.Add(1)
					if err != nil {
						r.log.Debugf("Unable to increase: %v", errors.Wrap(err, "progress bar"))
					}
				}
			})
		}
	}

	pool.StopAndWait()

	r.tw.Render()
	fmt.Fprint(os.Stderr, "\n")
	return
}

// AddHashes returns copy of <streams> with hashes added to every input as defined in config with *ToInputHashMap
func (r repo) AddHashes(streams []Stream) (out []Stream) {
	r.log.Info("Adding hashes to inputs of streams\n")
	r.tw.AppendHeader(table.Row{"Name", "Group", "Hash", "Result"})

	out = copier.PDeep(streams)
	var changed bool
	for sIdx, s := range out {
		for inpIdx, inp := range s.Inputs {
			// By inputs
			for _, rule := range r.cfg.Streams.InputToInputHashMap {
				if rule.By.MatchString(inp) {
					var err error
					inp, changed, err = conv.AddHash(rule.Hash, inp)
					if err != nil {
						r.log.Debug(err)
					}
					if changed {
						r.tw.AppendRow(table.Row{s.Name, s.FirstGroup(), rule.Hash, inp})
					}
				}
			}
			// By name
			for _, rule := range r.cfg.Streams.NameToInputHashMap {
				if rule.By.MatchString(s.Name) {
					var err error
					inp, changed, err = conv.AddHash(rule.Hash, inp)
					if err != nil {
						r.log.Debug(err)
					}
					if changed {
						r.tw.AppendRow(table.Row{s.Name, s.FirstGroup(), rule.Hash, inp})
					}
				}
			}
			// By group
			for _, rule := range r.cfg.Streams.GroupToInputHashMap {
				if rule.By.MatchString(s.FirstGroup()) {
					var err error
					inp, changed, err = conv.AddHash(rule.Hash, inp)
					if err != nil {
						r.log.Debug(err)
					}
					if changed {
						r.tw.AppendRow(table.Row{s.Name, s.FirstGroup(), rule.Hash, inp})
					}
				}
			}
			out[sIdx].Inputs[inpIdx] = inp
		}
	}

	r.tw.Render()
	fmt.Fprint(os.Stderr, "\n")
	return
}

// RemoveWithoutInputs returns copy of <streams> without streams which have no inputs
func (r repo) RemoveWithoutInputs(streams []Stream) (out []Stream) {
	r.log.Info("Removing streams without inputs\n")
	r.tw.AppendHeader(table.Row{"Name", "Group"})

	out = lo.Reject(streams, func(s Stream, _ int) bool {
		if s.hasNoInputs() {
			r.tw.AppendRow(table.Row{s.Name, s.FirstGroup()})
		}
		return s.hasNoInputs()
	})

	r.tw.Render()
	fmt.Fprint(os.Stderr, "\n")
	return
}

// DisableWithoutInputs returns copy of <streams> with all streams disabled if they have no inputs
func (r repo) DisableWithoutInputs(streams []Stream) (out []Stream) {
	r.log.Info("Disabling streams without inputs\n")
	r.tw.AppendHeader(table.Row{"Name", "Group"})

	for _, s := range streams {
		if s.hasNoInputs() {
			s = s.disable(r)
		}
		out = append(out, s)
	}

	r.tw.Render()
	fmt.Fprint(os.Stderr, "\n")
	return
}

// getInputsAmount returns total amount of inputs in <streams>
func getInputsAmount(streams []Stream) int {
	return lo.SumBy(streams, func(s Stream) int {
		return len(s.Inputs)
	})
}
