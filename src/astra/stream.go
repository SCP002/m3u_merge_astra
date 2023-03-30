package astra

import (
	"fmt"
	"net/http"
	"sort"
	"strings"
	"sync"

	"m3u_merge_astra/cfg"
	"m3u_merge_astra/deps"
	"m3u_merge_astra/util/copier"
	"m3u_merge_astra/util/logger"
	"m3u_merge_astra/util/network"
	"m3u_merge_astra/util/slice"
	"m3u_merge_astra/util/slice/find"
	urlUtil "m3u_merge_astra/util/url"

	"github.com/alitto/pond"
	"github.com/samber/lo"
)

// Stream represents astra stream object
type Stream struct {
	DisabledInputs []string          `json:"_input,omitempty"`
	Enabled        bool              `json:"enable"`
	Groups         map[string]string `json:"groups,omitempty"`
	HTTPKeepActive string            `json:"http_keep_active,omitempty"`
	ID             string            `json:"id,omitempty"`
	Inputs         []string          `json:"input,omitempty"`
	Name           string            `json:"name,omitempty"`
	Type           string            `json:"type,omitempty"`
	Unknown        map[string]any    `json:"-" jsonex:"true"` // All unknown fields go here.
	MarkAdded      bool              `json:"-"`               // Set added name prefix after processing?
	MarkDisabled   bool              `json:"-"`               // Set disabled name prefix after processing?
}

// NewStream returns new stream with default config
func NewStream(cfg cfg.Streams, id, name, group string, inputs []string) Stream {
	var groups map[string]string = nil
	if cfg.AddGroupsToNew {
		groups = map[string]string{cfg.GroupsCategoryForNew: group}
	}

	return Stream{
		DisabledInputs: []string{},
		Enabled:        cfg.MakeNewEnabled,
		Groups:         groups,
		ID:             id,
		Inputs:         inputs,
		Name:           name,
		Type:           string(cfg.NewType),
		MarkAdded:      true,
	}
}

// GetName used to satisfy util/slice.Named interface
func (s Stream) GetName() string {
	return s.Name
}

// FirstGroup returns alphabetically first "category: group" pair or empty string if groups are empty
func (s Stream) FirstGroup() string {
	if len(s.Groups) > 0 {
		entries := lo.Entries(s.Groups)
		sort.SliceStable(entries, func(i, j int) bool {
			return entries[i].Key < entries[j].Key
		})
		return fmt.Sprintf("%v: %v", entries[0].Key, entries[0].Value)
	}
	return ""
}

// UpdateInput updates first encountered input if both it and <newURL> match the InputUpdateMap from config in <r>.
//
// Runs <callback> with old URL for every updated input.
//
// If KeepInputHash is enabled in config, it also adds old input URL hash to <newURL>.
func (s Stream) UpdateInput(r deps.Global, newURL string, callback func(string)) Stream {
	s = copier.MustDeep(s)
	cfg := r.Cfg().Streams

	for inpIdx, oldURL := range s.Inputs {
		for _, updRec := range cfg.InputUpdateMap {
			if updRec.From.MatchString(oldURL) && updRec.To.MatchString(newURL) {
				// Append old hash to new URL.
				if cfg.KeepInputHash {
					oldHash, err := urlUtil.GetHash(oldURL)
					if err != nil {
						r.Log().Debug(err)
					}
					newURL, _, err = urlUtil.AddHash(oldHash, newURL)
					if err != nil {
						r.Log().Debug(err)
					}
				}
				if oldURL == newURL {
					continue
				}
				// Update first encountered matching input.
				callback(oldURL)
				s.Inputs[inpIdx] = newURL
				return s
			}
		}
	}
	return s
}

// HasInput returns true if stream inputs contain <tURLStr> parameter.
//
// If <withHash> is false, ignore hashes (everything after #) during the search.
func (s Stream) HasInput(log *logger.Logger, tURLStr string, withHash bool) bool {
	return lo.ContainsBy(s.Inputs, func(cURLStr string) bool {
		equal, err := urlUtil.Equal(tURLStr, cURLStr, withHash)
		if err != nil {
			log.Debug(err)
		}
		return equal
	})
}

// AddInput adds new <url> to stream inputs
func (s Stream) AddInput(url string) Stream {
	s.Inputs = slice.Prepend(s.Inputs, url)
	return s
}

// KnownInputs returns all inputs matching InputUpdateMap.From expression from <config>.
func (s Stream) KnownInputs(config cfg.Streams) []string {
	return lo.Filter(s.Inputs, func(inp string, _ int) bool {
		return lo.ContainsBy(config.InputUpdateMap, func(updRec cfg.UpdateRecord) bool {
			return updRec.From.MatchString(inp)
		})
	})
}

