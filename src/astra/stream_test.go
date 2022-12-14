package astra

import (
	"m3u_merge_astra/cfg"
	"m3u_merge_astra/util/copier"
	"m3u_merge_astra/util/network"
	"net/http"
	"regexp"
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"
)

func TestNewStream(t *testing.T) {
	cfg := newDefRepo().cfg.Streams
	s := NewStream(cfg, "0000", "Name", "Group", []string{"http://url"})

	expected := Stream{
		Enabled:        cfg.MakeNewEnabled,
		Type:           string(cfg.NewType),
		ID:             "0000",
		Name:           cfg.AddedPrefix + "Name",
		Inputs:         []string{"http://url"},
		DisabledInputs: make([]string, 0),
	}
	assert.Exactly(t, expected, s, "should create this stream")

	cfg.AddGroupsToNew = true
	s = NewStream(cfg, "0000", "Name", "Group", []string{"http://url"})

	expected.Groups = map[string]any{cfg.GroupsCategoryForNew: "Group"}
	assert.Exactly(t, expected, s, "should create this stream")
}

func TestGetName(t *testing.T) {
	s := Stream{Name: "Name"}
	assert.Exactly(t, s.Name, s.GetName(), "should return this name")
}

func TestFirstGroup(t *testing.T) {
	s := Stream{}
	assert.Empty(t, s.FirstGroup(), "should return empty group")
	s = Stream{Groups: map[string]any{"Category 1": "Group 1", "Category 2": "Group 2"}}
	assert.Exactly(t, "Group 1", s.FirstGroup(), "should return first group name")
}

func TestUpdateStreamInput(t *testing.T) {
	r := newDefRepo()

	r.cfg.Streams.InputUpdateMap = []cfg.UpdateRecord{
		{From: *regexp.MustCompile("update/from"), To: *regexp.MustCompile("update/to")},
		{From: *regexp.MustCompile("update/url"), To: *regexp.MustCompile("update/url")},
	}
	s1 := Stream{Inputs: []string{
		"http://irrelevant/from#a", "http://update/from/1#c", "http://update/url/1#b", "http://update/url/1#c",
	}}
	s1Original := copier.TDeep(t, s1)

	r.cfg.Streams.KeepInputHash = false
	s2 := s1.UpdateInput(r, "http://update/to/1")
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected := []string{
		"http://irrelevant/from#a", "http://update/to/1", "http://update/url/1#b", "http://update/url/1#c",
	}
	assert.Exactly(t, expected, s2.Inputs, "should have these inputs")

	s2 = s1.UpdateInput(r, "http://update/url/1#b")
	expected = []string{
		"http://irrelevant/from#a", "http://update/from/1#c", "http://update/url/1#b", "http://update/url/1#b",
	}
	assert.Exactly(t, expected, s2.Inputs, "relevant input should be updated discarding old hash")

	s2 = s1.UpdateInput(r, "http://irrelevant/to")
	assert.Exactly(t, s1, s2, "inputs should not be updated with irrelevant URL")

	r.cfg.Streams.KeepInputHash = true
	s2 = s1.UpdateInput(r, "http://update/url/2")
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected = []string{
		"http://irrelevant/from#a", "http://update/from/1#c", "http://update/url/2#b", "http://update/url/1#c",
	}
	assert.Exactly(t, expected, s2.Inputs, "relevant input should be updated keeping old hash")

	s2 = s1.UpdateInput(r, "http://update/url/2#c")
	expected = []string{
		"http://irrelevant/from#a", "http://update/from/1#c", "http://update/url/2#c&b", "http://update/url/1#c",
	}
	assert.Exactly(t, expected, s2.Inputs, "relevant input should be updated, merging hashes")

	s2 = s1.UpdateInput(r, "http://irrelevant/to")
	assert.Exactly(t, s1, s2, "inputs should not be updated with irrelevant URL")
}

