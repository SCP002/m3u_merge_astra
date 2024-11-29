package find

import (
	"m3u_merge_astra/cfg"
	"m3u_merge_astra/util/copier"
	"m3u_merge_astra/util/slice"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIndexOrElse(t *testing.T) {
	ol1 := []string{"A", "B"}
	ol1Original := copier.TestDeep(t, ol1)

	search := "B"
	ol2, o, idx := IndexOrElse(ol1, search, func(o string) bool {
		return o == search
	})

	assert.NotSame(t, &ol1, &ol2, "should return copy of objects")
	assert.Exactly(t, ol1Original, ol1, "should not modify the source objects")

	assert.Exactly(t, ol1, ol2, "should not add existing object")
	assert.Exactly(t, search, o, "should return fallback object")
	assert.Exactly(t, 1, idx, "should return index of the existing object")

	search = "C"
	ol2, o, idx = IndexOrElse(ol1, search, func(o string) bool {
		return o == search
	})

	assert.NotSame(t, &ol1, &ol2, "should return copy of objects")
	assert.Exactly(t, ol1Original, ol1, "should not modify the source objects")

	assert.Exactly(t, []string{"A", "B", "C"}, ol2, "should add new object")
	assert.Exactly(t, search, o, "should return fallback object")
	assert.Exactly(t, 2, idx, "should return index of the new object")
}

func TestNamed(t *testing.T) {
	cfg := cfg.General{
		SimilarTranslit:    true,
		SimilarTranslitMap: cfg.DefSimilarTranslitMap(),
		FullTranslit:       true,
		FullTranslitMap:    cfg.DefFullTranslitMap(),
	}
	ol := []slice.TestNamedStruct{
		{Name: "Name"}, {Name: "Name 2"}, {Name: "Name 3"}, {Name: "Name 2"},
	}

	o, idx, found := Named(cfg, ol, "name2")
	assert.NotSame(t, &ol[1], &o, "should return copy of object")
	assert.Exactly(t, ol[1], o, "should return object matching the specified name")
	assert.Exactly(t, 1, idx, "should return index of first found element")
	assert.True(t, found, "should find object matching the specified name")

	o, idx, found = Named(cfg, ol, "name4")
	assert.Exactly(t, slice.TestNamedStruct{}, o, "should return empty object if not found")
	assert.Exactly(t, -1, idx, "should return index -1 if not found")
	assert.False(t, found, "should return false if no object matching the specified name found")
}

func TestEverySimilar(t *testing.T) {
	cfg := cfg.General{
		SimilarTranslit:    true,
		SimilarTranslitMap: cfg.DefSimilarTranslitMap(),
		FullTranslit:       true,
		FullTranslitMap:    cfg.DefFullTranslitMap(),
	}
	ol := []slice.TestNamedStruct{
		/* 0 */ {Name: "Name"},
		/* 1 */ {Name: "Name 2"}, // <- Searching similar to this starting from index 2
		/* 2 */ {Name: "Name 3"},
		/* 3 */ {Name: "Name_2"},
		/* 4 */ {Name: "Name_3"},
		/* 5 */ {Name: "Name-2"},
	}

	idxNameMap := map[int]string{}
	EverySimilar(cfg, ol, "name2!", 2, func(foundObj slice.TestNamedStruct, foundIdx int) {
		idxNameMap[foundIdx] = foundObj.Name
	})

	expected := map[int]string{3: "Name_2", 5: "Name-2"}
	assert.Exactly(t, expected, idxNameMap, "should find these objects")
}

func TestGetSimilar(t *testing.T) {
	cfg := cfg.General{
		SimilarTranslit:    true,
		SimilarTranslitMap: cfg.DefSimilarTranslitMap(),
		FullTranslit:       true,
		FullTranslitMap:    cfg.DefFullTranslitMap(),
	}
	ol1 := []slice.TestNamedStruct{
		/* 0 */ {Name: "Name"},
		/* 1 */ {Name: "Name 2"}, // <- Searching similar to this
		/* 2 */ {Name: "Name 3"},
		/* 3 */ {Name: "Name_2"},
		/* 4 */ {Name: "Name_3"},
		/* 5 */ {Name: "Name-2"},
	}
	ol1Original := copier.TestDeep(t, ol1)
	ol2 := GetSimilar(cfg, ol1, "name2!")

	assert.NotSame(t, &ol1, &ol2, "should return copy of objects")
	assert.Exactly(t, ol1Original, ol1, "should not modify the source objects")

	expected := []slice.TestNamedStruct{{Name: "Name 2"}, {Name: "Name_2"}, {Name: "Name-2"}}
	assert.Exactly(t, expected, ol2, "should find these objects")
}

func TestHasAnySimilar(t *testing.T) {
	cfg := cfg.General{
		SimilarTranslit:    true,
		SimilarTranslitMap: cfg.DefSimilarTranslitMap(),
		FullTranslit:       true,
		FullTranslitMap:    cfg.DefFullTranslitMap(),
	}
	ol := []slice.TestNamedStruct{{Name: "Name"}, {Name: "Name 2"}, {Name: "Name 3"}}

	assert.True(t, HasAnySimilar(cfg, ol, "name2!"), "should find this object")
	assert.False(t, HasAnySimilar(cfg, ol, "Name 4"), "should not find this object")
	assert.True(t, HasAnySimilar(cfg, ol, "Name 4", "name2!"), "should find second object")
}