// InputsUpdateNote returns note is stream is disabled or if it will be enabled on inputs update
func (s Stream) InputsUpdateNote(cfg cfg.Streams) string {
	if !s.Enabled {
		return lo.Ternary(cfg.EnableOnInputUpdate, "Enabling the stream", "Stream is disabled")
	}
	return ""
}

// Enable enables the stream
func (s Stream) Enable() Stream {
	s.Enabled = true
	return s
}

// RemoveInputs removes all stream inputs equal <tInp>, running <callback> for every input removed
func (s Stream) RemoveInputsCb(tInp string, callback func()) Stream {
	rejectFn := func(cInp string, _ int) bool {
		if cInp == tInp {
			callback()
		}
		return cInp == tInp
	}
	s.Inputs = lo.Reject(s.Inputs, rejectFn)
	s.DisabledInputs = lo.Reject(s.DisabledInputs, rejectFn)
	return s
}

// removeInputs is the same as RemoveInputsCb but without callback
func (s Stream) removeInputs(tInp string) Stream {
	return s.RemoveInputsCb(tInp, func() {})
}

// Disable disables stream and sets MarkDisabled field
func (s Stream) disable() Stream {
	s.Enabled = false
	s.MarkDisabled = true
	return s
}

// removeDuplicatedInputsByRx returns stream with only unique inputs by first capture groups of regular expressions
// defined in config in <r>.
//
// Runs <callback> for every removed input.
func (s Stream) removeDuplicatedInputsByRx(r repo, callback func(string)) Stream {
	cfg := r.cfg.Streams

	// inputsCGMap is used to check if first capture group of input is the first one encountered in the list.
	// Value of the map is not used.
	inputsCGMap := map[string]bool{}

	for _, rx := range cfg.RemoveDuplicatedInputsByRxList {
		for _, inp := range s.Inputs {
			matchList := rx.FindStringSubmatch(inp)
			if len(matchList) < 2 {
				r.log.DebugCFi("Found no matches", "regexp", rx.String(), "for input", inp)
				continue
			}
			captureGroup := matchList[1]
			if _, duplicate := inputsCGMap[captureGroup]; duplicate {
				callback(inp)
				s.Inputs = slice.RemoveLast(s.Inputs, inp)
			} else {
				inputsCGMap[captureGroup] = true
			}
		}
	}

	return s
}

