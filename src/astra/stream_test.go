package astra

import (
	"fmt"
	"m3u_merge_astra/astra/analyzer"
	"m3u_merge_astra/cfg"
	"m3u_merge_astra/util/copier"
	"m3u_merge_astra/util/logger"
	"m3u_merge_astra/util/network"
	"m3u_merge_astra/util/slice"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/stretchr/testify/assert"
	"github.com/zenizh/go-capturer"
)

func TestNewStream(t *testing.T) {
	cfg := newDefRepo().cfg.Streams
	s := NewStream(cfg, "0000", "Name", "Group", []string{"http://url"})

	expected := Stream{
		DisabledInputs: make([]string, 0),
		Enabled:        cfg.MakeNewEnabled,
		HTTPKeepActive: strconv.Itoa(cfg.NewKeepActive),
		ID:             "0000",
		Inputs:         []string{"http://url"},
		Name:           "Name",
		Type:           string(cfg.NewType),
		MarkAdded:      true,
	}
	assert.Exactly(t, expected, s, "should create this stream")

	cfg.AddGroupsToNew = true
	s = NewStream(cfg, "0000", "Name", "Group", []string{"http://url"})

	expected.Groups = map[string]string{cfg.GroupsCategoryForNew: "Group"}
	assert.Exactly(t, expected, s, "should create this stream")
}

func TestGetName(t *testing.T) {
	s := Stream{Name: "Name"}
	assert.Exactly(t, s.Name, s.GetName(), "should return this name")
}

func TestFirstGroup(t *testing.T) {
	s := Stream{}
	assert.Empty(t, s.FirstGroup(), "should return empty group")
	s = Stream{
		Groups: map[string]string{
			"Category 3": "Group 3",
			"Category 5": "Group 5",
			"Category 1": "Group 1",
			"Category 2": "Group 2",
			"Category 4": "Group 4",
		},
	}
	for i := 0; i < 10000; i++ { // Test if logic relies on unstable iteration over maps
		if ok := assert.Exactly(t, "Category 1: Group 1", s.FirstGroup(), "should return first group"); !ok {
			t.FailNow()
		}
	}
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
	s1Original := copier.TestDeep(t, s1)

	oldInputs := []string{}
	r.cfg.Streams.KeepInputHash = false
	s2, updated := s1.UpdateInput(r, "http://update/to/1", func(oldInput string) {
		oldInputs = append(oldInputs, oldInput)
	})
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected := []string{
		"http://irrelevant/from#a", "http://update/to/1", "http://update/url/1#b", "http://update/url/1#c",
	}
	assert.Exactly(t, expected, s2.Inputs, "should have these inputs")
	assert.True(t, updated, "should return true as it was updated")

	s2, updated = s1.UpdateInput(r, "http://update/url/1#b", func(oldInput string) {
		oldInputs = append(oldInputs, oldInput)
	})
	expected = []string{
		"http://irrelevant/from#a", "http://update/from/1#c", "http://update/url/1#b", "http://update/url/1#b",
	}
	assert.Exactly(t, expected, s2.Inputs, "relevant input should be updated discarding old hash")
	assert.True(t, updated, "should return true as it was updated")

	s2, updated = s1.UpdateInput(r, "http://irrelevant/to", func(oldInput string) {
		oldInputs = append(oldInputs, oldInput)
	})
	assert.Exactly(t, s1, s2, "inputs should not be updated with irrelevant URL")
	assert.False(t, updated, "should return false as it was not updated")

	r.cfg.Streams.KeepInputHash = true
	s2, updated = s1.UpdateInput(r, "http://update/url/2", func(oldInput string) {
		oldInputs = append(oldInputs, oldInput)
	})
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected = []string{
		"http://irrelevant/from#a", "http://update/from/1#c", "http://update/url/2#b", "http://update/url/1#c",
	}
	assert.Exactly(t, expected, s2.Inputs, "relevant input should be updated keeping old hash")
	assert.True(t, updated, "should return true as it was updated")

	s2, updated = s1.UpdateInput(r, "http://update/url/2#c", func(oldInput string) {
		oldInputs = append(oldInputs, oldInput)
	})
	expected = []string{
		"http://irrelevant/from#a", "http://update/from/1#c", "http://update/url/2#c&b", "http://update/url/1#c",
	}
	assert.Exactly(t, expected, s2.Inputs, "relevant input should be updated, merging hashes")
	assert.True(t, updated, "should return true as it was updated")

	s2, updated = s1.UpdateInput(r, "http://irrelevant/to", func(oldInput string) {
		t.Fail()
	})
	assert.Exactly(t, s1, s2, "inputs should not be updated with irrelevant URL")
	assert.False(t, updated, "should return false as it was not updated")

	assert.Exactly(t, []string{
		"http://update/from/1#c",
		"http://update/url/1#c",
		"http://update/url/1#b",
		"http://update/url/1#b",
	}, oldInputs, "callbacks should return these old iputs")
}

func TestHasInput(t *testing.T) {
	log := logger.New(logger.DebugLevel)

	s := Stream{Inputs: []string{"http://other/input", "http://known/input#a"}}

	assert.False(t, s.HasInput(log, "http://known/input", true), "should not contain URL without hash")
	assert.True(t, s.HasInput(log, "http://known/input#a", true), "should contain URL")

	assert.True(t, s.HasInput(log, "http://known/input", false), "should contain URL without hash")
	assert.True(t, s.HasInput(log, "http://known/input#b", false), "should contain URL with different hashes")

	assert.False(t, s.HasInput(log, "http://foreign/input", true), "should not contain URL")
	assert.False(t, s.HasInput(log, "http://foreign/input#a", true), "should not contain URL")

	assert.False(t, s.HasInput(log, "http://foreign/input", false), "should not contain URL")
	assert.False(t, s.HasInput(log, "http://foreign/input#b", false), "should not contain URL")

	s = Stream{Inputs: []string{"http://other/input#a", "http:/other/input/2#b"}}

	assert.False(t, s.HasInput(log, "http://foreign/input", true), "should not contain URL")
	assert.False(t, s.HasInput(log, "http://foreign/input#a", true), "should not contain URL")

	assert.False(t, s.HasInput(log, "http://foreign/input", false), "should not contain URL")
	assert.False(t, s.HasInput(log, "http://foreign/input#b", false), "should not contain URL")
}

func TestAddInput(t *testing.T) {
	s1 := Stream{Inputs: []string{"http://input/1", "http://input/2"}}
	s1Original := copier.TestDeep(t, s1)

	s2 := s1.AddInput("http://input/3")
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected := []string{"http://input/3", "http://input/1", "http://input/2"}
	assert.Exactly(t, expected, s2.Inputs, "should have these inputs")
}

func TestKnownInputs(t *testing.T) {
	config := cfg.NewDefCfg().Streams

	config.InputUpdateMap = []cfg.UpdateRecord{
		{From: *regexp.MustCompile("known/input/1")},
		{From: *regexp.MustCompile("known/input/2")},
	}

	s := Stream{Inputs: []string{"http://known/input/2#a", "http://other/input", "http://known/input/1"}}
	ki := s.KnownInputs(config)

	expected := []string{"http://known/input/2#a", "http://known/input/1"}
	assert.Exactly(t, expected, ki, "should have these inputs")
}

func TestInputsUpdateNote(t *testing.T) {
	cfg := cfg.NewDefCfg().Streams

	cfg.EnableOnInputUpdate = false
	s := Stream{Enabled: false}
	assert.Exactly(t, "Stream is disabled", s.InputsUpdateNote(cfg), "should return this note")
	s = Stream{Enabled: true}
	assert.Exactly(t, "", s.InputsUpdateNote(cfg), "should not return a note if enabled")

	cfg.EnableOnInputUpdate = true
	s = Stream{Enabled: false}
	assert.Exactly(t, "", s.InputsUpdateNote(cfg), "should not return a note as EnableOnInputUpdate = true")
	s = Stream{Enabled: true}
	assert.Exactly(t, "", s.InputsUpdateNote(cfg), "should not return a note as EnableOnInputUpdate = true")
}

