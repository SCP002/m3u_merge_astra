package m3u

import (
	"m3u_merge_astra/util/copier"
	"regexp"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/utahta/go-openuri"
)

func TestGetName(t *testing.T) {
	ch := Channel{Name: "Name"}
	assert.Exactly(t, ch.Name, ch.GetName(), "should return this name")
}

func TestReplaceGroup(t *testing.T) {
	r := newDefRepo()

	r.cfg.M3U.ChannGroupMap = map[string]string{
		"Group 1":      "Group 1",
		"From Group 2": "To Group 2",
	}

	c1 := Channel{Name: "From Group 2", Group: "From Group 2", URL: "From Group 2"}
	c1Original := copier.TDeep(t, c1)
	c2 := c1.replaceGroup(r)

	assert.NotSame(t, &c1, &c2, "should return copy of channel")
	assert.Exactly(t, c1Original, c1, "should not modify the source")

	expected := Channel{Name: "From Group 2", Group: "To Group 2", URL: "From Group 2"}
	assert.Exactly(t, expected, c2, "should replace group")

	c1 = Channel{Group: "Group 1"}
	c1Original = copier.TDeep(t, c1)
	c2 = c1.replaceGroup(r)

	assert.NotSame(t, &c1, &c2, "should return copy of channel")
	assert.Exactly(t, c1Original, c1, "should not modify the source")

	assert.Exactly(t, c1, c2, "group should stay the same")
}

func TestParse(t *testing.T) {
	r := newDefRepo()

	playlist, err := openuri.Open("test.m3u8")
	assert.NoError(t, err, "Should read playlist")

	cl := r.Parse(playlist)

	assert.Len(t, cl, 5, "should parse this amount of channels")

	expected := Channel{Name: ", Channel 1", Group: "Group 1", URL: "http://channel/url/1"}
	assert.Exactly(t, expected, cl[0], "should have this channel")

	expected = Channel{Name: ",:It,\"s, - a difficult name |", Group: "Ext Group", URL: "ftp://channel/url/2"}
	assert.Exactly(t, expected, cl[1], "should have this channel with a group from #EXTGRP")

	expected = Channel{Name: "Channel 3", Group: "Group 3", URL: "/path/to/file/3"}
	assert.Exactly(t, expected, cl[2], "should have this channel, prioritize group-title over #EXTGRP")

	expected = Channel{Name: "Channel 4", Group: "Ext Group", URL: "file:///channel/url/4"}
	assert.Exactly(t, expected, cl[3], "should have this channel with a group from previous #EXTGRP")

	expected = Channel{Name: "Channel 5", Group: "#EXTGRP: Ext Group 2", URL: "file:///C:/channel/url/5"}
	assert.Exactly(t, expected, cl[4], "should have this channel, overwrite previous #EXTGRP")
}

func TestReplaceGroups(t *testing.T) {
	r := newDefRepo()

	r.cfg.M3U.ChannGroupMap = map[string]string{
		"From Group 1": "To Group 1",
		"From Group 2": "To Group 2",
	}

	cl1 := []Channel{
		{Group: "Other"}, {Group: "From Group 2"}, {Group: "From Group 1"},
	}
	cl1Original := copier.TDeep(t, cl1)

	cl2 := r.ReplaceGroups(cl1)
	assert.NotSame(t, &cl1, &cl2, "should return copy of channels")
	assert.Exactly(t, cl1Original, cl1, "should not modify the source")

	assert.Len(t, cl2, len(cl1), "amount of output channels should stay the same")

	expected := Channel{Group: "Other"}
	assert.Exactly(t, expected, cl2[0], "should not modify channel with unknown group")

	expected = Channel{Group: "To Group 2"}
	assert.Exactly(t, expected, cl2[1], "should replace known group")

	expected = Channel{Group: "To Group 1"}
	assert.Exactly(t, expected, cl2[2], "should replace known group")
}

func TestRemoveBlocked(t *testing.T) {
	r := newDefRepo()

	r.cfg.M3U.ChannNameBlacklist = []regexp.Regexp{
		*regexp.MustCompile("Name 1"),
		*regexp.MustCompile("Name 2"),
	}
	r.cfg.M3U.ChannGroupBlacklist = []regexp.Regexp{
		*regexp.MustCompile("Group 1"),
		*regexp.MustCompile("Group 2"),
	}
	r.cfg.M3U.ChannURLBlacklist = []regexp.Regexp{
		*regexp.MustCompile("url/1"),
		*regexp.MustCompile("url/2"),
	}

	cl1 := []Channel{
		/* 0 */ {Name: "The Name 2 Something", Group: "Other", URL: "http://other/url"},
		/* 1 */ {Name: "Other", Group: "The Group 2 Something", URL: "http://other/url"},
		/* 2 */ {Name: "Other", Group: "Other", URL: "http://url/2/something"},
		/* 3 */ {Name: "The Name 1 Something", Group: "The Group 1 Something", URL: "http://url/1/something"},
		/* 4 */ {Name: "Other", Group: "Other", URL: "Other"},
		/* 5 */ {},
	}
	cl1Original := copier.TDeep(t, cl1)

	cl2 := r.RemoveBlocked(cl1)
	assert.NotSame(t, &cl1, &cl2, "should return copy of channels")
	assert.Exactly(t, cl1Original, cl1, "should not modify the source")

	assert.Len(t, cl2, 2, "should remove blocked channels")

	expected := Channel{Name: "Other", Group: "Other", URL: "Other"}
	assert.Exactly(t, expected, cl2[0], "should keep channels without any property matching blacklist")

	assert.Exactly(t, Channel{}, cl2[1], "should keep empty channel")
}

func TestHasUrl(t *testing.T) {
	r := newDefRepo()

	cl := []Channel{{URL: "http://other/input"}, {URL: "http://known/input#a"}}

	assert.False(t, r.HasURL(cl, "http://known/input", true), "should not contain URL without hash")
	assert.True(t, r.HasURL(cl, "http://known/input#a", true), "should contain URL")

	assert.True(t, r.HasURL(cl, "http://known/input", false), "should contain URL without hash")
	assert.True(t, r.HasURL(cl, "http://known/input#b", false), "should contain URL with different hashes")

	assert.False(t, r.HasURL(cl, "http://foreign/input", true), "should not contain URL")
	assert.False(t, r.HasURL(cl, "http://foreign/input#a", true), "should not contain URL")

	assert.False(t, r.HasURL(cl, "http://foreign/input", false), "should not contain URL")
	assert.False(t, r.HasURL(cl, "http://foreign/input#b", false), "should not contain URL")

	cl = []Channel{{URL: "http://other/input#a"}, {URL: "http:/other/input/2#b"}}

	assert.False(t, r.HasURL(cl, "http://foreign/input", true), "should not contain URL")
	assert.False(t, r.HasURL(cl, "http://foreign/input#a", true), "should not contain URL")

	assert.False(t, r.HasURL(cl, "http://foreign/input", false), "should not contain URL")
	assert.False(t, r.HasURL(cl, "http://foreign/input#b", false), "should not contain URL")
}
