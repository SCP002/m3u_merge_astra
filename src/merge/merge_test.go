package merge

import (
	"m3u_merge_astra/astra"
	"m3u_merge_astra/cfg"
	"m3u_merge_astra/m3u"
	"m3u_merge_astra/util/copier"
	"regexp"
	"strings"
	"testing"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
)

func TestRenameStreams(t *testing.T) {
	r := newDefRepo()

	sl1 := []astra.Stream{
		{Name: "Known name 2"}, {Name: "Known name"}, {Name: "Other name A"},
	}
	sl1Original := copier.TDeep(t, sl1)
	cl1 := []m3u.Channel{
		{Name: "Other_Name_B"}, {Name: "Known_Name_2"}, {Name: "Known_Name"},
	}
	cl1Original := copier.TDeep(t, cl1)

	sl2 := r.RenameStreams(sl1, cl1)

	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source streams")
	assert.Exactly(t, cl1Original, cl1, "should not modify the source channels")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	expected := astra.Stream{Name: "Known_Name_2"}
	assert.Exactly(t, expected, sl2[0], "should rename stream to it's channel counterpart")

	expected = astra.Stream{Name: "Known_Name"}
	assert.Exactly(t, expected, sl2[1], "should rename stream to it's channel counterpart")

	expected = astra.Stream{Name: "Other name A"}
	assert.Exactly(t, expected, sl2[2], "should not rename stream if no channel counterpart name found")
}

func TestUpdateInputs(t *testing.T) {
	r := newDefRepo()
	r.cfg.Streams.InputUpdateMap = []cfg.UpdateRecord{
		{From: *regexp.MustCompile("known/input/2"), To: *regexp.MustCompile("known/input/2")},
		{From: *regexp.MustCompile("known/input/1"), To: *regexp.MustCompile("known/input/1")},
	}

	sl1 := []astra.Stream{
		{Name: "Known name", Inputs: []string{
			"http://other/input", "http://known/input/2#a", "http://known/input/1", "http://known/input/1#b",
		}},
		{Name: "Known name 2", Inputs: []string{"http://known/input/2#b"}},
		{Name: "Other name A", Inputs: []string{"http://known/input/1#a"}},
	}
	sl1Original := copier.TDeep(t, sl1)
	cl1 := []m3u.Channel{
		{Name: "Other_Name_B", URL: "http://new/known/input/1#a"},
		{Name: "Known_Name", URL: "http://new/known/input/2#b"},
		{Name: "Known_Name_2", URL: "http://other/url"},
		{Name: "Known-Name", URL: "http://new/known/input/1"},
		{Name: "KnownName", URL: "http://_other_/input"},
	}
	cl1Original := copier.TDeep(t, cl1)

	r.cfg.Streams.KeepInputHash = false
	sl2 := r.UpdateInputs(sl1, cl1)

	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source streams")
	assert.Exactly(t, cl1Original, cl1, "should not modify the source channels")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	expected := []string{
		"http://other/input", "http://new/known/input/2#b", "http://new/known/input/1", "http://known/input/1#b",
	}
	assert.Exactly(t, expected, sl2[0].Inputs, "should have these inputs")

	assert.Exactly(t, sl1[1], sl2[1], "should not update to input to channel URL not in cfg.Streams.InputUpdateMap")

	assert.Exactly(t, sl1[2], sl2[2], "only streams with known names should be updated")

	r.cfg.Streams.KeepInputHash = true
	sl2 = r.UpdateInputs(sl1, cl1)

	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source streams")
	assert.Exactly(t, cl1Original, cl1, "should not modify the source channels")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	expected = []string{
		"http://other/input", "http://new/known/input/2#b&a", "http://new/known/input/1", "http://known/input/1#b",
	}
	assert.Exactly(t, expected, sl2[0].Inputs, "should have these inputs")

	assert.Exactly(t, sl1[1], sl2[1], "should not update to input to channel URL not in cfg.Streams.InputUpdateMap")

	assert.Exactly(t, sl1[2], sl2[2], "only streams with known names should be updated")
}

