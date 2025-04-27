package astra

import (
	"fmt"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"m3u_merge_astra/astra/analyzer"
	"m3u_merge_astra/cfg"
	"m3u_merge_astra/deps"
	"m3u_merge_astra/util/copier"
	"m3u_merge_astra/util/iter"
	"m3u_merge_astra/util/logger"
	"m3u_merge_astra/util/network"
	"m3u_merge_astra/util/slice"
	"m3u_merge_astra/util/slice/find"
	urlUtil "m3u_merge_astra/util/url"

	"github.com/alitto/pond"
	"github.com/go-co-op/gocron"
	"github.com/google/go-cmp/cmp"
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
	Remove         bool              `json:"remove,omitempty"` // Used by API to remove stream.
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
		HTTPKeepActive: strconv.Itoa(cfg.NewKeepActive),
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
	if len(s.Groups) == 0 {
		return ""
	}
	entries := lo.Entries(s.Groups)
	sort.SliceStable(entries, func(i, j int) bool {
		return entries[i].Key < entries[j].Key
	})
	return fmt.Sprintf("%v: %v", entries[0].Key, entries[0].Value)
}

// UpdateInput updates first encountered input if both it and <newURL> match the InputUpdateMap from config in <r>.
//
// Returns deep copy of stream.
//
// Returns true as the second return value if stream was updated.
//
// Runs <callback> with old URL for every updated input.
//
// If KeepInputHash is enabled in config, it also adds old input URL hash to <newURL>.
func (s Stream) UpdateInput(r deps.Global, newURL string, callback func(string)) (Stream, bool) {
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
				return s, true
			}
		}
	}
	return s, false
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

// InputsUpdateNote returns note if stream is disabled and enabling on inputs update is off
func (s Stream) InputsUpdateNote(cfg cfg.Streams) string {
	return lo.Ternary(!s.Enabled && !cfg.EnableOnInputUpdate, "Stream is disabled", "")
}