func TestHasInput(t *testing.T) {
	r := newDefRepo()

	s := Stream{Inputs: []string{"http://other/input", "http://known/input#a"}}

	assert.False(t, s.HasInput(r, "http://known/input", true), "should not contain URL without hash")
	assert.True(t, s.HasInput(r, "http://known/input#a", true), "should contain URL")

	assert.True(t, s.HasInput(r, "http://known/input", false), "should contain URL without hash")
	assert.True(t, s.HasInput(r, "http://known/input#b", false), "should contain URL with different hashes")

	assert.False(t, s.HasInput(r, "http://foreign/input", true), "should not contain URL")
	assert.False(t, s.HasInput(r, "http://foreign/input#a", true), "should not contain URL")

	assert.False(t, s.HasInput(r, "http://foreign/input", false), "should not contain URL")
	assert.False(t, s.HasInput(r, "http://foreign/input#b", false), "should not contain URL")

	s = Stream{Inputs: []string{"http://other/input#a", "http:/other/input/2#b"}}

	assert.False(t, s.HasInput(r, "http://foreign/input", true), "should not contain URL")
	assert.False(t, s.HasInput(r, "http://foreign/input#a", true), "should not contain URL")

	assert.False(t, s.HasInput(r, "http://foreign/input", false), "should not contain URL")
	assert.False(t, s.HasInput(r, "http://foreign/input#b", false), "should not contain URL")
}

func TestAddInput(t *testing.T) {
	r := newDefRepo()

	s1 := Stream{Inputs: []string{"http://input/1", "http://input/2"}}
	s1Original := copier.TDeep(t, s1)

	s2 := s1.AddInput(r, "http://input/3", false)
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected := []string{"http://input/3", "http://input/1", "http://input/2"}
	assert.Exactly(t, expected, s2.Inputs, "should have these inputs")
}

func TestKnownInputs(t *testing.T) {
	r := newDefRepo()

	r.cfg.Streams.InputUpdateMap = []cfg.UpdateRecord{
		{From: *regexp.MustCompile("known/input/1")},
		{From: *regexp.MustCompile("known/input/2")},
	}

	s := Stream{Inputs: []string{"http://known/input/2#a", "http://other/input", "http://known/input/1"}}
	ki := s.KnownInputs(r)

	expected := []string{"http://known/input/2#a", "http://known/input/1"}
	assert.Exactly(t, expected, ki, "should have these inputs")
}

func TestRemoveInputs(t *testing.T) {
	r := newDefRepo()

	s1 := Stream{Inputs: []string{"http://input/1", "http://input/2", "http://input/1"}}
	s1Original := copier.TDeep(t, s1)

	s2 := s1.RemoveInputs(r, "http://input/1", false)
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected := []string{"http://input/2"}
	assert.Exactly(t, expected, s2.Inputs, "should have these inputs")
}

func TestDisableStream(t *testing.T) {
	r := newDefRepo()

	s1 := Stream{Name: "Name", Enabled: false}
	s1Original := copier.TDeep(t, s1)
	s2 := s1.disable(r)
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected := Stream{Name: r.cfg.Streams.DisabledPrefix + "Name", Enabled: false}
	assert.Exactly(t, expected, s2, "should have disabled prefix")

	s1 = Stream{Name: r.cfg.Streams.DisabledPrefix + "Name", Enabled: false}
	s2 = s1.disable(r)
	expected = Stream{Name: r.cfg.Streams.DisabledPrefix + "Name", Enabled: false}
	assert.Exactly(t, expected, s2, "should stay disabled")

	s1 = Stream{Name: r.cfg.Streams.AddedPrefix + "Name", Enabled: false}
	s2 = s1.disable(r)
	expected = Stream{Name: r.cfg.Streams.AddedPrefix + "Name", Enabled: false}
	assert.Exactly(t, expected, s2, "should stay unchanged")

	s1 = Stream{Name: "Name", Enabled: true}
	s2 = s1.disable(r)
	expected = Stream{Name: r.cfg.Streams.DisabledPrefix + "Name", Enabled: false}
	assert.Exactly(t, expected, s2, "should have disabled prefix and set Enabled field to false")

	s1 = Stream{Name: r.cfg.Streams.DisabledPrefix + "Name", Enabled: true}
	s2 = s1.disable(r)
	expected = Stream{Name: r.cfg.Streams.DisabledPrefix + "Name", Enabled: false}
	assert.Exactly(t, expected, s2, "disabled prefix should stay and set Enabled field to false")

	s1 = Stream{Name: r.cfg.Streams.AddedPrefix + "Name", Enabled: true}
	s2 = s1.disable(r)
	expected = Stream{Name: r.cfg.Streams.AddedPrefix + "Name", Enabled: false}
	assert.Exactly(t, expected, s2, "name should stay unmodified and set Enabled field to false")
}