func TestRemoveInputsByUpdateMap(t *testing.T) {
	r := newDefRepo()
	r.cfg.Streams.InputUpdateMap = []cfg.UpdateRecord{
		{From: *regexp.MustCompile("known/input/1")},
		{From: *regexp.MustCompile("known/input/2")},
	}

	sl1 := []astra.Stream{
		{Name: "Known name", Inputs: []string{"http://other/input", "http://known/input/2", "http://known/input/1#a"}},
		{Name: "Known name 2", Inputs: []string{"http://known/input/2#a"}},
		{Name: "Other name A", Inputs: []string{"http://known/input/1#a", "http://known/input/2", "http://other/input"}},
	}
	sl1Original := copier.TDeep(t, sl1)
	cl1 := []m3u.Channel{
		{Name: "Other_Name_B", URL: "http://other/url"},
		{Name: "Known_Name", URL: "http://known/input/1#b"},
		{Name: "Known_Name_2", URL: "http://other/url"},
		{Name: "Known_Name_2", URL: "http://known/input/2"},
	}
	cl1Original := copier.TDeep(t, cl1)
	sl2 := r.RemoveInputsByUpdateMap(sl1, cl1)

	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source streams")
	assert.Exactly(t, cl1Original, cl1, "should not modify the source channels")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	expected := astra.Stream{
		Name:           "Known name",
		Inputs:         []string{"http://other/input", "http://known/input/1#a"},
		DisabledInputs: make([]string, 0),
	}
	assert.Exactly(t, expected, sl2[0], "unknown inputs and inputs which only differ by hash should stay")

	expected = astra.Stream{Name: "Known name 2", Inputs: []string{"http://known/input/2#a"}}
	assert.Exactly(t, expected, sl2[1], "known inputs which only differ by hash should stay")

	expected = astra.Stream{
		Name:           "Other name A",
		Inputs:         []string{"http://other/input"},
		DisabledInputs: make([]string, 0),
	}
	assert.Exactly(t, expected, sl2[2], "known inputs not found in channels should be removed, unknown should stay")
}

func TestAddNewInputs(t *testing.T) {
	r := newDefRepo()

	sl1 := []astra.Stream{
		{Name: "Known name", Inputs: []string{"http://input/1#a", "http://input/2"}},
		{Name: "Known name 2", Inputs: []string{"http://input/1"}},
		{Name: "Known name 3", Inputs: []string{"http://input/1"}},
		{Name: "Other name A", Inputs: []string{"http://input/1#b"}},
	}
	sl1Original := copier.TDeep(t, sl1)
	cl1 := []m3u.Channel{
		{Name: "Known_Name_2", URL: "http://input/1"},
		{Name: "Known_Name", URL: "http://input/2#a"},
		{Name: "Known_Name_3", URL: "http://input/2"},
		{Name: "Other_Name_B", URL: "http://input/1#c"},
		{Name: "Known-Name-3", URL: "http://input/3"},
	}
	cl1Original := copier.TDeep(t, cl1)

	r.cfg.Streams.HashCheckOnAddNewInputs = true
	sl2 := r.AddNewInputs(sl1, cl1)

	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source streams")
	assert.Exactly(t, cl1Original, cl1, "should not modify the source channels")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	expected := astra.Stream{
		Name:   "Known name",
		Inputs: []string{"http://input/2#a", "http://input/1#a", "http://input/2"},
	}
	assert.Exactly(t, expected, sl2[0], "should add input which only differ by hash as HashCheckOnAddNewStreamInputs"+
		" = true")

	expected = astra.Stream{Name: "Known name 2", Inputs: []string{"http://input/1"}}
	assert.Exactly(t, expected, sl2[1], "should add only unknown inputs")

	expected = astra.Stream{
		Name:   "Known name 3",
		Inputs: []string{"http://input/3", "http://input/2", "http://input/1"},
	}
	assert.Exactly(t, expected, sl2[2], "should add unknown inputs")

	assert.Exactly(t, sl1[3], sl2[3], "should not change streams with unknown names")

	r.cfg.Streams.HashCheckOnAddNewInputs = false
	sl2 = r.AddNewInputs(sl1, cl1)

	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source streams")
	assert.Exactly(t, cl1Original, cl1, "should not modify the source channels")

	assert.Len(t, sl2, len(sl1), "amount of output streams should stay the same")

	assert.Exactly(t, sl1[0], sl2[0], "should not add input which only differ by hash as HashCheckOnAddNewStreamInputs"+
		" = false")

	assert.Exactly(t, sl1[1], sl2[1], "should add only unknown inputs")

	expected = astra.Stream{
		Name:   "Known name 3",
		Inputs: []string{"http://input/3", "http://input/2", "http://input/1"},
	}
	assert.Exactly(t, expected, sl2[2], "should add unknown inputs")

	assert.Exactly(t, sl1[3], sl2[3], "should not change streams with unknown names")
}

