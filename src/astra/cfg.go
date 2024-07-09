package astra

import (
	"m3u_merge_astra/util/copier"
	"m3u_merge_astra/util/slice"
	"m3u_merge_astra/util/slice/find"

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
	Remove bool    `json:"remove,omitempty"` // Used by API to remove the category
}

// Group represents group of astra streams
type Group struct {
	Name string `json:"name"`
}

// AddNewGroups returns deep copy of categories <cats> with new categories and groups from <streams>
func (r repo) AddNewGroups(cats []Category, streams []Stream) []Category {
	r.log.Info("Adding new categories and groups from streams to categories field")

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
			r.log.InfoCFi("Adding new category and group from streams to categories field",
				"category", sCat.Name, "group", g.Name)
		}, sCat.Groups...)
	}

	return cats
}