func TestEnableStream(t *testing.T) {
	s1 := Stream{Enabled: false, MarkDisabled: true}
	s1Original := copier.TestDeep(t, s1)

	s2 := s1.Enable()
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected := Stream{Enabled: true, MarkDisabled: false}
	assert.Exactly(t, expected, s2, "should set Enabled field to true and MarkDisabled field to false")

	s1 = Stream{Enabled: false, MarkDisabled: false}
	s1Original = copier.TestDeep(t, s1)

	s2 = s1.Enable()
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected = Stream{Enabled: true, MarkDisabled: false}
	assert.Exactly(t, expected, s2, "should set Enabled field to true")

	s1 = Stream{Enabled: true, MarkDisabled: true}
	s1Original = copier.TestDeep(t, s1)

	s2 = s1.Enable()
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected = Stream{Enabled: true, MarkDisabled: false}
	assert.Exactly(t, expected, s2, "should set MarkDisabled field to false")

	s1 = Stream{Enabled: true, MarkDisabled: false}
	s1Original = copier.TestDeep(t, s1)

	s2 = s1.Enable()
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	assert.Exactly(t, s1, s2, "should not change enabled stream with MarkDisabled field set to false")
}

func TestRemoveInputsCb(t *testing.T) {
	s1 := Stream{Inputs: []string{"http://input/1", "http://input/2", "http://input/1"}}
	s1Original := copier.TestDeep(t, s1)

	count := 0
	s2 := s1.RemoveInputsCb("http://input/1", func() {
		count++
	})
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected := []string{"http://input/2"}
	assert.Exactly(t, expected, s2.Inputs, "should have these inputs")
	assert.Exactly(t, 2, count, "should remove 2 inputs")
}

func TestRemoveInputs(t *testing.T) {
	s := Stream{}
	s.removeInputs("") // Should not panic. Tested with RemoveInputsCb.
}

func TestDisableStream(t *testing.T) {
	s1 := Stream{Enabled: false, MarkDisabled: false}
	s1Original := copier.TestDeep(t, s1)

	s2 := s1.disable()
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected := Stream{Enabled: false, MarkDisabled: true}
	assert.Exactly(t, expected, s2, "should set disabled prefix flag")

	s1 = Stream{Enabled: true, MarkDisabled: false}
	s1Original = copier.TestDeep(t, s1)

	s2 = s1.disable()
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected = Stream{Enabled: false, MarkDisabled: true}
	assert.Exactly(t, expected, s2, "should set disabled prefix flag")

	s1 = Stream{Enabled: false, MarkDisabled: true}

	s2 = s1.disable()

	assert.Exactly(t, s1, s2, "should not change the stream")

	s1 = Stream{Enabled: true, MarkDisabled: true}

	s2 = s1.disable()

	expected = Stream{Enabled: false, MarkDisabled: true}
	assert.Exactly(t, expected, s2, "should disable the stream")
}

func TestRemoveDuplicatedInputsByRx(t *testing.T) {
	r := newDefRepo()

	r.cfg.Streams.RemoveDuplicatedInputsByRxList = []regexp.Regexp{
		*regexp.MustCompile(`^.*:\/\/([^#?/]*)`),     // By host
		*regexp.MustCompile(`^.*:\/\/.*?\/([^#?]*)`), // By path
	}

	s1 := Stream{Inputs: []string{
		"http://host1/path1",
		"http://host1/path2",
		"http://host1/path3",
		"http://host2/path1",
		"http://host2/path4",
		"http://host3/path5",
		"",
	}}
	s1Original := copier.TestDeep(t, s1)

	removed := []string{}
	s2 := s1.removeDuplicatedInputsByRx(r, func(input string) {
		removed = append(removed, input)
	})
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected := []string{"http://host1/path1", "http://host3/path5", ""}
	assert.Exactly(t, expected, s2.Inputs, "should remove inputs duplicated by host and path")

	expected = []string{"http://host1/path2", "http://host1/path3", "http://host2/path4", "http://host2/path1"}
	assert.Exactly(t, expected, removed, "callback should return these removed iputs")
}

func TestDisableAllButOneInputByRx(t *testing.T) {
	r := newDefRepo()

	r.cfg.Streams.DisableAllButOneInputByRxList = []regexp.Regexp{
		*regexp.MustCompile(`[#&]no_sync(&|$)`),
		*regexp.MustCompile(`[#&]i_might_stay(&|$)`),
	}

	s1 := Stream{
		Inputs:         []string{"http://input/1#abc", "http://input/1#no_sync&abc", "http://input/1#abc&i_might_stay"},
		DisabledInputs: []string{"http://input/1#def"},
	}
	s1Original := copier.TestDeep(t, s1)

	disabled := []string{}
	s2 := s1.disableAllButOneInputByRx(r.cfg.Streams, func(input string) {
		disabled = append(disabled, input)
	})
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected := []string{"http://input/1#no_sync&abc"}
	assert.Exactly(t, expected, s2.Inputs,
		"should remove all inputs except the first matching first found regular expression")

	expected = []string{"http://input/1#abc", "http://input/1#abc&i_might_stay"}
	assert.Exactly(t, expected, disabled, "callback should return these disabled iputs")

	expected = []string{"http://input/1#def", "http://input/1#abc", "http://input/1#abc&i_might_stay"}
	assert.Exactly(t, expected, s2.DisabledInputs, "should add all normal removed inputs to the list of disabled ones")

	s1 = Stream{
		Inputs: []string{
			"http://input/1#abc&i_might_stay",
			"http://input/1#abc&i_might_stay",
			"http://input/1#abc",
		},
		DisabledInputs: []string{},
	}
	s1Original = copier.TestDeep(t, s1)

	disabled = []string{}
	s2 = s1.disableAllButOneInputByRx(r.cfg.Streams, func(input string) {
		disabled = append(disabled, input)
	})
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected = []string{"http://input/1#abc&i_might_stay"}
	assert.Exactly(t, expected, s2.Inputs,
		"should remove all inputs except the first matching first found regular expression")

	expected = []string{"http://input/1#abc&i_might_stay", "http://input/1#abc"}
	assert.Exactly(t, expected, disabled, "callback should return these disabled iputs")

	expected = []string{"http://input/1#abc&i_might_stay", "http://input/1#abc"}
	assert.Exactly(t, expected, s2.DisabledInputs, "should add all normal removed inputs to the list of disabled ones")
}

func TestRemoveDisabledInputs(t *testing.T) {
	s1 := Stream{
		Inputs:         []string{"http://input/1"},
		DisabledInputs: []string{"http://input/2", "http://input/2"},
	}
	s1Original := copier.TestDeep(t, s1)

	removed := []string{}
	s2 := s1.removeDisabledInputs(func(input string) {
		removed = append(removed, input)
	})
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected := []string{"http://input/1"}
	assert.Exactly(t, expected, s2.Inputs, "should not remove enabled inputs")

	expected = []string{"http://input/2", "http://input/2"}
	assert.Exactly(t, expected, removed, "callback should return these removed iputs")

	expected = []string{}
	assert.Exactly(t, expected, s2.DisabledInputs, "should remove all disabled inputs")
}

func TestRemoveBlockedInputs(t *testing.T) {
	cfg := cfg.NewDefCfg().Streams

	cfg.InputBlacklist = []regexp.Regexp{
		*regexp.MustCompile("input/1"),
		*regexp.MustCompile("input/3"),
	}

	s1 := Stream{Inputs: []string{"http://input/1", "http://input/2", "http://input/1", "http://input/3"}}
	s1Original := copier.TestDeep(t, s1)

	removed := []string{}
	s2 := s1.removeBlockedInputs(cfg, func(input string) {
		removed = append(removed, input)
	})

	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected := []string{"http://input/2"}
	assert.Exactly(t, expected, s2.Inputs, "should have these inputs")

	expected = []string{"http://input/1", "http://input/1", "http://input/3"}
	assert.Exactly(t, expected, removed, "callback should return these removed iputs")
}

func TestHasNoInputs(t *testing.T) {
	s := Stream{}
	assert.True(t, s.hasNoInputs())

	s = Stream{Inputs: []string{}}
	assert.True(t, s.hasNoInputs())

	s = Stream{Inputs: []string{"http://input"}}
	assert.False(t, s.hasNoInputs())
}

func TestHasPrefix(t *testing.T) {
	prefix := "prefix: "

	s := Stream{Name: "Name"}
	assert.False(t, s.hasPrefix(prefix))

	s = Stream{Name: prefix + "Name"}
	assert.True(t, s.hasPrefix(prefix))

	s = Stream{Name: "Na" + prefix + "me"}
	assert.False(t, s.hasPrefix(prefix))

	s = Stream{}
	assert.False(t, s.hasPrefix(prefix))

	prefix = ""
	s = Stream{Name: "Name"}
	assert.False(t, s.hasPrefix(prefix))
}

