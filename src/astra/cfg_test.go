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
		{Groups: map[string]string{"Category 3": ""}},
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

	// Test with changes, categories to remove in the end of cl1
	cl1 := []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}}}, // 0
		{Name: "Category 2", Groups: []Group{{Name: "B"}}}, // 1
		{Name: "Category 3", Groups: []Group{{Name: "C"}}}, // 2
		{Name: "Category 4", Groups: []Group{{Name: "D"}}}, // 3
		{Name: "Category 5", Groups: []Group{{Name: "E"}}}, // 4
	}
	cl1Original := copier.TestDeep(t, cl1)

	cl2 := []Category{
		{Name: "Category 2", Groups: []Group{{Name: "C"}, {Name: "D"}}},
		{Name: "Category 4", Groups: []Group{{Name: "D"}}, Remove: true},
		{Name: "Category 6", Groups: []Group{{Name: "A"}, {Name: "B"}}},
		{Name: "Category 5", Remove: true},
		{Name: "Category 3", Groups: []Group{{Name: "D"}, {Name: "E"}}},
		{Name: "Category 7", Groups: []Group{{Name: "A"}, {Name: "B"}}},
	}
	cl2Original := copier.TestDeep(t, cl2)

	changed := r.ChangedCategories(cl1, cl2)
	assert.Exactly(t, cl1Original, cl1, "should not modify the source categories")
	assert.Exactly(t, cl2Original, cl2, "should not modify the source categories")

	expected := []lo.Entry[int, Category]{
		{Key: 1, Value: Category{Name: "Category 2", Groups: []Group{{Name: "C"}, {Name: "D"}}}},
		{Key: -1, Value: Category{Name: "Category 6", Groups: []Group{{Name: "A"}, {Name: "B"}}}},
		{Key: 2, Value: Category{Name: "Category 3", Groups: []Group{{Name: "D"}, {Name: "E"}}}},
		{Key: -1, Value: Category{Name: "Category 7", Groups: []Group{{Name: "A"}, {Name: "B"}}}},
		{Key: 4, Value: Category{Name: "Category 5", Remove: true}},
		{Key: 3, Value: Category{Name: "Category 4", Groups: []Group{{Name: "D"}}, Remove: true}},
	}
	assert.Exactly(t, expected, changed, "should return that category map")

	// Test with changes, categories to remove in the beginning of cl1
	cl1 = []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}}}, // 0
		{Name: "Category 2", Groups: []Group{{Name: "B"}}}, // 1
		{Name: "Category 3", Groups: []Group{{Name: "C"}}}, // 2
		{Name: "Category 4", Groups: []Group{{Name: "D"}}}, // 3
		{Name: "Category 5", Groups: []Group{{Name: "E"}}}, // 4
	}
	cl1Original = copier.TestDeep(t, cl1)

	cl2 = []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}}, Remove: true},
		{Name: "Category 2", Groups: []Group{{Name: "B"}}, Remove: true},
		{Name: "Category 6", Groups: []Group{{Name: "A"}, {Name: "B"}}},
		{Name: "Category 5", Groups: []Group{{Name: "F"}, {Name: "G"}}},
		{Name: "Category 3", Groups: []Group{{Name: "D"}, {Name: "E"}}},
		{Name: "Category 7", Groups: []Group{{Name: "A"}, {Name: "B"}}},
	}
	cl2Original = copier.TestDeep(t, cl2)

	changed = r.ChangedCategories(cl1, cl2)
	assert.Exactly(t, cl1Original, cl1, "should not modify the source categories")
	assert.Exactly(t, cl2Original, cl2, "should not modify the source categories")

	expected = []lo.Entry[int, Category]{
		{Key: -1, Value: Category{Name: "Category 6", Groups: []Group{{Name: "A"}, {Name: "B"}}}},
		{Key: 4, Value: Category{Name: "Category 5", Groups: []Group{{Name: "F"}, {Name: "G"}}}},
		{Key: 2, Value: Category{Name: "Category 3", Groups: []Group{{Name: "D"}, {Name: "E"}}}},
		{Key: -1, Value: Category{Name: "Category 7", Groups: []Group{{Name: "A"}, {Name: "B"}}}},
		{Key: 1, Value: Category{Name: "Category 2", Groups: []Group{{Name: "B"}}, Remove: true}},
		{Key: 0, Value: Category{Name: "Category 1", Groups: []Group{{Name: "A"}}, Remove: true}},
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
	r := newDefRepo()

	cl1 := []Category{
		{Name: "Category 1", Groups: []Group{{Name: "A"}, {Name: "A"}, {Name: "B"}}},
		{Name: "Category 2", Groups: []Group{{Name: "A"}, {Name: "B"}, {Name: "B"}}},
		{Name: "Category 1", Groups: []Group{{Name: "C"}, {Name: "A"}, {Name: "D"}}},
		{Name: "Category 2", Groups: []Group{{Name: "A"}, {Name: "C"}}},
		{Name: "Category 2", Groups: []Group{{Name: "D"}, {Name: "E"}}},
		{Name: "Category 3", Groups: []Group{{Name: "X"}}},
	}
	cl1Original := copier.TestDeep(t, cl1)

	cl2 := r.MergeCategories(cl1)

	assert.NotSame(t, &cl1, &cl2, "should return copy of categories")
	assert.Exactly(t, cl1Original, cl1, "should not modify the source categories")

	expected := []Category{
		{
			Name:   "Category 1",
			Groups: []Group{{Name: "A"}, {Name: "A", Remove: true}, {Name: "B"}, {Name: "C"}, {Name: "D"}},
		},
		{
			Name:   "Category 2",
			Groups: []Group{{Name: "A"}, {Name: "B"}, {Name: "B", Remove: true}, {Name: "C"}, {Name: "D"}, {Name: "E"}},
		},
		{
			Name:   "Category 1",
			Groups: []Group{{Name: "C"}, {Name: "A"}, {Name: "D"}},
			Remove: true,
		},
		{
			Name:   "Category 2",
			Groups: []Group{{Name: "A"}, {Name: "C"}},
			Remove: true,
		},
		{
			Name:   "Category 2",
			Groups: []Group{{Name: "D"}, {Name: "E"}},
			Remove: true,
		},
		{
			Name:   "Category 3",
			Groups: []Group{{Name: "X"}},
		},
	}
	assert.Exactly(t, expected, cl2, "should return unique categories with the combined groups")
}