func TestAddNewStreams(t *testing.T) {
	r := newDefRepo()

	sl1 := []astra.Stream{
		{Name: "Known name", Inputs: []string{"http://some/url"}},
		{Name: "Other name A", Inputs: []string{"http://some/url/2"}},
	}
	sl1Original := copier.TDeep(t, sl1)
	cl1 := []m3u.Channel{
		{Name: "Other name B", Group: "Group", URL: "http://some/url/2"},
		{Name: "Known name", URL: "http://some/url"},
		{Name: "Other name B", Group: "Group", URL: "http://some/url/3"},
	}
	cl1Original := copier.TDeep(t, cl1)

	r.cfg.Streams.AddNewWithKnownInputs = true
	r.cfg.Streams.AddGroupsToNew = false
	sl2, gl := r.AddNewStreams(sl1, cl1)

	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source streams")
	assert.Exactly(t, cl1Original, cl1, "should not modify the source channels")

	assert.Len(t, sl2, 3, "should add new stream")

	assert.Exactly(t, sl1[0], sl2[0], "should not change existing streams")

	assert.Exactly(t, sl1[1], sl2[1], "should not change existing streams")

	expected := astra.Stream{
		Enabled:        r.cfg.Streams.MakeNewEnabled,
		Type:           string(r.cfg.Streams.NewType),
		ID:             sl2[2].ID,
		Name:           r.cfg.Streams.AddedPrefix + "Other name B",
		Inputs:         []string{"http://some/url/2"},
		DisabledInputs: make([]string, 0),
	}
	assert.Exactly(t, expected, sl2[2], "should add new stream")

	assert.Empty(t, gl, "should return empty group list as AddGroupsToNew = false")

	r.cfg.Streams.AddNewWithKnownInputs = true
	r.cfg.Streams.AddGroupsToNew = true
	sl2, gl = r.AddNewStreams(sl1, cl1)

	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source streams")
	assert.Exactly(t, cl1Original, cl1, "should not modify the source channels")

	assert.Len(t, sl2, 3, "should add new stream")

	assert.Exactly(t, sl1[0], sl2[0], "should not change existing streams")

	assert.Exactly(t, sl1[1], sl2[1], "should not change existing streams")

	expected.ID = sl2[2].ID
	expected.Groups = map[string]any{r.cfg.Streams.GroupsCategoryForNew: "Group"}
	assert.Exactly(t, expected, sl2[2], "should add new stream")

	assert.Exactly(t, []string{"Group"}, gl, "should return these added groups")

	sl1 = []astra.Stream{{Name: "Other name", Inputs: []string{"http://some/url"}}}
	cl1 = []m3u.Channel{
		{Name: "Other name", URL: "http://some/url"},
		{Name: "Other name", URL: "http://some/url/2"},
	}
	sl2, gl = r.AddNewStreams(sl1, cl1)

	assert.Exactly(t, sl1, sl2, "should not change as M3U channels does not contain any new name")

	assert.Empty(t, gl, "should return empty group list")

	sl1 = []astra.Stream{
		{Name: "Known name", Inputs: []string{"http://some/url#a"}},
		{Name: "Other name A", Inputs: []string{"http://known/url#b"}},
	}
	sl1Original = copier.TDeep(t, sl1)
	cl1 = []m3u.Channel{
		{Name: "Other name B", URL: "http://known/url#c"},
		{Name: "Known name", URL: "http://some/url/2#a"},
	}
	cl1Original = copier.TDeep(t, cl1)

	r.cfg.Streams.AddNewWithKnownInputs = false
	sl2, gl = r.AddNewStreams(sl1, cl1)

	assert.NotSame(t, &sl1, &sl2, "should return copy of streams")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source streams")
	assert.Exactly(t, cl1Original, cl1, "should not modify the source channels")

	assert.Exactly(t, sl1, sl2, "should not change as AddNewStreamsWithKnownInputs = false and hash difference should"+
		"be ignored")

	assert.Empty(t, gl, "should return empty group list")
}

func TestGenerateUID(t *testing.T) {
	sl := []astra.Stream{}

	for i := 0; i < 10000; i++ {
		uid := generateUID(sl)
		// Check length
		assert.Len(t, uid, 4, "ID should be 4 characters long")
		// Check if not uppercase
		hasUpperCase := strings.ContainsAny(uid, "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
		assert.False(t, hasUpperCase, "ID should not contain uppercase characters")
		// Check if unique
		contains := lo.ContainsBy(sl, func(s astra.Stream) bool {
			return s.ID == uid
		})
		if contains {
			assert.FailNow(t, "ID should be unique")
		}
		// Append
		sl = append(sl, astra.Stream{ID: uid})
	}
}