func TestSetPrefix(t *testing.T) {
	prefix := "prefix: "

	s1 := Stream{Name: "Name"}
	s1Original := copier.TestDeep(t, s1)

	s2 := s1.setPrefix(prefix)
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected := Stream{Name: prefix + "Name"}
	assert.Exactly(t, expected, s2, "stream name should start with prefix")
}

func TestRemovePrefix(t *testing.T) {
	prefix := "prefix: "

	s1 := Stream{Name: "Name"}
	s1Original := copier.TestDeep(t, s1)

	s2 := s1.removePrefix(prefix)
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	assert.Exactly(t, s1, s2, "should not modify the stream without prefix")

	s1 = Stream{Name: prefix + "Name"}
	s1Original = copier.TestDeep(t, s1)

	s2 = s1.removePrefix(prefix)
	assert.NotSame(t, &s1, &s2, "should return copy of stream")
	assert.Exactly(t, s1Original, s1, "should not modify the source")

	expected := Stream{Name: "Name"}
	assert.Exactly(t, expected, s2, "should remove prefix")

	s1 = Stream{Name: "Na" + prefix + "me"}

	s2 = s1.removePrefix(prefix)

	assert.Exactly(t, s1, s2, "should not modify the stream with prefix string in the middle of the name")

	prefix = ""
	s1 = Stream{Name: "Name"}

	s2 = s1.removePrefix(prefix)

	assert.Exactly(t, s1, s2, "should not modify the stream with empty prefix set in config")
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

func TestRemoveNamePrefixes(t *testing.T) {
	r := newDefRepo()
	addedPrefix := r.cfg.Streams.AddedPrefix
	disabledPrefix := r.cfg.Streams.DisabledPrefix

	sl1 := []Stream{
		/* 0 */ {Name: "Name 1", MarkAdded: false, MarkDisabled: false},
		/* 1 */ {Name: disabledPrefix + "Name 2", MarkAdded: false, MarkDisabled: true},
		/* 2 */ {Name: disabledPrefix + "Name 3", MarkAdded: false, MarkDisabled: false},
		/* 3 */ {Name: addedPrefix + "Name 4", MarkAdded: false, MarkDisabled: true},
		/* 4 */ {Name: disabledPrefix + addedPrefix + addedPrefix + "Name 5", MarkAdded: true, MarkDisabled: true},
		/* 5 */ {Name: addedPrefix + disabledPrefix + "Name 6", MarkAdded: false, MarkDisabled: false},
		/* 6 */ {Name: "Na" + addedPrefix + disabledPrefix + "me 7", MarkAdded: false, MarkDisabled: false},
	}
	sl1Original := copier.TestDeep(t, sl1)

	sl2 := r.RemoveNamePrefixes(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	assert.Exactly(t, sl1[0], sl2[0], "should not change the stream without prefixes")

	expected := Stream{Name: "Name 2", MarkAdded: false, MarkDisabled: true}
	assert.Exactly(t, expected, sl2[1], "should remove disabled prefix")

	expected = Stream{Name: "Name 3", MarkAdded: false, MarkDisabled: true}
	assert.Exactly(t, expected, sl2[2], "should remove disabled prefix and set MarkDisabled to true")

	expected = Stream{Name: "Name 4", MarkAdded: true, MarkDisabled: true}
	assert.Exactly(t, expected, sl2[3], "should remove added prefix, set MarkAdded to true and ignore MarkDisabled")

	expected = Stream{Name: "Name 5", MarkAdded: true, MarkDisabled: true}
	assert.Exactly(t, expected, sl2[4], "should remove prefixes")

	expected = Stream{Name: "Name 6", MarkAdded: true, MarkDisabled: true}
	assert.Exactly(t, expected, sl2[5], "should remove prefixes and set both MarkAdded and MarkDisabled to true")

	assert.Exactly(t, sl1[6], sl2[6], "should not change the stream with prefix strings in the middle of the name")

	// Test log output
	out := capturer.CaptureStderr(func() {
		r := newDefRepo()
		addedPrefix := r.cfg.Streams.AddedPrefix

		sl1 := []Stream{{ID: "0", Name: addedPrefix + "Name 1", Groups: map[string]string{"Cat": "Grp"}}}

		_ = r.RemoveNamePrefixes(sl1)
	})
	assert.Contains(t, out, `Temporarily removing name prefix from stream: ID "0", old name "_ADDED: Name 1", `+
		`new name "Name 1", group "Cat: Grp"`)
}

func TestSort(t *testing.T) {
	r := newDefRepo()

	sl1 := []Stream{
		{Name: "C"}, {Name: "A"}, {}, {Name: "B"},
	}
	sl1Original := copier.TestDeep(t, sl1)

	sl2 := r.Sort(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	expected := []Stream{{Name: ""}, {Name: "A"}, {Name: "B"}, {Name: "C"}}
	assert.Exactly(t, expected, sl2, "should sort streams by name")
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
	sl1Original := copier.TestDeep(t, sl1)

	sl2 := r.RemoveBlockedInputs(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	expected := []string{"http://input/2"}
	assert.Exactly(t, expected, sl2[0].Inputs, "should have these inputs")

	assert.Len(t, sl2[1].Inputs, 0, "should remove all specified inputs")

	assert.Len(t, sl2[2].Inputs, 0, "should stay 0")

	// Test log output
	out := capturer.CaptureStderr(func() {
		r := newDefRepo()

		r.cfg.Streams.InputBlacklist = []regexp.Regexp{*regexp.MustCompile("input/1")}

		sl1 := []Stream{
			{ID: "0", Name: "Name 1", Groups: map[string]string{"Cat": "Grp"}, Inputs: []string{"http://input/1"}},
		}

		_ = r.RemoveBlockedInputs(sl1)
	})
	assert.Contains(t, out, `Removing blocked input from stream: ID "0", name "Name 1", group "Cat: Grp", `+
		`input "http://input/1"`)
}

func TestRemoveDuplicatedInputs(t *testing.T) {
	r := newDefRepo()

	sl1 := []Stream{
		{Inputs: []string{"http://input/2", "http://input/1", "http://input/3", "http://input/2", "http://input/3"}},
		{},
		{Inputs: []string{"http://input/4", "http://input/5", "http://input/1"}},
		{Inputs: []string{"http://input/2", "http://input/4", "http://input/6"}},
	}
	sl1Original := copier.TestDeep(t, sl1)

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

	// Test log output
	out := capturer.CaptureStderr(func() {
		r := newDefRepo()

		r.cfg.Streams.InputBlacklist = []regexp.Regexp{*regexp.MustCompile("input/1")}

		sl1 := []Stream{
			{ID: "0", Name: "Name 1", Groups: map[string]string{"Cat": "Grp"}, Inputs: []string{"http://input/1"}},
			{ID: "1", Name: "Name 2", Groups: map[string]string{"Cat": "Grp"}, Inputs: []string{"http://input/1"}},
		}

		_ = r.RemoveDuplicatedInputs(sl1)
	})
	assert.Contains(t, out, `Removing duplicated input from stream: ID "1", name "Name 2", group "Cat: Grp", `+
		`input "http://input/1"`)
}

func TestAllRemoveDuplicatedInputsByRx(t *testing.T) {
	r := newDefRepo()

	r.cfg.Streams.RemoveDuplicatedInputsByRxList = []regexp.Regexp{
		*regexp.MustCompile(`^.*:\/\/([^#?/]*)`),     // By host
		*regexp.MustCompile(`^.*:\/\/.*?\/([^#?]*)`), // By path
	}

	sl1 := []Stream{
		/* 0 */ {Inputs: []string{"http://host1/path1", "http://host2/path2", "http://host1/path3"}},
		/* 1 */ {Inputs: []string{"http://host1/path1", "http://host1/path2", "http://host2/path1"}},
		/* 2 */ {Inputs: []string{"http://host1/path1", "http://host2/path2"}},
		/* 3 */ {Inputs: []string{"http://host1/path1", "http://host2/path1"}},
	}
	sl1Original := copier.TestDeep(t, sl1)

	sl2 := r.RemoveDuplicatedInputsByRx(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	expected := []string{"http://host1/path1", "http://host2/path2"}
	assert.Exactly(t, expected, sl2[0].Inputs, "should remove inputs duplicated by host")

	expected = []string{"http://host1/path1"}
	assert.Exactly(t, expected, sl2[1].Inputs, "should remove inputs duplicated by host and path")

	assert.Exactly(t, sl1[2], sl2[2], "should not modify stream with unique inputs")

	expected = []string{"http://host1/path1"}
	assert.Exactly(t, expected, sl2[1].Inputs, "should remove inputs duplicated by path")

	// Test log output
	out := capturer.CaptureStderr(func() {
		r := newDefRepo()

		r.cfg.Streams.RemoveDuplicatedInputsByRxList = []regexp.Regexp{
			*regexp.MustCompile(`^.*:\/\/([^#?/]*)`), // By host
		}

		sl1 := []Stream{
			{
				ID:     "0",
				Name:   "Name 1",
				Groups: map[string]string{"Cat": "Grp"},
				Inputs: []string{"http://input/1", "http://input/1"},
			},
		}

		_ = r.RemoveDuplicatedInputsByRx(sl1)
	})
	assert.Contains(t, out, `Removing duplicated input per stream by regular expressions: ID "0", name "Name 1", `+
		`group "Cat: Grp", input "http://input/1"`)
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
	sl1Original := copier.TestDeep(t, sl1)

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

	// Test Streams.EnableOnInputUpdate
	r.cfg.Streams.EnableOnInputUpdate = false
	sl1 = []Stream{
		{Enabled: false, MarkDisabled: true, Name: "Name", Inputs: []string{"http://input/a"}},
		{Enabled: false, MarkDisabled: true, Name: "Name", Inputs: []string{"http://input/b"}},
	}
	sl1Original = copier.TestDeep(t, sl1)

	sl2 = r.UniteInputs(sl1)

	assert.False(t, sl2[0].Enabled, "stream should stay disabled as EnableOnInputUpdate = false")
	assert.True(t, sl2[0].MarkDisabled, "MarkDisabled should stay true as EnableOnInputUpdate = false")
	assert.False(t, sl2[1].Enabled, "stream should stay disabled as it has no new inputs")
	assert.True(t, sl2[1].MarkDisabled, "MarkDisabled should stay true as it has no new inputs")

	r.cfg.Streams.EnableOnInputUpdate = true

	sl2 = r.UniteInputs(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.True(t, sl2[0].Enabled, "stream should become enabled as EnableOnInputUpdate = true")
	assert.False(t, sl2[0].MarkDisabled, "MarkDisabled should become false as EnableOnInputUpdate = true")
	assert.False(t, sl2[1].Enabled, "stream should stay disabled as it has no new inputs")
	assert.True(t, sl2[1].MarkDisabled, "MarkDisabled should stay true as it has no new inputs")

	sl1 = []Stream{
		{Enabled: false, MarkDisabled: true, Name: "Name", Inputs: []string{"http://input/a"}},
	}

	sl2 = r.UniteInputs(sl1)

	assert.Exactly(t, sl1, sl2, "should stay the same because it was not updated")

	// Test log output
	out := capturer.CaptureStderr(func() {
		r := newDefRepo()
		r.cfg.Streams.EnableOnInputUpdate = false

		sl1 := []Stream{
			{ID: "0", Enabled: false, Name: "Name 1", Inputs: []string{"http://input/a"}},
			{ID: "1", Enabled: false, Name: "Name_1", Inputs: []string{"http://input/b"}},
		}

		_ = r.UniteInputs(sl1)
	})
	assert.Contains(t, out, `Uniting inputs of streams: from ID "1", from name "Name_1", input "http://input/b", `+
		`to ID "0", to name "Name 1", note "Stream is disabled"`)

	out = capturer.CaptureStderr(func() {
		r := newDefRepo()
		r.cfg.Streams.EnableOnInputUpdate = true

		sl1 := []Stream{
			{Enabled: false, ID: "0", Name: "Name 1", Inputs: []string{"http://input/a"}},
			{Enabled: false, ID: "1", Name: "Name_1", Inputs: []string{"http://input/b"}},
		}

		_ = r.UniteInputs(sl1)
	})
	assert.Contains(t, out, `Enabling the stream (uniting inputs of streams, enable_on_input_update is on): `+
		`ID "0", name "Name 1"`)
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
	sl1Original := copier.TestDeep(t, sl1)

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
	sl1Original = copier.TestDeep(t, sl1)

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
	sl1Original = copier.TestDeep(t, sl1)

	sl2 = r.SortInputs(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Exactly(t, sl1, sl2, "should stay the same")
}

func TestRemoveDeadInputs(t *testing.T) {
	r := newDefRepo()
	r.cfg.Streams.InputMaxConns = 100
	r.cfg.Streams.UseAnalyzer = false

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
		{
			Name:   "Name 1",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"https://127.0.0.1:5656/dead/timeout/1", "https://127.0.0.1:5656/alive/1"},
		},
		{
			Name:   "Name 2",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"http://127.0.0.1:3434/alive/2", "http://dead/no_such_host/1", "http://ignore/2"},
		},
		{
			Name:   "Name 3",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"http://127.0.0.1:3434/dead/timeout/" + strings.Repeat("x", 40), "https://ignore/1"},
		},
		{
			Name:   "Name 4",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"rtp://skip/1", "rtsp://skip/2", "file:///skip/3.ts", "http://127.0.0.1:3434/dead/404/1"},
		},
	}
	sl1Original := copier.TestDeep(t, sl1)

	httpClient := network.NewHttpClient(time.Second * 3)
	analyzerClient := analyzer.NewFake()
	sl2 := r.RemoveDeadInputs(httpClient, analyzerClient, sl1)
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

	// Test concurrency
	// On Windows may unexpectedly freeze after ~20 seconds of the test runnning.
	// Use test_remove_dead_inputs.sh

	sl1 = []Stream{
		{
			Name:   "Name 1",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"http://127.0.0.1:3434/dead/404/1", "rtp://skip/1", "http://127.0.0.1:3434/dead/404/1"},
		},
		{
			Name:   "Name 2",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"http://ignore/1", "http://127.0.0.1:3434/dead/404/2", "http://127.0.0.1:3434/dead/404/2"},
		},
		{
			Name:   "Name 3",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"rtsp://skip/2", "http://ignore/2", "rtsp://skip/2", "https://127.0.0.1:5656/alive/1"},
		},
	}
	sl1Original = copier.TestDeep(t, sl1)

	httpClient = network.NewFakeHttpClient(time.Second * 3)

	for i := 0; i < 10000; i++ {
		sl2 = r.RemoveDeadInputs(httpClient, analyzerClient, sl1)
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

	// Test log output
	out := capturer.CaptureStderr(func() {
		r := newDefRepo()
		r.cfg.Streams.UseAnalyzer = false

		sl1 := []Stream{{
			ID:     "0",
			Name:   "Name 1",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"https://127.0.0.1:5656/dead/timeout/1"}},
		}

		httpClient := network.NewHttpClient(time.Second * 3)
		analyzerClient := analyzer.NewFake()
		_ = r.RemoveDeadInputs(httpClient, analyzerClient, sl1)
	})
	msg := `Start checking input: stream ID "0", stream name "Name 1", stream index "0", ` +
		`input "https://127.0.0.1:5656/dead/timeout/1"`
	assert.Contains(t, out, msg)
	msg = `Removing dead input from stream: ID "0", name "Name 1", group "Cat: Grp", ` +
		`input "https://127.0.0.1:5656/dead/timeout/1", reason "Timeout"`
	assert.Contains(t, out, msg)
	msg = `End checking input: stream ID "0", stream name "Name 1", stream index "0", ` +
		`input "https://127.0.0.1:5656/dead/timeout/1"`
	assert.Contains(t, out, msg)
}

