package astra

import (
	"m3u_merge_astra/util/copier"
	"m3u_merge_astra/util/slice"
	"m3u_merge_astra/util/slice/find"
	"sort"

	"github.com/google/go-cmp/cmp"
	"github.com/samber/lo"
)

// Cfg represents astra config
type Cfg struct {
	Categories []Category `json:"categories"`
	Streams    []Stream   `json:"make_stream"`
}

// Category represents category for groups of astra streams
type Category struct {
	Name   string  `json:"name"`
	Groups []Group `json:"groups"`
	Remove bool    `json:"remove,omitempty"` // Used by API to remove category
}

// Group represents group of astra streams
type Group struct {
	Name   string `json:"name"`
	Remove bool   `json:"remove,omitempty"` // Used by API to remove group
}

// UpdateCategories returns deep copy of categories <cats> with new and changed categories and groups from <streams>
func (r repo) UpdateCategories(cats []Category, streams []Stream) []Category {
	r.log.Info("Updating categories field with new and changed categories and groups from streams")

	cats = copier.MustDeep(cats)

	// Transform []Stream into []Category
	sCats := lo.FlatMap(streams, func(s Stream, _ int) []Category {
		return lo.MapToSlice(s.Groups, func(catName string, groupName string) Category {
			return Category{Name: catName, Groups: lo.WithoutEmpty([]Group{
				{Name: groupName},
			})}
		})
	})

	// Update input categories with categories from []Stream
	for _, sCat := range sCats {
		var idx int
		cats, _, idx = find.IndexOrElse(cats, Category{Name: sCat.Name}, func(c Category) bool {
			return c.Name == sCat.Name
		})
		cats[idx].Groups = slice.AppendNew(cats[idx].Groups, func(g Group) {
			r.log.InfoCFi("Updating categories field with", "category", sCat.Name, "group", g.Name)
		}, sCat.Groups...)
	}

	return cats
}

// ChangedCategories returns new and changed categories and groups from <newCats>, which are not in <oldCats>.
//
// Key (index) in <out> is negative for new categories and actual indexes for changed categories.
func (r repo) ChangedCategories(oldCats, newCats []Category) (out []lo.Entry[int, Category]) {
	r.log.Info("Building changed categories list")

	for _, newCat := range newCats {
		oldCat, oldCatIdx, found := lo.FindIndexOf(oldCats, func(oldCat Category) bool {
			return newCat.Name == oldCat.Name
		})
		if found {
			if !cmp.Equal(oldCat, newCat) {
				out = append(out, lo.Entry[int, Category]{Key: oldCatIdx, Value: newCat})
			}
		} else {
			out = append(out, lo.Entry[int, Category]{Key: -1, Value: newCat})
		}
	}

	// Move categories which should be removed to the end of the output and order by indexes in decreasing order
	sort.SliceStable(out, func(i, j int) bool {
		if !out[i].Value.Remove && out[j].Value.Remove {
			return true
		}
		if out[i].Value.Remove && out[j].Value.Remove {
			return out[i].Key > out[j].Key
		}
		return i < j
	})

	return
}

// MergeCategories returns shallow copy of <cats> with unique categories and their groups set from all categories with
// the same name.
//
// Categories to be removed has Remove field set to true.
func (r repo) MergeCategories(cats []Category) []Category {
	r.log.Info("Merging categories")

	groupUniqFn := func(g Group) string {
		return g.Name
	}

	groupTransformFn := func(g Group, _ int, dupl bool) Group {
		if dupl {
			g.Remove = true
		}
		return g
	}

	cats = lo.Map(cats, func(c Category, _ int) Category {
		c.Groups = slice.MapFindDuplBy(c.Groups, groupUniqFn, groupTransformFn)
		c.Remove = true // Will set to false for uniques later
		return c
	})

	for _, cat := range cats {
		_, firstIdx, found := lo.FindIndexOf(cats, func(c Category) bool {
			return c.Name == cat.Name
		})
		if found {
			cats[firstIdx].Groups = slice.AppendNew(cats[firstIdx].Groups, nil, cat.Groups...)
			cats[firstIdx].Remove = false
		}
	}

	return cats
}