// Enable enables the stream and sets MarkDisabled field to false
func (s Stream) Enable() Stream {
	s.Enabled = true
	s.MarkDisabled = false
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

// Disable disables stream and sets MarkDisabled field to true
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
				r.log.DebugFi("Found no matches", "regexp", rx.String(), "for input", inp)
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

// disableAllButOneInputByRx returns stream with all inputs disabled except the input which matches any regular
// expression defined in config.
//
// Runs <callback> for every disabled input.
func (s Stream) disableAllButOneInputByRx(cfg cfg.Streams, callback func(string)) Stream {
	for _, rx := range cfg.DisableAllButOneInputByRxList {
		for _, inp := range s.Inputs {
			if rx.MatchString(inp) {
				inputsToDisable := slice.RemoveFirst(s.Inputs, inp)
				iter.ForEach(inputsToDisable, callback)
				s.DisabledInputs = append(s.DisabledInputs, inputsToDisable...)
				s.Inputs = []string{inp}
				return s
			}
		}
	}

	return s
}

// removeDisabledInputs returns stream with all disabled inputs removed.
//
// Runs <callback> for every removed input.
func (s Stream) removeDisabledInputs(callback func(string)) Stream {
	for _, inp := range s.DisabledInputs {
		callback(inp)
	}
	s.DisabledInputs = []string{}
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
	r.log.Info("Temporarily removing name prefixes from streams")

	for _, s := range streams {
		oldName := s.Name
		for s.hasPrefix(r.cfg.Streams.AddedPrefix) || s.hasPrefix(r.cfg.Streams.DisabledPrefix) {
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
			r.log.InfoFi("Temporarily removing name prefix from stream", "ID", s.ID, "old name", oldName,
				"new name", s.Name, "group", s.FirstGroup())
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
	r.log.Info("Removing blocked inputs from streams")

	for _, s := range streams {
		out = append(out, s.removeBlockedInputs(r.cfg.Streams, func(input string) {
			r.log.InfoFi("Removing blocked input from stream", "ID", s.ID, "name", s.Name, "group", s.FirstGroup(),
				"input", input)
		}))
	}

	return
}

// RemoveDuplicatedInputs returns shallow copy of <streams> with only unique inputs
func (r repo) RemoveDuplicatedInputs(streams []Stream) (out []Stream) {
	r.log.Info("Removing duplicated inputs from streams")

	// inputsMap is used to check if input is the first one encountered in the list. Value of the map is not used.
	inputsMap := map[string]bool{}

	for _, s := range streams {
		for _, inp := range s.Inputs {
			if _, duplicate := inputsMap[inp]; duplicate {
				r.log.InfoFi("Removing duplicated input from stream", "ID", s.ID, "name", s.Name,
					"group", s.FirstGroup(), "input", inp)
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
	r.log.Info("Removing duplicated inputs per stream by regular expressions")

	for _, s := range streams {
		out = append(out, s.removeDuplicatedInputsByRx(r, func(input string) {
			r.log.InfoFi("Removing duplicated input per stream by regular expressions", "ID", s.ID, "name", s.Name,
				"group", s.FirstGroup(), "input", input)
		}))
	}

	return
}

// UniteInputs returns deep copy of <streams> with inputs of every equally named stream moved to the first stream found.
//
// If cfg.Streams.EnableOnInputUpdate is enabled in config, it also enables every stream with new inputs.
func (r repo) UniteInputs(streams []Stream) (out []Stream) {
	r.log.Info("Uniting inputs of streams")

	out = copier.MustDeep(streams)
	for currIdx, currStream := range out {
		find.EverySimilar(r.cfg.General, out, currStream.Name, currIdx+1, func(nextStream Stream, nextIdx int) {
			for _, nextInput := range nextStream.Inputs {
				r.log.InfoFi("Uniting inputs of streams", "from ID", nextStream.ID, "from name", nextStream.Name,
					"input", nextInput, "to ID", currStream.ID, "to name", currStream.Name,
					"note", currStream.InputsUpdateNote(r.cfg.Streams))
				if !currStream.HasInput(r.log, nextInput, true) {
					currStream = currStream.AddInput(nextInput)
					if r.cfg.Streams.EnableOnInputUpdate && !currStream.Enabled {
						r.log.InfoFi("Enabling the stream (uniting inputs of streams, enable_on_input_update is on)",
							"ID", currStream.ID, "name", currStream.Name)
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

// RemoveDeadInputs returns deep copy of <streams> without dead inputs.
//
// For detailed description, see removeDeadInputs method.
func (r repo) RemoveDeadInputs(httpClient *http.Client, analyzer analyzer.Analyzer, streams []Stream) (out []Stream) {
	r.log.Info("Removing dead inputs from streams")
	return r.removeDeadInputs(httpClient, analyzer, streams, false)
}

// DisableDeadInputs returns deep copy of <streams> with dead inputs disabled.
//
// For detailed description, see removeDeadInputs method.
func (r repo) DisableDeadInputs(httpClient *http.Client, analyzer analyzer.Analyzer, streams []Stream) (out []Stream) {
	r.log.Info("Disabling dead inputs of streams")
	return r.removeDeadInputs(httpClient, analyzer, streams, true)
}

// AddHashes returns deep copy of <streams> with hashes added to every input as defined in config with *ToInputHashMap
func (r repo) AddHashes(streams []Stream) (out []Stream) {
	r.log.Info("Adding hashes to inputs of streams")

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
						r.log.InfoFi("Adding hash to input of stream", "ID", s.ID, "name", s.Name,
							"group", s.FirstGroup(), "hash", rule.Hash, "result", inp)
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
						r.log.InfoFi("Adding hash to input of stream", "ID", s.ID, "name", s.Name,
							"group", s.FirstGroup(), "hash", rule.Hash, "result", inp)
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
						r.log.InfoFi("Adding hash to input of stream", "ID", s.ID, "name", s.Name,
							"group", s.FirstGroup(), "hash", rule.Hash, "result", inp)
					}
				}
			}
			out[sIdx].Inputs[inpIdx] = inp
		}
	}

	return
}

// DisableAllButOneInputByRx returns shallow copy of <streams> with all inputs in every stream disabled except the input
// which matches any regular expression defined in config.
func (r repo) DisableAllButOneInputByRx(streams []Stream) (out []Stream) {
	r.log.Info("Disabling all but one input per stream by regular expressions")

	for _, s := range streams {
		out = append(out, s.disableAllButOneInputByRx(r.cfg.Streams, func(input string) {
			r.log.InfoFi("Disabling other input per stream by regular expressions", "ID", s.ID, "name", s.Name,
				"group", s.FirstGroup(), "input", input)
		}))
	}

	return
}

// RemoveDisabledInputs returns shallow copy of <streams> with all disabled inputs removed
func (r repo) RemoveDisabledInputs(streams []Stream) (out []Stream) {
	r.log.Info("Removing disabled inputs")

	for _, s := range streams {
		out = append(out, s.removeDisabledInputs(func(input string) {
			r.log.InfoFi("Removing disabled input", "ID", s.ID, "name", s.Name,	"group", s.FirstGroup(), "input", input)
		}))
	}

	return
}

// SetKeepActive returns shallow copy of <streams> with HTTPKeepActive set on every stream as defined in config with
// *ToKeepActiveMap.
func (r repo) SetKeepActive(streams []Stream) (out []Stream) {
	r.log.Info("Setting keep active on streams")

	for _, s := range streams {
		// By inputs
		for _, rule := range r.cfg.Streams.InputToKeepActiveMap {
			if slice.RxMatchAny(rule.By, s.Inputs...) {
				keepActiveStr := strconv.Itoa(rule.KeepActive)
				if s.HTTPKeepActive != keepActiveStr {
					r.log.InfoFi("Setting keep active on stream", "ID", s.ID, "name", s.Name, "group", s.FirstGroup(),
						"keep active", keepActiveStr)
					s.HTTPKeepActive = keepActiveStr
				}
				goto Append
			}
		}
		// By name
		for _, rule := range r.cfg.Streams.NameToKeepActiveMap {
			if rule.By.MatchString(s.Name) {
				keepActiveStr := strconv.Itoa(rule.KeepActive)
				if s.HTTPKeepActive != keepActiveStr {
					r.log.InfoFi("Setting keep active on stream", "ID", s.ID, "name", s.Name, "group", s.FirstGroup(),
						"keep active", keepActiveStr)
					s.HTTPKeepActive = keepActiveStr
				}
				goto Append
			}
		}
		// By group
		for _, rule := range r.cfg.Streams.GroupToKeepActiveMap {
			if rule.By.MatchString(s.FirstGroup()) {
				keepActiveStr := strconv.Itoa(rule.KeepActive)
				if s.HTTPKeepActive != keepActiveStr {
					r.log.InfoFi("Setting keep active on stream", "ID", s.ID, "name", s.Name, "group", s.FirstGroup(),
						"keep active", keepActiveStr)
					s.HTTPKeepActive = keepActiveStr
				}
				goto Append
			}
		}
	Append:
		out = append(out, s)
	}

	return
}

// RemoveWithoutInputs returns shallow copy of <streams> with Remove field set to true on streams which have no inputs
func (r repo) RemoveWithoutInputs(streams []Stream) (out []Stream) {
	r.log.Info("Removing streams without inputs")

	out = lo.Map(streams, func(s Stream, _ int) Stream {
		if !s.Remove && s.hasNoInputs() {
			r.log.InfoFi("Removing stream without inputs", "ID", s.ID, "name", s.Name, "group", s.FirstGroup())
			s.Remove = true
		}
		return s
	})

	return
}

// DisableWithoutInputs returns shallow copy of <streams> with all streams disabled if they have no inputs
func (r repo) DisableWithoutInputs(streams []Stream) (out []Stream) {
	r.log.Info("Disabling streams without inputs")

	for _, s := range streams {
		if s.Enabled && s.hasNoInputs() {
			r.log.InfoFi("Disabling stream without inputs", "ID", s.ID, "name", s.Name, "group", s.FirstGroup())
			s = s.disable()
		}
		out = append(out, s)
	}

	return
}

// RemoveNamePrefixes returns shallow copy of <streams> with name prefixes on every stream if MarkAdded or MarkDisabled
// is true.
func (r repo) AddNamePrefixes(streams []Stream) (out []Stream) {
	r.log.Info("Adding name prefixes to streams")

	for _, s := range streams {
		oldName := s.Name
		if s.MarkAdded {
			s = s.setPrefix(r.cfg.Streams.AddedPrefix)
		}
		if s.MarkDisabled {
			s = s.setPrefix(r.cfg.Streams.DisabledPrefix)
		}
		if oldName != s.Name {
			r.log.InfoFi("Adding name prefix to stream", "ID", s.ID, "old name", oldName, "new name", s.Name,
				"group", s.FirstGroup())
		}
		out = append(out, s)
	}

	return
}

// ChangedStreams returns new and changed streams from <newStreams>, which are not in <oldStreams>
func (r repo) ChangedStreams(oldStreams, newStreams []Stream) (out []Stream) {
	r.log.Info("Building changed streams list")

	for _, newStream := range newStreams {
		oldStream, _, found := lo.FindIndexOf(oldStreams, func(oldStream Stream) bool {
			return newStream.ID == oldStream.ID
		})
		if found {
			cmpOption := cmp.FilterPath(func(p cmp.Path) bool {
				lastPathItem := p.Last().String()
				return lastPathItem == ".MarkAdded" || lastPathItem == ".MarkDisabled"
			}, cmp.Ignore())
			if !cmp.Equal(oldStream, newStream, cmpOption) {
				out = append(out, newStream)
			}
		} else {
			out = append(out, newStream)
		}
	}

	return
}

// removeDeadInputs returns deep copy of <streams> without dead inputs.
//
// If <disable> is true, disable dead inputs instead of deleting them.
//
// If cfg.Streams.UseAnalyzer is false:
//
// It removes inputs which do not respond in time or respond with status code >= 400 using <httpClient>.
//
// Supports HTTP(S).
//
// If cfg.Streams.UseAnalyzer is true:
//
// It removes inputs with bitrate lower than specified in config or with amount of errors higher than specified in
// config using <analyzer>.
//
// Supports HTTP(S), UDP, RTP, RTSP.
func (r repo) removeDeadInputs(httpClient *http.Client, analyzer analyzer.Analyzer, streams []Stream,
	disable bool) (out []Stream) {
	// canCheck returns true if <inp> can be checked
	canCheck := func(inp string) bool {
		if slice.AnyRxMatch(r.cfg.Streams.DeadInputsCheckBlacklist, inp) {
			return false
		}
		if slice.HasAnyPrefix(inp, "http://", "https://") {
			return true
		}
		if r.cfg.Streams.UseAnalyzer && slice.HasAnyPrefix(inp, "udp://", "rtp://", "rtsp://") {
			return true
		}
		return false
	}

	// getRemovalReason returns reason why <inp> should be removed
	getRemovalReason := func(inp string) string {
		if r.cfg.Streams.UseAnalyzer {
			result, err := analyzer.Check(r.cfg.Streams.AnalyzerWatchTime, r.cfg.Streams.AnalyzerMaxAttempts, inp)
			if err != nil {
				r.log.Errorf("Failed to run analyzer: %v. Ignoring input %v", err, inp)
				return ""
			}
			// Check bitrate
			hasVideoOnly := result.HasVideo && !result.HasAudio
			hasAudioOnly := !result.HasVideo && result.HasAudio
			bitrate := result.Bitrate
			if hasVideoOnly {
				if bitrate < r.cfg.Streams.AnalyzerVideoOnlyBitrateThreshold {
					return fmt.Sprintf("Bitrate %v < %v", bitrate, r.cfg.Streams.AnalyzerVideoOnlyBitrateThreshold)
				}
			} else if hasAudioOnly {
				if bitrate < r.cfg.Streams.AnalyzerAudioOnlyBitrateThreshold {
					return fmt.Sprintf("Bitrate %v < %v", bitrate, r.cfg.Streams.AnalyzerAudioOnlyBitrateThreshold)
				}
			} else if bitrate < r.cfg.Streams.AnalyzerBitrateThreshold {
				return fmt.Sprintf("Bitrate %v < %v", bitrate, r.cfg.Streams.AnalyzerBitrateThreshold)
			}
			// Check errors
			ccErrorsThreshold := r.cfg.Streams.AnalyzerCCErrorsThreshold
			pcrErrorsThreshold := r.cfg.Streams.AnalyzerPCRErrorsThreshold
			pesErrorsThreshold := r.cfg.Streams.AnalyzerPESErrorsThreshold
			if ccErrorsThreshold >= 0 && result.CCErrors > ccErrorsThreshold {
				return fmt.Sprintf("CC errors %v > %v", result.CCErrors, ccErrorsThreshold)
			}
			if pcrErrorsThreshold >= 0 && result.PCRErrors > pcrErrorsThreshold {
				return fmt.Sprintf("PCR errors %v > %v", result.PCRErrors, pcrErrorsThreshold)
			}
			if pesErrorsThreshold >= 0 && result.PESErrors > pesErrorsThreshold {
				return fmt.Sprintf("PES errors %v > %v", result.PESErrors, pesErrorsThreshold)
			}
		} else {
			resp, err := httpClient.Get(inp)
			if err == nil {
				defer resp.Body.Close()
			}
			// Not checking Content-Type header as server can return text/html but stream still will be playable
			// Not checking response body as some streams can periodically respond with no content but still be playable
			if err != nil {
				errType := network.GetErrType(err)
				return lo.Ternary(errType == network.Unknown, err.Error(), string(errType))
			} else if resp.StatusCode >= 400 {
				return fmt.Sprintf("Responded with: %v", resp.Status)
			}
		}
		return ""
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

	progressScheduler := gocron.NewScheduler(time.UTC)
	_, err := progressScheduler.Every(30).Seconds().Do(func() {
		msg := lo.Ternary(disable, "Disabling dead inputs of streams", "Removing dead inputs from streams")
		r.log.InfoFi(msg, "progress", getProgress())
	})
	if err != nil {
		r.log.Errorf("Failed to print progress of removing dead inputs: %v", err)
	}
	progressScheduler.StartAsync()

	out = copier.MustDeep(streams)
	for sIdx, s := range out {
		for _, inp := range s.Inputs {
			pool.Submit(func() {
				r.log.DebugFi("Start checking input", "stream ID", s.ID, "stream name", s.Name, "stream index", sIdx,
					"input", inp)
				if canCheck(inp) {
					removalReason := getRemovalReason(inp)
					if removalReason != "" {
						msg := lo.Ternary(disable, "Disabling dead input of stream", "Removing dead input from stream")
						r.log.WarnFi(msg, "ID", s.ID, "name", s.Name, "group", s.FirstGroup(), "input", inp,
							"reason", removalReason)
						mut.Lock()
						out[sIdx].Inputs = slice.RemoveLast(out[sIdx].Inputs, inp)
						mut.Unlock()
						if disable {
							out[sIdx].DisabledInputs = append(out[sIdx].DisabledInputs, inp)
						}
					}
				}

				mut.Lock()
				inputsDone++
				mut.Unlock()
				r.log.DebugFi("End checking input", "stream ID", s.ID, "stream name", s.Name, "stream index", sIdx,
					"input", inp)
			})
		}
	}

	pool.StopAndWait()
	progressScheduler.Stop()

	return
}

// getInputsAmount returns total amount of inputs in <streams>
func getInputsAmount(streams []Stream) int {
	return lo.SumBy(streams, func(s Stream) int {
		return len(s.Inputs)
	})
}