func TestAnalyzerRemoveDeadInputs(t *testing.T) {
	r := newDefRepo()
	r.cfg.Streams.InputMaxConns = 100
	r.cfg.Streams.UseAnalyzer = true
	r.cfg.Streams.AnalyzerAudioOnlyBitrateThreshold = 100
	r.cfg.Streams.AnalyzerVideoOnlyBitrateThreshold = 200
	r.cfg.Streams.AnalyzerBitrateThreshold = 300
	r.cfg.Streams.AnalyzerCCErrorsThreshold = 10
	r.cfg.Streams.AnalyzerPCRErrorsThreshold = 20
	r.cfg.Streams.AnalyzerPESErrorsThreshold = 30

	r.cfg.Streams.DeadInputsCheckBlacklist = []regexp.Regexp{
		*regexp.MustCompile(`ignore/1`),
	}
	sl1 := []Stream{
		{
			Name:   "Name 1",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"http://dead/audio/50", "http://alive/audio/100", "http://dead/cc/15",
				"http://alive/cc/10"},
		},
		{
			Name:   "Name 2",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"https://dead/video/150", "udp://alive/video/210", "https://dead/pcr/25",
				"https://alive/pcr/20"},
		},
		{
			Name:   "Name 3",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"rtp://dead/full/250", "rtsp://alive/full/400", "udp://dead/pes/35", "rtp://alive/pes/30"},
		},
		{
			Name:   "Name 4",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"dvb://0000#pnr=0000&cam=0000", "xxx://unknown/1", "http://ignore/1"},
		},
	}
	sl1Original := copier.TestDeep(t, sl1)

	httpClient := network.NewFakeHttpClient(time.Second * 3)
	analyzerClient := analyzer.NewFake()
	analyzerClient.AddResult("http://dead/audio/50", analyzer.Result{HasAudio: true, Bitrate: 50})
	analyzerClient.AddResult("http://alive/audio/100", analyzer.Result{HasAudio: true, Bitrate: 100})
	analyzerClient.AddResult("http://dead/cc/15", analyzer.Result{CCErrors: 15, Bitrate: 1000})
	analyzerClient.AddResult("http://alive/cc/10", analyzer.Result{CCErrors: 10, Bitrate: 1000})
	analyzerClient.AddResult("https://dead/video/150", analyzer.Result{HasVideo: true, Bitrate: 150})
	analyzerClient.AddResult("udp://alive/video/210", analyzer.Result{HasVideo: true, Bitrate: 210})
	analyzerClient.AddResult("https://dead/pcr/25", analyzer.Result{PCRErrors: 25, Bitrate: 1000})
	analyzerClient.AddResult("https://alive/pcr/20", analyzer.Result{PCRErrors: 20, Bitrate: 1000})
	analyzerClient.AddResult("rtp://dead/full/250", analyzer.Result{HasAudio: true, HasVideo: true, Bitrate: 250})
	analyzerClient.AddResult("rtsp://alive/full/400", analyzer.Result{HasAudio: true, HasVideo: true, Bitrate: 400})
	analyzerClient.AddResult("udp://dead/pes/35", analyzer.Result{PESErrors: 35, Bitrate: 1000})
	analyzerClient.AddResult("rtp://alive/pes/30", analyzer.Result{PESErrors: 30, Bitrate: 1000})

	sl2 := r.RemoveDeadInputs(httpClient, analyzerClient, sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	expected := []string{"http://alive/audio/100", "http://alive/cc/10"}
	assert.Exactly(t, expected, sl2[0].Inputs, "should have these inputs")

	expected = []string{"udp://alive/video/210", "https://alive/pcr/20"}
	assert.Exactly(t, expected, sl2[1].Inputs, "should have these inputs")

	expected = []string{"rtsp://alive/full/400", "rtp://alive/pes/30"}
	assert.Exactly(t, expected, sl2[2].Inputs, "should have these inputs")

	assert.Exactly(t, sl1[3].Inputs, sl2[3].Inputs, "should not remove inputs with unsupported protocols or ignored")

	// Test negative errors threshold
	r.cfg.Streams.AnalyzerCCErrorsThreshold = -1
	r.cfg.Streams.AnalyzerPCRErrorsThreshold = -1
	r.cfg.Streams.AnalyzerPESErrorsThreshold = -1

	sl1 = []Stream{
		{
			Name:   "Name 1",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"http://alive/cc/0", "http://alive/pcr/1", "http://alive/pes/10"},
		},
	}
	sl1Original = copier.TestDeep(t, sl1)

	analyzerClient.AddResult("http://alive/cc/0", analyzer.Result{CCErrors: 0, Bitrate: 1000})
	analyzerClient.AddResult("http://alive/pcr/1", analyzer.Result{PCRErrors: 1, Bitrate: 1000})
	analyzerClient.AddResult("http://alive/pes/10", analyzer.Result{PESErrors: 10, Bitrate: 1000})

	sl2 = r.RemoveDeadInputs(httpClient, analyzerClient, sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	assert.Exactly(t, sl1[0].Inputs, sl2[0].Inputs, "should not remove inputs with errors if thresholds are negative")

	// Test log output
	out := capturer.CaptureStderr(func() {
		r := newDefRepo()
		r.cfg.Streams.UseAnalyzer = true
		r.cfg.Streams.AnalyzerAudioOnlyBitrateThreshold = 100

		sl1 := []Stream{{
			ID:     "0",
			Name:   "Name 1",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"https://dead/audio/50"}},
		}

		httpClient := network.NewFakeHttpClient(time.Second * 3)
		analyzerClient := analyzer.NewFake()
		analyzerClient.AddResult("https://dead/audio/50", analyzer.Result{HasAudio: true, Bitrate: 50})

		_ = r.RemoveDeadInputs(httpClient, analyzerClient, sl1)
	})
	msg := `Start checking input: stream ID "0", stream name "Name 1", stream index "0", ` +
		`input "https://dead/audio/50"`
	assert.Contains(t, out, msg)
	msg = `Removing dead input from stream: ID "0", name "Name 1", group "Cat: Grp", ` +
		`input "https://dead/audio/50", reason "Bitrate 50 < 100"`
	assert.Contains(t, out, msg)
	msg = `End checking input: stream ID "0", stream name "Name 1", stream index "0", ` +
		`input "https://dead/audio/50"`
	assert.Contains(t, out, msg)
}

