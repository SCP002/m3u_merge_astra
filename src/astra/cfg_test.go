package astra

import (
	"testing"

	"m3u_merge_astra/util/copier"

	"github.com/samber/lo"
	"github.com/stretchr/testify/assert"
	"github.com/zenizh/go-capturer"
)

func TestUpdateCategories(t *testing.T) {
	r := newDefRepo()

	cl1 := []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}, {Name: "A"}, {Name: "B"}}},
		{Name: "Category 2", Groups: []Group{{Name: "C"}, {Name: "D"}, {Name: "D"}}},
	}
	cl1Original := copier.TestDeep(t, cl1)
	sl1 := []Stream{
		{Groups: map[string]string{"Category 3": "A"}},
		{Groups: map[string]string{"Category 3": "B"}},
		{Groups: map[string]string{"Category 3": "B"}},
		{Groups: map[string]string{"Category 3": "C"}},
		{Groups: map[string]string{"Category 1": ""}},
		{Groups: map[string]string{"Category 1": "B"}},
		{Groups: map[string]string{"Category 2": "D"}},
		{Groups: map[string]string{"Category 2": "C"}},
		{Groups: map[string]string{"Category 2": "B"}},
		{Groups: map[string]string{"Category 2": "A"}},
	}
	sl1Original := copier.TestDeep(t, sl1)

	cl2 := r.UpdateCategories(cl1, sl1)

	assert.NotSame(t, &cl1, &cl2, "should return copy of categories")
	assert.Exactly(t, cl1Original, cl1, "should not modify the source categories")
	assert.Exactly(t, sl1Original, sl1, "should not modify the source streams")

	expected := []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}, {Name: "A"}, {Name: "B"}}},
		{Name: "Category 2", Groups: []Group{{Name: "C"}, {Name: "D"}, {Name: "D"}, {Name: "B"}, {Name: "A"}}},
		{Name: "Category 3", Groups: []Group{{Name: "A"}, {Name: "B"}, {Name: "C"}}},
	}
	assert.Exactly(t, expected, cl2, "should add new category with the specified groups")

	// Test log output
	out := capturer.CaptureStderr(func() {
		r := newDefRepo()

		cl1 := []Category{}

		sl1 := []Stream{{Groups: map[string]string{"Cat": "Grp"}}}

		_ = r.UpdateCategories(cl1, sl1)
	})
	assert.Contains(t, out, `Updating categories field with: category "Cat", group "Grp"`)
}

func TestChangedCategories(t *testing.T) {
	r := newDefRepo()

	// Test with changes
	cl1 := []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}}},
		{Name: "Category 2", Groups: []Group{{Name: "B"}}},
	}
	cl1Original := copier.TestDeep(t, cl1)

	cl2 := []Category{
		{Name: "Category 2", Groups: []Group{{Name: "C"}, {Name: "D"}}},
		{Name: "Category 3", Groups: []Group{{Name: "A"}, {Name: "B"}}},
	}
	cl2Original := copier.TestDeep(t, cl2)

	changed := r.ChangedCategories(cl1, cl2)
	assert.Exactly(t, cl1Original, cl1, "should not modify the source categories")
	assert.Exactly(t, cl2Original, cl2, "should not modify the source categories")

	expected := []lo.Entry[int, Category]{
		{Key: 1, Value: Category{Name: "Category 2", Groups: []Group{{Name: "C"}, {Name: "D"}}}},
		{Key: -1, Value: Category{Name: "Category 3", Groups: []Group{{Name: "A"}, {Name: "B"}}}},
	}
	assert.Exactly(t, expected, changed, "should return that category map")

	// Test without changes
	cl1 = []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}}},
		{Name: "Category 2", Groups: []Group{{Name: "B"}}},
	}
	cl1Original = copier.TestDeep(t, cl1)

	cl2 = []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}}},
		{Name: "Category 2", Groups: []Group{{Name: "B"}}},
	}
	cl2Original = copier.TestDeep(t, cl2)

	changed = r.ChangedCategories(cl1, cl2)
	assert.Exactly(t, cl1Original, cl1, "should not modify the source categories")
	assert.Exactly(t, cl2Original, cl2, "should not modify the source categories")

	expected = nil
	assert.Exactly(t, expected, changed, "should return empty map if no changs found")
}

func TestMergeCategories(t *testing.T) {
	cl1 := []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}, {Name: "A"}, {Name: "B"}}},
		{Name: "Category 2", Groups: []Group{{Name: "A"}, {Name: "B"}, {Name: "B"}}},
		{Name: "Category 1", Groups: []Group{{Name: "C"}, {Name: "A"}, {Name: "D"}}},
		{Name: "Category 2", Groups: []Group{{Name: "A"}, {Name: "C"}}},
		{Name: "Category 2", Groups: []Group{{Name: "D"}, {Name: "E"}}},
	}
	cl1Original := copier.TestDeep(t, cl1)

	cl2 := MergeCategories(cl1)

	assert.NotSame(t, &cl1, &cl2, "should return copy of categories")
	assert.Exactly(t, cl1Original, cl1, "should not modify the source categories")

	expected := []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}, {Name: "B"}, {Name: "C"}, {Name: "D"}}},
		{Name: "Category 2", Groups: []Group{{Name: "A"}, {Name: "B"}, {Name: "C"}, {Name: "D"}, {Name: "E"}}},
	}

	assert.Exactly(t, expected, cl2, "should return unique categories with the combined groups")
}
