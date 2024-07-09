package astra

import (
	"testing"

	"m3u_merge_astra/util/copier"

	"github.com/stretchr/testify/assert"
	"github.com/zenizh/go-capturer"
)

func TestAddNewGroups(t *testing.T) {
	r := newDefRepo()

	cl1 := []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}, {Name: "A"}, {Name: "B"}}},
		{Name: "Category 2", Groups: []Group{{Name: "C"}, {Name: "D"}}},
	}
	cl1Original := copier.TestDeep(t, cl1)
	sl1 := []Stream{
		{Groups: map[string]string{"Category 3": "A"}},
		{Groups: map[string]string{"Category 3": "B"}},
		{Groups: map[string]string{"Category 3": "B"}},
		{Groups: map[string]string{"Category 1": ""}},
		{Groups: map[string]string{"Category 1": "B"}},
		{Groups: map[string]string{"Category 2": "D"}},
		{Groups: map[string]string{"Category 2": "C"}},
		{Groups: map[string]string{"Category 2": "B"}},
		{Groups: map[string]string{"Category 2": "A"}},
	}
	sl1Original := copier.TestDeep(t, sl1)

	cl2 := r.AddNewGroups(cl1, sl1)

	assert.NotSame(t, &cl1, &cl2, "should return copy of categories")
	assert.Exactly(t, cl1Original, cl1, "should not modify the source categories")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source streams")

	expected := []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}, {Name: "A"}, {Name: "B"}}},
		{Name: "Category 2", Groups: []Group{{Name: "C"}, {Name: "D"}, {Name: "B"}, {Name: "A"}}},
		{Name: "Category 3", Groups: []Group{{Name: "A"}, {Name: "B"}}},
	}
	assert.Exactly(t, expected, cl2, "should add new category with the specified groups")

	// Test log output
	out := capturer.CaptureStderr(func() {
		r := newDefRepo()

		cl1 := []Category{}

		sl1 := []Stream{{Groups: map[string]string{"Cat": "Grp"}}}

		_ = r.AddNewGroups(cl1, sl1)
	})
	assert.Contains(t, out, `Adding new category and group from streams to categories field: `+
		`category "Cat", group "Grp`)
}