func TestEnableStream(t *testing.T) {
	r := newDefRepo()

	s1 := Stream{Name: "Name", Enabled: false}
	s1Original := copier.TDeep(t, s1)
	s2 := s1.enable(r, false)
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected := Stream{Name: "Name", Enabled: true}
	assert.Exactly(t, expected, s2, "should set Enabled field to true")

	s1 = Stream{Name: r.cfg.Streams.DisabledPrefix + "Name", Enabled: false}
	s2 = s1.enable(r, false)
	expected = Stream{Name: "Name", Enabled: true}
	assert.Exactly(t, expected, s2, "should remove disabled prefix and set Enabled field to true")

	s1 = Stream{Name: "Name", Enabled: true}
	s2 = s1.enable(r, false)
	expected = Stream{Name: "Name", Enabled: true}
	assert.Exactly(t, expected, s2, "should stay unchanged")

	s1 = Stream{Name: r.cfg.Streams.DisabledPrefix + "Name", Enabled: true}
	s2 = s1.enable(r, false)
	expected = Stream{Name: "Name", Enabled: true}
	assert.Exactly(t, expected, s2, "should remove disabled prefix")

	s1 = Stream{Name: "Name", Enabled: false}
	s1Original = copier.TDeep(t, s1)
	s2 = s1.enable(r, true)
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	assert.Exactly(t, s1, s2, "should stay unchanged")

	s1 = Stream{Name: r.cfg.Streams.DisabledPrefix + "Name", Enabled: false}
	s2 = s1.enable(r, true)
	expected = Stream{Name: "Name", Enabled: true}
	assert.Exactly(t, expected, s2, "should remove disabled prefix and set Enabled field to true")

	s1 = Stream{Name: "Name", Enabled: true}
	s2 = s1.enable(r, true)
	assert.Exactly(t, s1, s2, "should stay unchanged")

	s1 = Stream{Name: r.cfg.Streams.DisabledPrefix + "Name", Enabled: true}
	s2 = s1.enable(r, true)
	expected = Stream{Name: "Name", Enabled: true}
	assert.Exactly(t, expected, s2, "should remove disabled prefix")
}

func TestRemoveBlockedInputs(t *testing.T) {
	r := newDefRepo()

	r.cfg.Streams.InputBlacklist = []regexp.Regexp{
		*regexp.MustCompile("input/1"),
		*regexp.MustCompile("input/3"),
	}

	s1 := Stream{Inputs: []string{"http://input/1", "http://input/2", "http://input/1", "http://input/3"}}
	s1Original := copier.TDeep(t, s1)

	s2 := s1.removeBlockedInputs(r)

	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected := []string{"http://input/2"}
	assert.Exactly(t, expected, s2.Inputs, "should have these inputs")
}