func TestProgressRemoveDeadInputs(t *testing.T) {
	log := logger.New(logger.DebugLevel)

	handleSleep2Sec := func(w http.ResponseWriter, req *http.Request) {
		log.Debugf("Got request to %v", req.URL)
		time.Sleep(time.Second * 2)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/sleep/2sec", handleSleep2Sec)

	// Run http & https servers as subset of current test to be able to fail it from another goroutines (servers) if any
	// server returns error
	var httpSrv, httpsSrv *http.Server
	t.Run("http_server", func(t *testing.T) {
		httpSrv, httpsSrv = network.NewHttpServer(mux, 3434, 5656, func(err error) {
			if !errors.Is(err, http.ErrServerClosed) {
				// Not using logging from testing.T or else message will not be displayed
				log.Errorf("Test server stopped with non-standard error: %v", err)
				t.FailNow()
			}
		})
	})
	defer httpSrv.Close()
	defer httpsSrv.Close()

	// Test log output
	// On Windows may unexpectedly freeze after ~20 seconds of the test runnning.
	// Use test_progress_remove_dead_inputs.sh
	out := capturer.CaptureStderr(func() {
		r := newDefRepo()
		r.cfg.Streams.InputMaxConns = 1
		r.cfg.Streams.UseAnalyzer = false

		sl1 := []Stream{{Inputs: slice.Filled("http://127.0.0.1:3434/sleep/2sec", 20)}}

		httpClient := network.NewFakeHttpClient(time.Second * 3)
		analyzerClient := analyzer.NewFake()

		_ = r.RemoveDeadInputs(httpClient, analyzerClient, sl1)
	})
	assert.Contains(t, out, `Removing dead inputs from streams: progress "14 / 20 (70%)"`)
}

func TestDisableDeadInputs(t *testing.T) {
	r := newDefRepo()
	r.cfg.Streams.InputMaxConns = 1
	r.cfg.Streams.UseAnalyzer = false

	// Create request handlers
	handleAlive := func(w http.ResponseWriter, req *http.Request) {
		r.log.Debugf("Got request to %v", req.URL)
		w.WriteHeader(200)
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/alive/", handleAlive)

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
	}
	sl1 := []Stream{
		{
			Name:   "Name 1",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{
				"http://dead/no_such_host/1",
				"http://dead/no_such_host/2",
				"http://127.0.0.1:3434/alive/1",
			},
			DisabledInputs: []string{"http://dead/existing/1"},
		},
		{
			Name:   "Name 2",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"http://127.0.0.1:3434/alive/2", "http://dead/no_such_host/3", "http://ignore/1"},
		},
	}
	sl1Original := copier.TestDeep(t, sl1)

	httpClient := network.NewHttpClient(time.Second * 3)
	analyzerClient := analyzer.NewFake()
	sl2 := r.DisableDeadInputs(httpClient, analyzerClient, sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	expected := []string{"http://127.0.0.1:3434/alive/1"}
	assert.Exactly(t, expected, sl2[0].Inputs, "should have these inputs")
	expected = []string{"http://dead/existing/1", "http://dead/no_such_host/1", "http://dead/no_such_host/2"}
	assert.Exactly(t, expected, sl2[0].DisabledInputs, "should have these disabled inputs")

	expected = []string{"http://127.0.0.1:3434/alive/2", "http://ignore/1"}
	assert.Exactly(t, expected, sl2[1].Inputs, "should have these inputs")
	expected = []string{"http://dead/no_such_host/3"}
	assert.Exactly(t, expected, sl2[1].DisabledInputs, "should have these disabled inputs")

	// Test log output
	out := capturer.CaptureStderr(func() {
		r := newDefRepo()
		r.cfg.Streams.UseAnalyzer = false

		sl1 := []Stream{{
			ID:     "0",
			Name:   "Name 1",
			Groups: map[string]string{"Cat": "Grp"},
			Inputs: []string{"http://dead/no_such_host/1"}},
		}

		httpClient := network.NewHttpClient(time.Second * 3)
		analyzerClient := analyzer.NewFake()
		_ = r.DisableDeadInputs(httpClient, analyzerClient, sl1)
	})
	msg := `Disabling dead input of stream: ID "0", name "Name 1", group "Cat: Grp", ` +
		`input "http://dead/no_such_host/1", reason "No such host"`
	assert.Contains(t, out, msg)
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
			Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://known/input/1#x", "http://other/input/1"},
		},
		{ // Index 1. Known name 1
			Name:   "Known name 1",
			Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://other/input/1#a", "http://other/input/2#x"},
		},
		{ // Index 2. Known group 2
			Name:   "Other name",
			Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
			Inputs: []string{"http://other/input/1#a&d", "http://other/input/2"},
		},
		{ // Index 3. Known inputs 2 and 1
			Name:   "Other name",
			Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://known/input/2#x", "http://known/input/1"},
		},
		{ // Index 4. Known name 2
			Name:   "Known name 2",
			Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://other/input/1"},
		},
		{ // Index 5. Known group 1
			Name:   "Other name",
			Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
			Inputs: []string{"http://other/input/1#c", "http://other/input/2#x"},
		},
		{ // Index 6. No matches
			Name:   "Other name",
			Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://other/input/2", "http://other/input/1#a"},
		},
		{ // Index 7. Matches by every parameter
			Name:   "Known name 1",
			Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
			Inputs: []string{"http://known/input/2#x", "http://other/input/1", "http://known/input/1"},
		},
		{ // Index 8. Matches by group 1 and input 1
			Name:   "Other name",
			Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
			Inputs: []string{"http://known/input/1", "http://other/input/1"},
		},
	}
	sl1Original := copier.TestDeep(t, sl1)

	sl2 := r.AddHashes(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	expected := Stream{ // Index 0. Known input 1
		Name:   "Other name",
		Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
		Inputs: []string{"http://known/input/1#x&e", "http://other/input/1"},
	}
	assert.Exactly(t, expected, sl2[0], "inputs matching only by StreamInputToInputHashMap should get hashes only for"+
		"the exact inputs")

	expected = Stream{ // Index 1. Known name 1
		Name:   "Known name 1",
		Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
		Inputs: []string{"http://other/input/1#a", "http://other/input/2#x&a"},
	}
	assert.Exactly(t, expected, sl2[1], "should add hash to every matching input")

	expected = Stream{ // Index 2. Known group 2
		Name:   "Other name",
		Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
		Inputs: []string{"http://other/input/1#a&d", "http://other/input/2#d"},
	}
	assert.Exactly(t, expected, sl2[2], "should add hash to every matching input")

	expected = Stream{ // Index 3. Known inputs 2 and 1
		Name:   "Other name",
		Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
		Inputs: []string{"http://known/input/2#x&f", "http://known/input/1#e"},
	}
	assert.Exactly(t, expected, sl2[3], "should add hash to every matching input")

	expected = Stream{ // Index 4. Known name 2
		Name:   "Known name 2",
		Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
		Inputs: []string{"http://other/input/1#b"},
	}
	assert.Exactly(t, expected, sl2[4], "should add hash to matching input")

	expected = Stream{ // Index 5. Known group 1
		Name:   "Other name",
		Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
		Inputs: []string{"http://other/input/1#c", "http://other/input/2#x&c"},
	}
	assert.Exactly(t, expected, sl2[5], "should add hash to every matching input")

	// Index 6. No matches
	assert.Exactly(t, sl1[6], sl2[6], "should not modify stream with no matches")

	expected = Stream{ // Index 7. Matches by every parameter
		Name:   "Known name 1",
		Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
		Inputs: []string{"http://known/input/2#x&f&a&d", "http://other/input/1#a&d", "http://known/input/1#e&a&d"},
	}
	assert.Exactly(t, expected, sl2[7], "should add hash to every matching input by every parameter")

	expected = Stream{ // Index 8. Matches by group 1 and input 1
		Name:   "Other name",
		Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
		Inputs: []string{"http://known/input/1#e&c", "http://other/input/1#c"},
	}
	assert.Exactly(t, expected, sl2[8], "should add hash to every matching input by every parameter")

	// Test log output
	out := capturer.CaptureStderr(func() {
		r := newDefRepo()
		r.cfg.Streams.NameToInputHashMap = []cfg.HashAddRule{
			{By: *regexp.MustCompile(`Name 1`), Hash: "a"},
		}

		sl1 := []Stream{
			{ID: "0", Name: "Name 1", Groups: map[string]string{"Cat": "Grp"}, Inputs: []string{"http://input/1"}},
		}

		_ = r.AddHashes(sl1)
	})
	assert.Contains(t, out, `Adding hash to input of stream: ID "0", name "Name 1", group "Cat: Grp", hash "a", `+
		`result "http://input/1#a"`)
}