// removeBlockedInputs removes blocked inputs from stream, running <callback> for every removed input
func (s Stream) removeBlockedInputs(cfg cfg.Streams, callback func(string)) Stream {
	rejectFn := func(input string, _ int) bool {
		reject := slice.AnyRxMatch(cfg.InputBlacklist, input)
		if reject {
			callback(input)
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

// hasPrefix returns true if name of the stream has <prefix>
func (s Stream) hasPrefix(prefix string) bool {
	return prefix != "" && strings.HasPrefix(s.Name, prefix)
}

// setPrefix returns stream named starting with <prefix>
func (s Stream) setPrefix(prefix string) Stream {
	s.Name = prefix + s.Name
	return s
}

// removePrefix returns stream named starting without <prefix>
func (s Stream) removePrefix(prefix string) Stream {
	s.Name = strings.TrimPrefix(s.Name, prefix)
	return s
}

// HasInput returns true if any input of <streams> contains <inp>.
//
// If <withHash> is false, ignore hashes (everything after #) during the search.
func (r repo) HasInput(streams []Stream, inp string, withHash bool) bool {
	return lo.ContainsBy(streams, func(s Stream) bool {
		return s.HasInput(r.log, inp, withHash)
	})
}

// RemoveNamePrefixes returns shallow copy of <streams> without name prefixes on every stream and MarkAdded or
// MarkDisabled fields set instead
func (r repo) RemoveNamePrefixes(streams []Stream) (out []Stream) {
	for _, s := range streams {
		oldName := s.Name
		for i := 0; i < 2; i++ { // Run twice to remove in any order
			if s.hasPrefix(r.cfg.Streams.AddedPrefix) {
				s = s.removePrefix(r.cfg.Streams.AddedPrefix)
				s.MarkAdded = true
			}
			if s.hasPrefix(r.cfg.Streams.DisabledPrefix) {
				s = s.removePrefix(r.cfg.Streams.DisabledPrefix)
				s.MarkDisabled = true
			}
		}
		if oldName != s.Name {
			r.log.InfoCFi("Temporarily removing name prefix from stream", "old name", oldName, "new name", s.Name,
				"group", s.FirstGroup())
		}
		out = append(out, s)
	}

	return
}

// Sort returns deep copy of <streams> sorted by name
func (r repo) Sort(streams []Stream) (out []Stream) {
	r.log.Info("Sorting astra streams")

	out = slice.Sort(streams)

	return
}

// RemoveBlockedInputs returns shallow copy of <streams> without blocked inputs
func (r repo) RemoveBlockedInputs(streams []Stream) (out []Stream) {
	for _, s := range streams {
		out = append(out, s.removeBlockedInputs(r.cfg.Streams, func(input string) {
			r.log.InfoCFi("Removing blocked input from stream", "name", s.Name, "group", s.FirstGroup(), "input", input)
		}))
	}

	return
}

// RemoveDuplicatedInputs returns shallow copy of <streams> with only unique inputs
func (r repo) RemoveDuplicatedInputs(streams []Stream) (out []Stream) {
	// inputsMap is used to check if input is the first one encountered in the list. Value of the map is not used.
	inputsMap := map[string]bool{}

	for _, s := range streams {
		for _, inp := range s.Inputs {
			if _, duplicate := inputsMap[inp]; duplicate {
				r.log.InfoCFi("Removing duplicated input from stream", "name", s.Name, "group", s.FirstGroup(),
					"input", inp)
				s.Inputs = slice.RemoveLast(s.Inputs, inp)
			} else {
				inputsMap[inp] = true
			}
		}
		out = append(out, s)
	}

	return
}

// RemoveDuplicatedInputsByRx returns shallow copy of <streams> with only unique inputs per stream by first capture
// groups of regular expressions defined in config.
func (r repo) RemoveDuplicatedInputsByRx(streams []Stream) (out []Stream) {
	for _, s := range streams {
		out = append(out, s.removeDuplicatedInputsByRx(r, func(input string) {
			r.log.InfoCFi("Removing duplicated input per stream by regular expressions", "name", s.Name,
				"group", s.FirstGroup(), "input", input)
		}))
	}

	return
}

// UniteInputs returns deep copy of <streams> with inputs of every equally named stream moved to the first stream found.
//
// If cfg.Streams.EnableOnInputUpdate is enabled in config, it also enables every stream with new inputs.
func (r repo) UniteInputs(streams []Stream) (out []Stream) {
	out = copier.MustDeep(streams)
	for currIdx, currStream := range out {
		find.EverySimilar(r.cfg.General, out, currStream.Name, currIdx + 1, func(nextStream Stream, nextIdx int) {
			for _, nextInput := range nextStream.Inputs {
				r.log.InfoCFi("Uniting inputs of streams", "from ID", nextStream.ID, "from name", nextStream.Name,
					"input", nextInput, "to ID", currStream.ID, "to name", currStream.Name,
					"note", currStream.InputsUpdateNote(r.cfg.Streams))
				if !currStream.HasInput(r.log, nextInput, true) {
					currStream = currStream.AddInput(nextInput)
					if r.cfg.Streams.EnableOnInputUpdate {
						r.log.DebugCFi("enable_on_input_update is on, enabling the stream", "ID", currStream.ID,
							"name", currStream.Name)
						currStream = currStream.Enable()
					}
					out[currIdx] = currStream
				}
				nextStream = nextStream.removeInputs(nextInput)
				out[nextIdx] = nextStream
			}
		})
	}

	return
}

// SortInputs returns deep copy of <streams> with all inputs sorted by InputWeightToTypeMap in config
func (r repo) SortInputs(streams []Stream) (out []Stream) {
	r.log.Info("Sorting inputs of streams")

	out = copier.MustDeep(streams)
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

// RemoveDeadInputs returns deep copy of <streams> without inputs which do not respond in time or respond with status
// code >= 400.
//
// Not checking Content-Type header as server can return text/html but stream still will be playable.
//
// Not checking response body as some streams can periodically respond with no content but still be playable.
//
// Currently supports only HTTP(S).
func (r repo) RemoveDeadInputs(httpClient *http.Client, streams []Stream) (out []Stream) {
	// canCheck returns true if <inp> can be checked
	canCheck := func(inp string) bool {
		if slice.AnyRxMatch(r.cfg.Streams.DeadInputsCheckBlacklist, inp) {
			return false
		}
		if strings.HasPrefix(inp, "http://") || strings.HasPrefix(inp, "https://") {
			return true
		}
		return false
	}

	// getReason returns reason why <inp> should be removed
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
	inputsAmount := getInputsAmount(streams)
	inputsDone := 0

	// getProgress returns formatted progress of inputs processed
	getProgress := func() string {
		mut.Lock()
		percent := (inputsDone * 100) / inputsAmount
		progress := fmt.Sprintf("%v / %v (%v%%)", inputsDone, inputsAmount, percent)
		mut.Unlock()
		return progress
	}

	out = copier.MustDeep(streams)
	for sIdx, s := range out {
		for _, inp := range s.Inputs {
			s, sIdx, inp := s, sIdx, inp
			pool.Submit(func() {
				r.log.DebugCFi("Start checking input", "stream name", s.Name, "stream index", sIdx, "input", inp,
					"progress", getProgress())
				if canCheck(inp) {
					reason := getReason(inp)
					if reason != "" {
						r.log.InfoCFi("Removing dead input from stream", "name", s.Name, "group", s.FirstGroup(),
							"input", inp, "reason", reason)
						mut.Lock()
						out[sIdx].Inputs = slice.RemoveLast(out[sIdx].Inputs, inp)
						mut.Unlock()
					}
				}

				mut.Lock()
				inputsDone++
				mut.Unlock()
				r.log.DebugCFi("End checking input", "stream name", s.Name, "stream index", sIdx, "input", inp,
					"progress", getProgress())
			})
		}
	}

	pool.StopAndWait()

	return
}

// AddHashes returns deep copy of <streams> with hashes added to every input as defined in config with *ToInputHashMap
func (r repo) AddHashes(streams []Stream) (out []Stream) {
	out = copier.MustDeep(streams)
	var changed bool

	for sIdx, s := range out {
		for inpIdx, inp := range s.Inputs {
			// By inputs
			for _, rule := range r.cfg.Streams.InputToInputHashMap {
				if rule.By.MatchString(inp) {
					var err error
					inp, changed, err = urlUtil.AddHash(rule.Hash, inp)
					if err != nil {
						r.log.Debug(err)
					}
					if changed {
						r.log.InfoCFi("Adding hash to input of stream", "name", s.Name, "group", s.FirstGroup(),
							"hash", rule.Hash, "result", inp)
					}
				}
			}
			// By name
			for _, rule := range r.cfg.Streams.NameToInputHashMap {
				if rule.By.MatchString(s.Name) {
					var err error
					inp, changed, err = urlUtil.AddHash(rule.Hash, inp)
					if err != nil {
						r.log.Debug(err)
					}
					if changed {
						r.log.InfoCFi("Adding hash to input of stream", "name", s.Name, "group", s.FirstGroup(),
							"hash", rule.Hash, "result", inp)
					}
				}
			}
			// By group
			for _, rule := range r.cfg.Streams.GroupToInputHashMap {
				if rule.By.MatchString(s.FirstGroup()) {
					var err error
					inp, changed, err = urlUtil.AddHash(rule.Hash, inp)
					if err != nil {
						r.log.Debug(err)
					}
					if changed {
						r.log.InfoCFi("Adding hash to input of stream", "name", s.Name, "group", s.FirstGroup(),
							"hash", rule.Hash, "result", inp)
					}
				}
			}
			out[sIdx].Inputs[inpIdx] = inp
		}
	}

	return
}

// RemoveWithoutInputs returns shallow copy of <streams> without streams which have no inputs
func (r repo) RemoveWithoutInputs(streams []Stream) (out []Stream) {
	out = lo.Reject(streams, func(s Stream, _ int) bool {
		if s.hasNoInputs() {
			r.log.InfoCFi("Removing stream without inputs", "name", s.Name, "group", s.FirstGroup())
		}
		return s.hasNoInputs()
	})

	return
}

// DisableWithoutInputs returns shallow copy of <streams> with all streams disabled if they have no inputs
func (r repo) DisableWithoutInputs(streams []Stream) (out []Stream) {
	for _, s := range streams {
		if s.Enabled && s.hasNoInputs() {
			r.log.InfoCFi("Disabling stream without inputs", "name", s.Name, "group", s.FirstGroup())
			s = s.disable()
		}
		out = append(out, s)
	}

	return
}

// RemoveNamePrefixes returns shallow copy of <streams> with name prefixes on every stream if MarkAdded or MarkDisabled
// is true.
func (r repo) AddNamePrefixes(streams []Stream) (out []Stream) {
	for _, s := range streams {
		oldName := s.Name
		if s.MarkAdded {
			s = s.setPrefix(r.cfg.Streams.AddedPrefix)
		}
		if s.MarkDisabled {
			s = s.setPrefix(r.cfg.Streams.DisabledPrefix)
		}
		if oldName != s.Name {
			r.log.InfoCFi("Adding name prefix to stream", "old name", oldName, "new name", s.Name,
				"group", s.FirstGroup())
		}
		out = append(out, s)
	}

	return
}

// getInputsAmount returns total amount of inputs in <streams>
func getInputsAmount(streams []Stream) int {
	return lo.SumBy(streams, func(s Stream) int {
		return len(s.Inputs)
	})
}