func TestRemoveDuplicatedInputs(t *testing.T) {
	r := newDefRepo()

	sl1 := []Stream{
		{Inputs: []string{"http://input/2", "http://input/1", "http://input/3", "http://input/2", "http://input/3"}},
		{},
		{Inputs: []string{"http://input/4", "http://input/5", "http://input/1"}},
		{Inputs: []string{"http://input/2", "http://input/4", "http://input/6"}},
	}
	sl1Original := copier.TDeep(t, sl1)

	sl2 := r.RemoveDuplicatedInputs(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	expected := []string{"http://input/2", "http://input/1", "http://input/3"}
	assert.Exactly(t, expected, sl2[0].Inputs, "should have these inputs")

	assert.Exactly(t, sl1[1], sl2[1], "should not modify streams without inputs")

	expected = []string{"http://input/4", "http://input/5"}
	assert.Exactly(t, expected, sl2[2].Inputs, "should remove inputs existing in previous streams")

	expected = []string{"http://input/6"}
	assert.Exactly(t, expected, sl2[3].Inputs, "should remove inputs existing in previous streams")
}

func TestUniteInputs(t *testing.T) {
	r := newDefRepo()

	sl1 := []Stream{
		/* 0 */ {Name: "Name", Inputs: []string{"http://input/a"}},
		/* 1 */ {Name: "Name 3", Inputs: []string{"http://input/a"}},
		/* 2 */ {Name: "Name 2", Inputs: []string{"http://input/a"}},
		/* 3 */ {Name: "Name_3", Inputs: []string{"http://input/b", "http://input/a"}},
		/* 4 */ {Name: "Name-3", Inputs: []string{"http://input/c"}},
		/* 5 */ {Name: "Name_2", Inputs: []string{"http://input/b", "http://input/a"}},
	}
	sl1Original := copier.TDeep(t, sl1)

	sl2 := r.UniteInputs(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	assert.Exactly(t, sl1[0], sl2[0], "should not modify stream without existing duplicate by name")

	expected := Stream{Name: "Name 3", Inputs: []string{"http://input/c", "http://input/b", "http://input/a"}}
	assert.Exactly(t, expected, sl2[1], "should be this stream")

	expected = Stream{Name: "Name 2", Inputs: []string{"http://input/b", "http://input/a"}}
	assert.Exactly(t, expected, sl2[2], "should be this stream")

	expected = Stream{Name: "Name_3", Inputs: make([]string, 0), DisabledInputs: make([]string, 0)}
	assert.Exactly(t, expected, sl2[3], "should remove inputs from subsequent streams duplicated by name")

	expected = Stream{Name: "Name-3", Inputs: make([]string, 0), DisabledInputs: make([]string, 0)}
	assert.Exactly(t, expected, sl2[4], "should remove inputs from subsequent streams duplicated by name")

	expected = Stream{Name: "Name_2", Inputs: make([]string, 0), DisabledInputs: make([]string, 0)}
	assert.Exactly(t, expected, sl2[5], "should remove inputs from subsequent streams duplicated by name")
}

func TestSortInputs(t *testing.T) {
	r := newDefRepo()

	// Multiple entries
	r.cfg.Streams.UnknownInputWeight = 25
	r.cfg.Streams.InputWeightToTypeMap = map[int]regexp.Regexp{
		20: *regexp.MustCompile(`input/20`),
		30: *regexp.MustCompile(`input/30`),
	}
	sl1 := []Stream{
		{Inputs: []string{"http://other/a", "http://other/b", "http://input/30", "http://input/20"}},
		{},
		{Inputs: []string{"http://other/c", "http://other/d"}},
	}
	sl1Original := copier.TDeep(t, sl1)

	sl2 := r.SortInputs(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	expected := []string{"http://input/20", "http://other/a", "http://other/b", "http://input/30"}
	assert.Exactly(t, expected, sl2[0].Inputs, "inputs should have this order")

	assert.Exactly(t, sl1[1], sl2[1], "should not modify streams without inputs")

	assert.Exactly(t, sl1[2], sl2[2], "should not modify streams with unknown inputs")

	// One entry
	r.cfg.Streams.UnknownInputWeight = 30
	r.cfg.Streams.InputWeightToTypeMap = map[int]regexp.Regexp{
		20: *regexp.MustCompile(`input/20`),
	}
	sl1 = []Stream{
		{Inputs: []string{"http://other/a", "http://other/b", "http://other/c", "http://input/20"}},
		{Inputs: []string{"http://other/d", "http://other/e"}},
	}
	sl1Original = copier.TDeep(t, sl1)

	sl2 = r.SortInputs(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	expected = []string{"http://input/20", "http://other/a", "http://other/b", "http://other/c"}
	assert.Exactly(t, expected, sl2[0].Inputs, "inputs should have this order")

	assert.Exactly(t, sl1[1], sl2[1], "should not modify streams with unknown inputs")

	// Empty map
	r.cfg.Streams.UnknownInputWeight = 50
	r.cfg.Streams.InputWeightToTypeMap = map[int]regexp.Regexp{}
	sl1 = []Stream{
		{Inputs: []string{"http://other/d", "http://other/c", "http://other/b", "http://other/a"}},
		{Inputs: []string{"http://other/f", "http://other/e"}},
	}
	sl1Original = copier.TDeep(t, sl1)

	sl2 = r.SortInputs(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Exactly(t, sl1, sl2, "should stay the same")
}

func TestHasNoInputs(t *testing.T) {
	s := Stream{}
	assert.True(t, s.hasNoInputs())
	s = Stream{Inputs: []string{}}
	assert.True(t, s.hasNoInputs())
	s = Stream{Inputs: []string{"http://input"}}
	assert.False(t, s.hasNoInputs())
}

func TestAnyHasInput(t *testing.T) {
	r := newDefRepo()

	sl := []Stream{
		{Inputs: []string{"http://other/input", "http:/other/input/2"}},
		{Inputs: []string{"http://other/input", "http://known/input#a"}},
	}
	assert.False(t, r.HasInput(sl, "http://known/input", true), "should not contain URL without hash")
	assert.True(t, r.HasInput(sl, "http://known/input#a", true), "should contain URL")

	assert.True(t, r.HasInput(sl, "http://known/input", false), "should contain URL without hash")
	assert.True(t, r.HasInput(sl, "http://known/input#b", false), "should contain URL with different hashes")

	assert.False(t, r.HasInput(sl, "http://foreign/input", true), "should not contain URL")
	assert.False(t, r.HasInput(sl, "http://foreign/input#a", true), "should not contain URL")

	assert.False(t, r.HasInput(sl, "http://foreign/input", false), "should not contain URL")
	assert.False(t, r.HasInput(sl, "http://foreign/input#b", false), "should not contain URL")

	sl = []Stream{
		{Inputs: []string{"http://other/input#a", "http:/other/input/2#b"}},
	}
	assert.False(t, r.HasInput(sl, "http://foreign/input", true), "should not contain URL")
	assert.False(t, r.HasInput(sl, "http://foreign/input#a", true), "should not contain URL")

	assert.False(t, r.HasInput(sl, "http://foreign/input", false), "should not contain URL")
	assert.False(t, r.HasInput(sl, "http://foreign/input#b", false), "should not contain URL")
}

func TestEnableAllStreams(t *testing.T) {
	r := newDefRepo()

	sl1 := []Stream{
		{Name: "Name", Enabled: false},
		{Name: r.cfg.Streams.DisabledPrefix + "Name", Enabled: false},
		{Name: r.cfg.Streams.DisabledPrefix + "Name 2", Enabled: true},
		{Name: r.cfg.Streams.AddedPrefix + "Name", Enabled: false},
	}
	sl1Original := copier.TDeep(t, sl1)

	sl2 := r.Enable(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	assert.Exactly(t, sl1[0], sl2[0], "should not enable stream without disabled prefix")

	expected := Stream{Name: "Name", Enabled: true}
	assert.Exactly(t, expected, sl2[1], "should remove disabled prefix and set Enabled field to true")

	expected = Stream{Name: "Name 2", Enabled: true}
	assert.Exactly(t, expected, sl2[2], "should remove disabled prefix and set Enabled field to true")

	assert.Exactly(t, sl1[3], sl2[3], "should not enable stream with added prefix")
}

func TestAllRemoveBlockedInputs(t *testing.T) {
	r := newDefRepo()
	r.cfg.Streams.InputBlacklist = []regexp.Regexp{
		*regexp.MustCompile("input/1"),
		*regexp.MustCompile("input/3"),
	}

	sl1 := []Stream{
		{Inputs: []string{"http://input/1", "http://input/2", "http://input/1", "http://input/3"}},
		{Inputs: []string{"http://input/1"}},
		{},
	}
	sl1Original := copier.TDeep(t, sl1)

	sl2 := r.RemoveBlockedInputs(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	expected := []string{"http://input/2"}
	assert.Exactly(t, expected, sl2[0].Inputs, "should have these inputs")

	assert.Len(t, sl2[1].Inputs, 0, "should remove all specified inputs")

	assert.Len(t, sl2[2].Inputs, 0, "should stay 0")
}

func TestRemoveDeadInputs(t *testing.T) {
	r := newDefRepo()
	r.cfg.Streams.InputMaxConns = 100

	// Create request handlers
	handleAlive := func(w http.ResponseWriter, req *http.Request) {
		r.log.Debugf("Got request to %v", req.URL)
		w.WriteHeader(200)
	}
	handleTimeout := func(w http.ResponseWriter, req *http.Request) {
		r.log.Debugf("Got request to %v", req.URL)
		time.Sleep(time.Second * 5)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/alive/", handleAlive)
	mux.HandleFunc("/dead/timeout/", handleTimeout)

	// Run http & https servers as subset of current test to be able to fail it from another goroutines (servers) if any
	// server returns error
	var httpSrv, httpsSrv *http.Server
	t.Run("http_server", func(t *testing.T) {
		httpSrv, httpsSrv = network.NewHttpServer(mux, 3434, 5656, func(err error) {
			if !errors.Is(err, http.ErrServerClosed) {
				// Not using logging from testing.T or else message will not be displayed
				r.log.Errorf("Test server stopped with non-standard error: %v", err)
				t.FailNow()
			}
		})
	})
	defer httpSrv.Close()
	defer httpsSrv.Close()

	// Check results
	r.cfg.Streams.DeadInputsCheckBlacklist = []regexp.Regexp{
		*regexp.MustCompile(`ignore/1`),
		*regexp.MustCompile(`ignore/2`),
	}
	sl1 := []Stream{
		{Inputs: []string{"https://127.0.0.1:5656/dead/timeout/1", "https://127.0.0.1:5656/alive/1"}},
		{Inputs: []string{"http://127.0.0.1:3434/alive/2", "http://dead/no_such_host/1", "http://ignore/2"}},
		{Inputs: []string{"http://127.0.0.1:3434/dead/timeout/" + strings.Repeat("x", 40), "https://ignore/1"}},
		{Inputs: []string{"rtp://skip/1", "rtsp://skip/2", "file:///skip/3.ts", "http://127.0.0.1:3434/dead/404/1"}},
	}
	sl1Original := copier.TDeep(t, sl1)

	client := network.NewHttpClient(false, time.Second*3)
	sl2 := r.RemoveDeadInputs(client, sl1, false)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	expected := []string{"https://127.0.0.1:5656/alive/1"}
	assert.Exactly(t, expected, sl2[0].Inputs, "should have these inputs")

	expected = []string{"http://127.0.0.1:3434/alive/2", "http://ignore/2"}
	assert.Exactly(t, expected, sl2[1].Inputs, "should have these inputs")

	expected = []string{"https://ignore/1"}
	assert.Exactly(t, expected, sl2[2].Inputs, "should have these inputs")

	expected = []string{"rtp://skip/1", "rtsp://skip/2", "file:///skip/3.ts"}
	assert.Exactly(t, expected, sl2[3].Inputs, "should not remove inputs with unsupported protocols")

	// ---------- Test concurrency ----------
	// Unexpectedly freezing on Windows 10 after ~20 seconds of the test runnning.
	// Use test_remove_dead_inputs.sh

	sl1 = []Stream{
		{Inputs: []string{"http://127.0.0.1:3434/dead/404/1", "rtp://skip/1", "http://127.0.0.1:3434/dead/404/1"}},
		{Inputs: []string{"http://ignore/1", "http://127.0.0.1:3434/dead/404/2", "http://127.0.0.1:3434/dead/404/2"}},
		{Inputs: []string{"rtsp://skip/2", "http://ignore/2", "rtsp://skip/2", "https://127.0.0.1:5656/alive/1"}},
	}
	sl1Original = copier.TDeep(t, sl1)

	client = network.NewHttpClient(true, time.Second*3)

	for i := 0; i < 10000; i++ {
		sl2 = r.RemoveDeadInputs(client, sl1, false)
		if ok := assert.NotSame(t, &sl1, &sl2, "should return copy of streams"); !ok {
			t.FailNow()
		}
		if ok := assert.Exactly(t, sl1Original, sl1, "should not modify the source"); !ok {
			t.FailNow()
		}
		if ok := assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same"); !ok {
			t.FailNow()
		}
		expected := []string{"rtp://skip/1"}
		if ok := assert.Exactly(t, expected, sl2[0].Inputs, "should have these inputs"); !ok {
			t.FailNow()
		}
		expected = []string{"http://ignore/1"}
		if ok := assert.Exactly(t, expected, sl2[1].Inputs, "should have these inputs"); !ok {
			t.FailNow()
		}
		expected = []string{"rtsp://skip/2", "http://ignore/2", "rtsp://skip/2", "https://127.0.0.1:5656/alive/1"}
		if ok := assert.Exactly(t, expected, sl2[2].Inputs, "should have these inputs"); !ok {
			t.FailNow()
		}
	}
}

func TestAddHashes(t *testing.T) {
	r := newDefRepo()
	r.cfg.Streams.NameToInputHashMap = []cfg.HashAddRule{
		{By: *regexp.MustCompile(`Known name 1`), Hash: "a"},
		{By: *regexp.MustCompile(`Known name 2`), Hash: "b"},
	}
	r.cfg.Streams.GroupToInputHashMap = []cfg.HashAddRule{
		{By: *regexp.MustCompile(`Known group 1`), Hash: "c"},
		{By: *regexp.MustCompile(`Known group 2`), Hash: "d"},
	}
	r.cfg.Streams.InputToInputHashMap = []cfg.HashAddRule{
		{By: *regexp.MustCompile(`http://known/input/1`), Hash: "e"},
		{By: *regexp.MustCompile(`http://known/input/2`), Hash: "f"},
	}

	sl1 := []Stream{
		{ // Index 0. Known input 1
			Name:   "Other name",
			Groups: map[string]any{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://known/input/1#x", "http://other/input/1"},
		},
		{ // Index 1. Known name 1
			Name:   "Known name 1",
			Groups: map[string]any{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://other/input/1#a", "http://other/input/2#x"},
		},
		{ // Index 2. Known group 2
			Name:   "Other name",
			Groups: map[string]any{r.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
			Inputs: []string{"http://other/input/1#a&d", "http://other/input/2"},
		},
		{ // Index 3. Known inputs 2 and 1
			Name:   "Other name",
			Groups: map[string]any{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://known/input/2#x", "http://known/input/1"},
		},
		{ // Index 4. Known name 2
			Name:   "Known name 2",
			Groups: map[string]any{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://other/input/1"},
		},
		{ // Index 5. Known group 1
			Name:   "Other name",
			Groups: map[string]any{r.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
			Inputs: []string{"http://other/input/1#c", "http://other/input/2#x"},
		},
		{ // Index 6. No matches
			Name:   "Other name",
			Groups: map[string]any{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://other/input/2", "http://other/input/1#a"},
		},
		{ // Index 7. Matches by every parameter
			Name:   "Known name 1",
			Groups: map[string]any{r.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
			Inputs: []string{"http://known/input/2#x", "http://other/input/1", "http://known/input/1"},
		},
		{ // Index 8. Matches by group 1 and input 1
			Name:   "Other name",
			Groups: map[string]any{r.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
			Inputs: []string{"http://known/input/1", "http://other/input/1"},
		},
	}
	sl1Original := copier.TDeep(t, sl1)

	sl2 := r.AddHashes(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	expected := Stream{ // Index 0. Known input 1
		Name:   "Other name",
		Groups: map[string]any{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
		Inputs: []string{"http://known/input/1#x&e", "http://other/input/1"},
	}
	assert.Exactly(t, expected, sl2[0], "inputs matching only by StreamInputToInputHashMap should get hashes only for"+
		"the exact inputs")

	expected = Stream{ // Index 1. Known name 1
		Name:   "Known name 1",
		Groups: map[string]any{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
		Inputs: []string{"http://other/input/1#a", "http://other/input/2#x&a"},
	}
	assert.Exactly(t, expected, sl2[1], "should add hash to every matching input")

	expected = Stream{ // Index 2. Known group 2
		Name:   "Other name",
		Groups: map[string]any{r.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
		Inputs: []string{"http://other/input/1#a&d", "http://other/input/2#d"},
	}
	assert.Exactly(t, expected, sl2[2], "should add hash to every matching input")

	expected = Stream{ // Index 3. Known inputs 2 and 1
		Name:   "Other name",
		Groups: map[string]any{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
		Inputs: []string{"http://known/input/2#x&f", "http://known/input/1#e"},
	}
	assert.Exactly(t, expected, sl2[3], "should add hash to every matching input")

	expected = Stream{ // Index 4. Known name 2
		Name:   "Known name 2",
		Groups: map[string]any{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
		Inputs: []string{"http://other/input/1#b"},
	}
	assert.Exactly(t, expected, sl2[4], "should add hash to matching input")

	expected = Stream{ // Index 5. Known group 1
		Name:   "Other name",
		Groups: map[string]any{r.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
		Inputs: []string{"http://other/input/1#c", "http://other/input/2#x&c"},
	}
	assert.Exactly(t, expected, sl2[5], "should add hash to every matching input")

	// Index 6. No matches
	assert.Exactly(t, sl1[6], sl2[6], "should not modify stream with no matches")

	expected = Stream{ // Index 7. Matches by every parameter
		Name:   "Known name 1",
		Groups: map[string]any{r.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
		Inputs: []string{"http://known/input/2#x&f&a&d", "http://other/input/1#a&d", "http://known/input/1#e&a&d"},
	}
	assert.Exactly(t, expected, sl2[7], "should add hash to every matching input by every parameter")

	expected = Stream{ // Index 8. Matches by group 1 and input 1
		Name:   "Other name",
		Groups: map[string]any{r.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
		Inputs: []string{"http://known/input/1#e&c", "http://other/input/1#c"},
	}
	assert.Exactly(t, expected, sl2[8], "should add hash to every matching input by every parameter")
}

func TestRemoveWithoutInputs(t *testing.T) {
	r := newDefRepo()

	sl1 := []Stream{
		{Groups: map[string]any{r.cfg.Streams.GroupsCategoryForNew: "Group"}},
		{Enabled: true, Name: "Name"},
		{Enabled: true, Inputs: []string{"http://input/1", "http://input/2"}},
		{Enabled: false, Name: r.cfg.Streams.DisabledPrefix + "Name"},
		{Inputs: []string{"http://input"}},
	}
	sl1Original := copier.TDeep(t, sl1)

	sl2 := r.RemoveWithoutInputs(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	expected := []Stream{
		{Enabled: true, Inputs: []string{"http://input/1", "http://input/2"}},
		{Inputs: []string{"http://input"}},
	}

	assert.Exactly(t, expected, sl2, "should remove streams without inputs")
}

func TestDisableWithoutInputs(t *testing.T) {
	r := newDefRepo()

	sl1 := []Stream{
		/*  0 */ {Enabled: true, Name: "Name", Inputs: []string{"http://input"}},
		/*  1 */ {Enabled: true, Inputs: []string{"http://input"}},
		/*  2 */ {Enabled: true, Name: r.cfg.Streams.DisabledPrefix + "Name", Inputs: []string{"http://input"}},
		/*  3 */ {Enabled: true, Name: r.cfg.Streams.AddedPrefix + "Name", Inputs: []string{"http://input"}},

		/*  4 */ {Enabled: true, Name: "Name"},
		/*  5 */ {Enabled: true},
		/*  6 */ {Enabled: true, Name: r.cfg.Streams.DisabledPrefix + "Name"},
		/*  7 */ {Enabled: true, Name: r.cfg.Streams.AddedPrefix + "Name"},

		/*  8 */ {Enabled: false, Name: "Name", Inputs: []string{"http://input"}},
		/*  9 */ {Enabled: false, Inputs: []string{"http://input"}},
		/* 10 */ {Enabled: false, Name: r.cfg.Streams.DisabledPrefix + "Name", Inputs: []string{"http://input"}},
		/* 11 */ {Enabled: false, Name: r.cfg.Streams.AddedPrefix + "Name", Inputs: []string{"http://input"}},

		/* 12 */ {Enabled: false, Name: "Name"},
		/* 13 */ {Enabled: false},
		/* 14 */ {Enabled: false, Name: r.cfg.Streams.DisabledPrefix + "Name"},
		/* 15 */ {Enabled: false, Name: r.cfg.Streams.AddedPrefix + "Name"},
	}
	sl1Original := copier.TDeep(t, sl1)

	sl2 := r.DisableWithoutInputs(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	assert.Exactly(t, sl1[0], sl2[0], "should not change the stream with inputs")
	assert.Exactly(t, sl1[1], sl2[1], "should not change the stream with inputs")
	assert.Exactly(t, sl1[2], sl2[2], "should not change the stream with inputs")
	assert.Exactly(t, sl1[3], sl2[3], "should not change the stream with inputs")

	expected := Stream{Name: r.cfg.Streams.DisabledPrefix + "Name", Enabled: false}
	assert.Exactly(t, expected, sl2[4], "should add disabled prefix and set Enabled field to false")

	expected = Stream{Name: r.cfg.Streams.DisabledPrefix, Enabled: false}
	assert.Exactly(t, expected, sl2[5], "should add disabled prefix and set Enabled field to false")

	expected = Stream{Name: r.cfg.Streams.DisabledPrefix + "Name", Enabled: false}
	assert.Exactly(t, expected, sl2[6], "should not rename prefixed streams and set Enabled field to false")

	expected = Stream{Name: r.cfg.Streams.AddedPrefix + "Name", Enabled: false}
	assert.Exactly(t, expected, sl2[7], "should not rename prefixed streams and set Enabled field to false")

	assert.Exactly(t, sl1[8], sl2[8], "should not change the stream with inputs")
	assert.Exactly(t, sl1[9], sl2[9], "should not change the stream with inputs")
	assert.Exactly(t, sl1[10], sl2[10], "should not change the stream with inputs")
	assert.Exactly(t, sl1[11], sl2[11], "should not change the stream with inputs")

	expected = Stream{Name: r.cfg.Streams.DisabledPrefix + "Name", Enabled: false}
	assert.Exactly(t, expected, sl2[12], "should add disabled prefix")

	expected = Stream{Name: r.cfg.Streams.DisabledPrefix, Enabled: false}
	assert.Exactly(t, expected, sl2[13], "should add disabled prefix")

	expected = Stream{Name: r.cfg.Streams.DisabledPrefix + "Name", Enabled: false}
	assert.Exactly(t, expected, sl2[14], "should not rename prefixed streams")

	expected = Stream{Name: r.cfg.Streams.AddedPrefix + "Name", Enabled: false}
	assert.Exactly(t, expected, sl2[15], "should not rename prefixed streams")
}

func TestGetInputsAmount(t *testing.T) {
	sl1 := []Stream{
		{Inputs: []string{"http://input/1"}},
		{Inputs: []string{"http://input/1", "http://input/1"}},
		{Inputs: []string{"http://input/1", "http://input/2"}},
		{Inputs: []string{"http://input/1"}},
		{},
	}

	assert.Exactly(t, 6, getInputsAmount(sl1), "should be 6 inputs in total")
}