func TestAllDisableAllButOneInputByRx(t *testing.T) {
	r := newDefRepo()

	r.cfg.Streams.DisableAllButOneInputByRxList = []regexp.Regexp{
		*regexp.MustCompile(`[#&]no_sync(&|$)`),
		*regexp.MustCompile(`[#&]i_might_stay(&|$)`),
	}

	sl1 := []Stream{
		{
			Inputs: []string{
				"http://input/1#abc",
				"http://input/1#no_sync&abc",
				"http://input/1#abc&i_might_stay",
			},
			DisabledInputs: []string{"http://input/1#def"},
		},
		{
			Inputs:         []string{"http://input/1#abc"},
			DisabledInputs: []string{"http://input/1#no_sync&abc"},
		},
		{
			Inputs: []string{
				"http://input/1#abc&i_might_stay",
				"http://input/1#abc&i_might_stay",
				"http://input/1#abc",
			},
			DisabledInputs: []string{},
		},
	}
	sl1Original := copier.TestDeep(t, sl1)

	sl2 := r.DisableAllButOneInputByRx(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	expected := []Stream{
		{
			Inputs:         []string{"http://input/1#no_sync&abc"},
			DisabledInputs: []string{"http://input/1#def", "http://input/1#abc", "http://input/1#abc&i_might_stay"},
		},
		{
			Inputs:         []string{"http://input/1#abc"},
			DisabledInputs: []string{"http://input/1#no_sync&abc"},
		},
		{
			Inputs:         []string{"http://input/1#abc&i_might_stay"},
			DisabledInputs: []string{"http://input/1#abc&i_might_stay", "http://input/1#abc"},
		},
	}
	assert.Exactly(t, expected, sl2, "should return these streams")

	// Test log output
	out := capturer.CaptureStderr(func() {
		r := newDefRepo()

		r.cfg.Streams.DisableAllButOneInputByRxList = []regexp.Regexp{
			*regexp.MustCompile(`[#&]no_sync(&|$)`),
		}

		sl1 := []Stream{
			{
				ID:     "0",
				Name:   "Name 1",
				Groups: map[string]string{"Cat": "Grp"},
				Inputs: []string{"http://input/1#abc", "http://input/1#no_sync&abc"},
			},
		}

		_ = r.DisableAllButOneInputByRx(sl1)
	})
	assert.Contains(t, out, `Disabling other input per stream by regular expressions: ID "0", name "Name 1", `+
		`group "Cat: Grp", input "http://input/1#abc"`)
}

func TestAllRemoveDisabledInputs(t *testing.T) {
	r := newDefRepo()

	sl1 := []Stream{
		{
			Inputs:         []string{"http://input/1", "http://input/2"},
			DisabledInputs: []string{"http://input/3", "http://input/4"},
		},
		{
			Inputs:         []string{"http://input/5"},
			DisabledInputs: []string{"http://input/6"},
		},
		{
			Inputs:         []string{},
			DisabledInputs: []string{},
		},
	}
	sl1Original := copier.TestDeep(t, sl1)

	sl2 := r.RemoveDisabledInputs(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	expected := []Stream{
		{
			Inputs:         []string{"http://input/1", "http://input/2"},
			DisabledInputs: []string{},
		},
		{
			Inputs:         []string{"http://input/5"},
			DisabledInputs: []string{},
		},
		{
			Inputs:         []string{},
			DisabledInputs: []string{},
		},
	}
	assert.Exactly(t, expected, sl2, "should return these streams")

	// Test log output
	out := capturer.CaptureStderr(func() {
		r := newDefRepo()

		sl1 := []Stream{
			{
				ID:             "0",
				Name:           "Name 1",
				Groups:         map[string]string{"Cat": "Grp"},
				DisabledInputs: []string{"http://input/1"},
			},
		}

		_ = r.RemoveDisabledInputs(sl1)
	})
	assert.Contains(t, out, `Removing disabled input: ID "0", name "Name 1", group "Cat: Grp", input "http://input/1"`)
}

func TestSetKeepActive(t *testing.T) {
	r := newDefRepo()
	r.cfg.Streams.NameToKeepActiveMap = []cfg.KeepActiveAddRule{
		{By: *regexp.MustCompile(`Known name 1`), KeepActive: 0},
		{By: *regexp.MustCompile(`Known name 2`), KeepActive: 1},
	}
	r.cfg.Streams.GroupToKeepActiveMap = []cfg.KeepActiveAddRule{
		{By: *regexp.MustCompile(`Known group 1`), KeepActive: 2},
		{By: *regexp.MustCompile(`Known group 2`), KeepActive: 3},
	}
	r.cfg.Streams.InputToKeepActiveMap = []cfg.KeepActiveAddRule{
		{By: *regexp.MustCompile(`http://known/input/1`), KeepActive: 4},
		{By: *regexp.MustCompile(`http://known/input/2`), KeepActive: 5},
	}

	sl1 := []Stream{
		{ // Index 0. Known input 1
			Name:   "Other name",
			Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://known/input/1", "http://other/input/1"},
		},
		{ // Index 1. Known name 1
			Name:   "Known name 1",
			Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://other/input/1", "http://other/input/2"},
		},
		{ // Index 2. Known group 2
			Name:   "Other name",
			Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
			Inputs: []string{"http://other/input/1", "http://other/input/2"},
		},
		{ // Index 3. Known inputs 2 and 1
			Name:   "Other name",
			Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://known/input/2", "http://known/input/1"},
		},
		{ // Index 4. Known name 2 and group 1
			Name:   "Known name 2",
			Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
			Inputs: []string{"http://other/input/1"},
		},
		{ // Index 5. Known group 1
			Name:   "Other name",
			Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
			Inputs: []string{"http://other/input/1", "http://other/input/2"},
		},
		{ // Index 6. No matches
			Name:   "Other name",
			Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
			Inputs: []string{"http://other/input/2", "http://other/input/1"},
		},
		{ // Index 7. Matches by every parameter
			Name:   "Known name 1",
			Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
			Inputs: []string{"http://other/input/1", "http://known/input/2"},
		},
		{ // Index 8. Matches by group 1 and input 1
			Name:   "Other name",
			Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
			Inputs: []string{"http://known/input/1", "http://other/input/1"},
		},
	}
	sl1Original := copier.TestDeep(t, sl1)

	sl2 := r.SetKeepActive(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	expected := Stream{ // Index 0. Known input 1
		Name:           "Other name",
		Groups:         map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
		HTTPKeepActive: "4",
		Inputs:         []string{"http://known/input/1", "http://other/input/1"},
	}
	assert.Exactly(t, expected, sl2[0], "should set HTTPKeepActive")

	expected = Stream{ // Index 1. Known name 1
		Name:           "Known name 1",
		Groups:         map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
		HTTPKeepActive: "0",
		Inputs:         []string{"http://other/input/1", "http://other/input/2"},
	}
	assert.Exactly(t, expected, sl2[1], "should set HTTPKeepActive to 0")

	expected = Stream{ // Index 2. Known group 2
		Name:           "Other name",
		Groups:         map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
		HTTPKeepActive: "3",
		Inputs:         []string{"http://other/input/1", "http://other/input/2"},
	}
	assert.Exactly(t, expected, sl2[2], "should set HTTPKeepActive")

	expected = Stream{ // Index 3. Known inputs 2 and 1
		Name:           "Other name",
		Groups:         map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Other group"},
		HTTPKeepActive: "4",
		Inputs:         []string{"http://known/input/2", "http://known/input/1"},
	}
	assert.Exactly(t, expected, sl2[3], "should set HTTPKeepActive by first found rule by inputs")

	expected = Stream{ // Index 4. Known name 2 and group 1
		Name:           "Known name 2",
		Groups:         map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
		HTTPKeepActive: "1",
		Inputs:         []string{"http://other/input/1"},
	}
	assert.Exactly(t, expected, sl2[4], "should set HTTPKeepActive by name")

	expected = Stream{ // Index 5. Known group 1
		Name:           "Other name",
		Groups:         map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
		HTTPKeepActive: "2",
		Inputs:         []string{"http://other/input/1", "http://other/input/2"},
	}
	assert.Exactly(t, expected, sl2[5], "should set HTTPKeepActive")

	// Index 6. No matches
	assert.Exactly(t, sl1[6], sl2[6], "should not modify stream with no matches")

	expected = Stream{ // Index 7. Matches by every parameter
		Name:           "Known name 1",
		Groups:         map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Known group 2"},
		HTTPKeepActive: "5",
		Inputs:         []string{"http://other/input/1", "http://known/input/2"},
	}
	assert.Exactly(t, expected, sl2[7], "should set HTTPKeepActive by first found rule by inputs")

	expected = Stream{ // Index 8. Matches by group 1 and input 1
		Name:           "Other name",
		Groups:         map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Known group 1"},
		HTTPKeepActive: "4",
		Inputs:         []string{"http://known/input/1", "http://other/input/1"},
	}
	assert.Exactly(t, expected, sl2[8], "should set HTTPKeepActive by first found rule by inputs")

	// Test log output
	out := capturer.CaptureStderr(func() {
		r := newDefRepo()
		r.cfg.Streams.NameToKeepActiveMap = []cfg.KeepActiveAddRule{
			{By: *regexp.MustCompile(`Name 1`), KeepActive: 1},
		}

		sl1 := []Stream{{ID: "0", Name: "Name 1", Groups: map[string]string{"Cat": "Grp"}}}

		_ = r.SetKeepActive(sl1)
	})
	assert.Contains(t, out, `Setting keep active on stream: ID "0", name "Name 1", group "Cat: Grp", keep active "1"`)
}

func TestRemoveWithoutInputs(t *testing.T) {
	r := newDefRepo()

	sl1 := []Stream{
		{Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Group"}},
		{Enabled: true, Name: "Name"},
		{Enabled: true, Inputs: []string{"http://input/1", "http://input/2"}},
		{Enabled: false, Name: r.cfg.Streams.DisabledPrefix + "Name"},
		{Inputs: []string{"http://input"}},
	}
	sl1Original := copier.TestDeep(t, sl1)

	sl2 := r.RemoveWithoutInputs(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	expected := []Stream{
		{Groups: map[string]string{r.cfg.Streams.GroupsCategoryForNew: "Group"}, Remove: true},
		{Enabled: true, Name: "Name", Remove: true},
		{Enabled: true, Inputs: []string{"http://input/1", "http://input/2"}},
		{Enabled: false, Name: r.cfg.Streams.DisabledPrefix + "Name", Remove: true},
		{Inputs: []string{"http://input"}},
	}

	assert.Exactly(t, expected, sl2, "should remove streams without inputs")

	// Test log output
	out := capturer.CaptureStderr(func() {
		r := newDefRepo()

		sl1 := []Stream{
			{ID: "0", Name: "Name 1", Groups: map[string]string{"Cat": "Grp"}},
			{ID: "1", Name: "Name 2", Groups: map[string]string{"Cat": "Grp"}, Remove: true},
		}

		_ = r.RemoveWithoutInputs(sl1)
	})
	assert.Contains(t, out, `Removing stream without inputs: ID "0", name "Name 1", group "Cat: Grp"`)
	assert.NotContains(t, out, `Removing stream without inputs: ID "1", name "Name 2", group "Cat: Grp"`)
}

func TestDisableWithoutInputs(t *testing.T) {
	r := newDefRepo()

	sl1 := []Stream{
		/* 0 */ {MarkDisabled: false, Enabled: true, Name: "Name", Inputs: []string{"http://input"}},
		/* 1 */ {MarkDisabled: false, Enabled: true, Name: "Name", Inputs: []string{}},
		/* 2 */ {MarkDisabled: false, Enabled: false, Name: "Name", Inputs: []string{"http://input"}},
		/* 3 */ {MarkDisabled: false, Enabled: false, Name: "Name"},

		/* 4 */ {MarkDisabled: true, Enabled: true, Name: "Name", Inputs: []string{"http://input"}},
		/* 5 */ {MarkDisabled: true, Enabled: true, Name: "Name", Inputs: []string{}},
		/* 6 */ {MarkDisabled: true, Enabled: false, Name: "Name", Inputs: []string{"http://input"}},
		/* 7 */ {MarkDisabled: true, Enabled: false, Name: "Name"},
	}
	sl1Original := copier.TestDeep(t, sl1)

	sl2 := r.DisableWithoutInputs(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	assert.Exactly(t, sl1[0], sl2[0], "should not modify the stream with inputs")

	expected := Stream{MarkDisabled: true, Enabled: false, Name: "Name", Inputs: []string{}}
	assert.Exactly(t, expected, sl2[1], "should disable stream without inputs and set MarkDisabled to true")

	assert.Exactly(t, sl1[2], sl2[2], "should not modify disabled stream or stream with inputs")

	assert.Exactly(t, sl1[3], sl2[3], "should not modify disabled streams")

	assert.Exactly(t, sl1[4], sl2[4], "should not modify the stream with inputs")

	expected = Stream{MarkDisabled: true, Enabled: false, Name: "Name", Inputs: []string{}}
	assert.Exactly(t, expected, sl2[5], "should disable stream without inputs")

	assert.Exactly(t, sl1[6], sl2[6], "should not modify disabled stream or stream with inputs")

	assert.Exactly(t, sl1[7], sl2[7], "should not modify disabled streams")

	// Test log output
	out := capturer.CaptureStderr(func() {
		r := newDefRepo()

		sl1 := []Stream{{ID: "0", Enabled: true, Name: "Name 1", Groups: map[string]string{"Cat": "Grp"}}}

		_ = r.DisableWithoutInputs(sl1)
	})
	assert.Contains(t, out, `Disabling stream without inputs: ID "0", name "Name 1", group "Cat: Grp"`)
}

func TestAddNamePrefixes(t *testing.T) {
	r := newDefRepo()

	sl1 := []Stream{
		/* 0 */ {MarkAdded: false, MarkDisabled: false, Name: "Name_1"},
		/* 1 */ {MarkAdded: true, MarkDisabled: false, Name: "Name_2"},
		/* 2 */ {MarkAdded: false, MarkDisabled: true, Name: "Name_3"},
		/* 3 */ {MarkAdded: true, MarkDisabled: true, Name: "Name_4"},
	}
	sl1Original := copier.TestDeep(t, sl1)

	sl2 := r.AddNamePrefixes(sl1)
	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	assert.Exactly(t, sl1[0], sl2[0], "should not modify the stream with MarkAdded and MarkDisabled set to false")

	expected := Stream{MarkAdded: true, MarkDisabled: false, Name: r.cfg.Streams.AddedPrefix + "Name_2"}
	assert.Exactly(t, expected, sl2[1], "should add added prefix to the name")

	expected = Stream{MarkAdded: false, MarkDisabled: true, Name: r.cfg.Streams.DisabledPrefix + "Name_3"}
	assert.Exactly(t, expected, sl2[2], "should add disabled prefix to the name")

	expected = Stream{MarkAdded: true, MarkDisabled: true,
		Name: r.cfg.Streams.DisabledPrefix + r.cfg.Streams.AddedPrefix + "Name_4"}
	assert.Exactly(t, expected, sl2[3], "should add both disabled and added prefixes to the name")

	// Check if logs are not printed if prefixes are empty
	out := capturer.CaptureStderr(func() {
		r := newDefRepo()
		r.cfg.Streams.AddedPrefix = ""
		r.cfg.Streams.DisabledPrefix = ""

		sl1 := []Stream{{MarkAdded: true, Name: "Name_1"}}

		_ = r.AddNamePrefixes(sl1)
	})
	assert.NotContains(t, out, "Name_1")

	// Check if logs are not printed if prefixes are empty
	out = capturer.CaptureStderr(func() {
		r := newDefRepo()

		sl1 := []Stream{{ID: "0", MarkAdded: true, Name: "Name_1", Groups: map[string]string{"Cat": "Grp"}}}

		_ = r.AddNamePrefixes(sl1)
	})
	assert.Contains(t, out, fmt.Sprintf(`Adding name prefix to stream: ID "0", old name "Name_1", `+
		`new name "%vName_1", group "Cat: Grp"`, r.cfg.Streams.AddedPrefix))
}

func TestChangedStreams(t *testing.T) {
	r := newDefRepo()

	sl1 := []Stream{
		{Name: "Stream 1", ID: "0001", Enabled: true, Inputs: []string{"A", "B"}},
		{Name: "Stream 2", ID: "0002", Enabled: false, Inputs: []string{"C", "D"}},
		{Name: "Stream 3", ID: "0003", Enabled: false, Inputs: []string{"E", "F"}},
		{Name: "Stream 4", ID: "0004", Enabled: true, Inputs: []string{"G", "H"}, Groups: map[string]string{"A": "B"}},
		{Name: "Stream 5", ID: "0005"},
		{Name: "Stream 6", ID: "0006", Enabled: true, Inputs: []string{"I", "J"}, MarkAdded: true, MarkDisabled: false},
	}
	sl1Original := copier.TestDeep(t, sl1)

	sl2 := []Stream{
		// No changes
		{Name: "Stream 1", ID: "0001", Enabled: true, Inputs: []string{"A", "B"}},
		// Changed inputs
		{Name: "Stream 2", ID: "0002", Enabled: false, Inputs: []string{"C2", "D"}},
		// No changes
		{Name: "Stream 3", ID: "0003", Enabled: false, Inputs: []string{"E", "F"}},
		// Changed groups
		{Name: "Stream 4", ID: "0004", Enabled: true, Inputs: []string{"G", "H"}, Groups: map[string]string{"C": "D"}},
		// New
		{Name: "Stream 7", ID: "0007", Enabled: true, Inputs: []string{"K", "L"}, Groups: map[string]string{"E": "F"}},
		// Changed name
		{Name: "Stream 5*", ID: "0005"},
		// Changed MarkAdded / MarkDisabled (no changes)
		{Name: "Stream 6", ID: "0006", Enabled: true, Inputs: []string{"I", "J"}, MarkAdded: false, MarkDisabled: true},
		// New
		{Name: "Stream 8", ID: "0008", Enabled: false, DisabledInputs: []string{"A", "B"}},
	}
	sl2Original := copier.TestDeep(t, sl2)

	actual := r.ChangedStreams(sl1, sl2)

	assert.NotSame(t, &sl1, &actual, "should return copy of streams")
	assert.NotSame(t, &sl2, &actual, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source")
	assert.Exactly(t, sl2Original, sl2, "should not modify the source")

	expected := []Stream{
		{Name: "Stream 2", ID: "0002", Enabled: false, Inputs: []string{"C2", "D"}},
		{Name: "Stream 4", ID: "0004", Enabled: true, Inputs: []string{"G", "H"}, Groups: map[string]string{"C": "D"}},
		{Name: "Stream 7", ID: "0007", Enabled: true, Inputs: []string{"K", "L"}, Groups: map[string]string{"E": "F"}},
		{Name: "Stream 5*", ID: "0005"},
		{Name: "Stream 8", ID: "0008", Enabled: false, DisabledInputs: []string{"A", "B"}},
	}
	assert.Exactly(t, expected, actual, "should return that changed streams")
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
