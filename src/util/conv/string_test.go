package conv

import (
	"m3u_merge_astra/cfg"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestIsNameSame(t *testing.T) {
	cfg := cfg.General{
		FullTranslitMap:    cfg.DefFullTranslitMap(),
		SimilarTranslitMap: cfg.DefSimilarTranslitMap(),
	}

	// Left "НТВ" is a cyrillic visually similar to latin

	cfg.SimilarTranslit = false
	cfg.FullTranslit = false
	assert.False(t, IsNameSame(cfg, "НТВ HD", "htb hd"), "names should not be equvalent")
	assert.False(t, IsNameSame(cfg, "монитор", "Monitor!"), "names should not be equvalent")
	assert.True(t, IsNameSame(cfg, "Some Thing", "@Something"), "names should be equvalent")

	cfg.SimilarTranslit = true
	cfg.FullTranslit = false
	assert.True(t, IsNameSame(cfg, "НТВ HD", "htb hd"), "names should be equvalent")
	assert.False(t, IsNameSame(cfg, "монитор", "Monitor!"), "names should not be equvalent")
	assert.True(t, IsNameSame(cfg, "Some Thing", "@Something"), "names should be equvalent")

	cfg.SimilarTranslit = false
	cfg.FullTranslit = true
	assert.False(t, IsNameSame(cfg, "НТВ HD", "htb hd"), "names should not be equvalent")
	assert.True(t, IsNameSame(cfg, "монитор", "Monitor!"), "names should be equvalent")
	assert.True(t, IsNameSame(cfg, "Some Thing", "@Something"), "names should be equvalent")

	cfg.SimilarTranslit = true
	cfg.FullTranslit = true
	assert.True(t, IsNameSame(cfg, "НТВ HD", "htb hd"), "names should be equvalent")
	assert.True(t, IsNameSame(cfg, "монитор", "Monitor!"), "names should be equvalent")
	assert.True(t, IsNameSame(cfg, "Some Thing", "@Something"), "names should be equvalent")

	assert.False(t, IsNameSame(cfg, "Some Thing (+2)", "@Something2"), "should not discard the + symbol")
}

func TestLinksEqual(t *testing.T) {
	// Both good, with hash
	equal, err := LinksEqual("http://url/1", "http://url/1", true)
	assert.True(t, equal, "should be equal")
	assert.NoError(t, err, "should not return error")

	equal, err = LinksEqual("http://url/1#a", "http://url/1#b", true)
	assert.False(t, equal, "should not be equal")
	assert.NoError(t, err, "should not return error")

	equal, err = LinksEqual("http://url/1#a", "http://url/2#a", true)
	assert.False(t, equal, "should not be equal")
	assert.NoError(t, err, "should not return error")

	// Both bad, with hash
	equal, err = LinksEqual("http://{bad/url/1", "http://{bad/url/1", true)
	assert.True(t, equal, "should be equal")
	assert.NoError(t, err, "should not return error")

	equal, err = LinksEqual("http://{bad/url/1#a", "http://{bad/url/1#b", true)
	assert.False(t, equal, "should not be equal")
	assert.NoError(t, err, "should not return error")

	equal, err = LinksEqual("http://{bad/url/1#a", "http://{bad/url/2#a", true)
	assert.False(t, equal, "should not be equal")
	assert.NoError(t, err, "should not return error")

	// Both good, no hash
	equal, err = LinksEqual("http://url/1", "http://url/1", false)
	assert.True(t, equal, "should be equal")
	assert.NoError(t, err, "should not return error")

	equal, err = LinksEqual("http://url/1#a", "http://url/1#b", false)
	assert.True(t, equal, "should be equal")
	assert.NoError(t, err, "should not return error")

	equal, err = LinksEqual("http://url/1#a", "http://url/2#a", false)
	assert.False(t, equal, "should not be equal")
	assert.NoError(t, err, "should not return error")

	// Right bad, no hash
	equal, err = LinksEqual("http://url/1", "http://{bad/url/1", false)
	assert.False(t, equal, "should not be equal")
	assert.Error(t, err, "should return error")

	equal, err = LinksEqual("http://url/1#a", "http://{bad/url/1#b", false)
	assert.False(t, equal, "should not be equal")
	assert.Error(t, err, "should return error")

	equal, err = LinksEqual("http://url/1#a", "http://{bad/url/2#a", false)
	assert.False(t, equal, "should not be equal")
	assert.Error(t, err, "should return error")

	// Both bad, no hash
	equal, err = LinksEqual("http://{bad/url/1", "http://{bad/url/1", false)
	assert.True(t, equal, "should be equal")
	assert.Error(t, err, "should return error")

	equal, err = LinksEqual("http://{bad/url/1#a", "http://{bad/url/1#b", false)
	assert.False(t, equal, "should not be equal")
	assert.Error(t, err, "should return error")

	equal, err = LinksEqual("http://{bad/url/1#a", "http://{bad/url/2#a", false)
	assert.False(t, equal, "should not be equal")
	assert.Error(t, err, "should return error")
}

func TestGetHah(t *testing.T) {
	// Good URL's
	hash, err := GetHash("http://url")
	assert.Empty(t, hash, "should return empty string")
	assert.NoError(t, err, "should not return error")

	hash, err = GetHash("http://url#a")
	assert.Exactly(t, "a", hash, "should return hash")
	assert.NoError(t, err, "should not return error")

	hash, err = GetHash("http://url#a&b")
	assert.Exactly(t, "a&b", hash, "should return hash")
	assert.NoError(t, err, "should not return error")

	// Bad URL's
	hash, err = GetHash("http://{bad/url")
	assert.Empty(t, hash, "should not retun hash from invalid URL")
	assert.Error(t, err, "should return error")

	hash, err = GetHash("http://{bad/url#a")
	assert.Empty(t, hash, "should not retun hash from invalid URL")
	assert.Error(t, err, "should return error")

	hash, err = GetHash("http://{bad/url#a&b")
	assert.Empty(t, hash, "should not retun hash from invalid URL")
	assert.Error(t, err, "should return error")
}

func TestAddHash(t *testing.T) {
	// Good URL's
	result, changed, err := AddHash("", "http://url")
	assert.Exactly(t, "http://url", result, "should stay unmodified")
	assert.False(t, changed)
	assert.NoError(t, err, "should not return error")

	result, changed, err = AddHash("", "http://url#a")
	assert.Exactly(t, "http://url#a", result, "should stay unmodified")
	assert.False(t, changed)
	assert.NoError(t, err, "should not return error")

	result, changed, err = AddHash("a", "http://url")
	assert.Exactly(t, "http://url#a", result, "should get new hash")
	assert.True(t, changed)
	assert.NoError(t, err, "should not return error")

	result, changed, err = AddHash("#a", "http://url#a")
	assert.Exactly(t, "http://url#a", result, "should stay unmodified")
	assert.False(t, changed)
	assert.NoError(t, err, "should not return error")

	result, changed, err = AddHash("b", "http://url#a")
	assert.Exactly(t, "http://url#a&b", result, "should merge hashes")
	assert.True(t, changed)
	assert.NoError(t, err, "should not return error")

	result, changed, err = AddHash("a", "http://url#a&b")
	assert.Exactly(t, "http://url#a&b", result, "should stay unmodified")
	assert.False(t, changed)
	assert.NoError(t, err, "should not return error")

	result, changed, err = AddHash("c", "http://url#a&b")
	assert.Exactly(t, "http://url#a&b&c", result, "should merge hashes")
	assert.True(t, changed)
	assert.NoError(t, err, "should not return error")

	result, changed, err = AddHash("c&d", "http://url#a&b")
	assert.Exactly(t, "http://url#a&b&c&d", result, "should merge hashes")
	assert.True(t, changed)
	assert.NoError(t, err, "should not return error")

	result, changed, err = AddHash("#c&d", "http://url#a&b")
	assert.Exactly(t, "http://url#a&b&c&d", result, "should merge hashes")
	assert.True(t, changed)
	assert.NoError(t, err, "should not return error")

	// Bad URL's
	result, changed, err = AddHash("", "http://{bad/url")
	assert.Exactly(t, "http://{bad/url", result, "should stay unmodified")
	assert.False(t, changed)
	assert.NoError(t, err, "should not return error")

	result, changed, err = AddHash("", "http://{bad/url#a")
	assert.Exactly(t, "http://{bad/url#a", result, "should stay unmodified")
	assert.False(t, changed)
	assert.NoError(t, err, "should not return error")

	result, changed, err = AddHash("a", "http://{bad/url")
	assert.Exactly(t, "http://{bad/url", result, "should stay unmodified")
	assert.False(t, changed)
	assert.Error(t, err, "should return error")

	result, changed, err = AddHash("#a", "http://{bad/url#a")
	assert.Exactly(t, "http://{bad/url#a", result, "should stay unmodified")
	assert.False(t, changed)
	assert.Error(t, err, "should return error")

	result, changed, err = AddHash("b", "http://{bad/url#a")
	assert.Exactly(t, "http://{bad/url#a", result, "should stay unmodified")
	assert.False(t, changed)
	assert.Error(t, err, "should return error")

	result, changed, err = AddHash("a", "http://{bad/url#a&b")
	assert.Exactly(t, "http://{bad/url#a&b", result, "should stay unmodified")
	assert.False(t, changed)
	assert.Error(t, err, "should return error")

	result, changed, err = AddHash("c", "http://{bad/url#a&b")
	assert.Exactly(t, "http://{bad/url#a&b", result, "should stay unmodified")
	assert.False(t, changed)
	assert.Error(t, err, "should return error")

	result, changed, err = AddHash("c&d", "http://{bad/url#a&b")
	assert.Exactly(t, "http://{bad/url#a&b", result, "should stay unmodified")
	assert.False(t, changed)
	assert.Error(t, err, "should return error")

	result, changed, err = AddHash("#c&d", "http://{bad/url#a&b")
	assert.Exactly(t, "http://{bad/url#a&b", result, "should stay unmodified")
	assert.False(t, changed)
	assert.Error(t, err, "should return error")
}

func TestContainsAny(t *testing.T) {
	assert.True(t, ContainsAny("some words", "some"), "should contain element")
	assert.True(t, ContainsAny("some words", "unknown", "some"), "should contain element")
	assert.False(t, ContainsAny("some words", "unknown", "unknown 2"), "should not contain any element")
}

func TestHasParameter(t *testing.T) {
	assert.True(t, hasParameter("a", "a"), "should contain parameter")
	assert.True(t, hasParameter("a", "#a"), "should contain parameter")
	assert.False(t, hasParameter("b", "a"), "should not contain parameter")
	assert.False(t, hasParameter("b", "#a"), "should not contain parameter")

	assert.True(t, hasParameter("a", "a&b"), "should contain parameter")
	assert.True(t, hasParameter("b", "#a&b"), "should contain parameter")
	assert.False(t, hasParameter("c", "a&b"), "should not contain parameter")
	assert.False(t, hasParameter("c", "#a&b"), "should not contain parameter")

	assert.True(t, hasParameter("a&b", "a&b"), "should contain parameter")
	assert.True(t, hasParameter("a&b", "#a&b"), "should contain parameter")
	assert.True(t, hasParameter("a&b", "a&b&c"), "should contain parameter")
	assert.False(t, hasParameter("a&b&c", "a&b"), "should not contain parameter")

	assert.True(t, hasParameter("b&a", "a&b"), "should contain parameter")
	assert.True(t, hasParameter("b&a", "#a&b"), "should contain parameter")
	assert.True(t, hasParameter("b&a", "a&b&c"), "should contain parameter")
	assert.True(t, hasParameter("b&a", "#a&b&c"), "should contain parameter")

	assert.False(t, hasParameter("a&c", "a&b"), "should not contain parameter")
	assert.False(t, hasParameter("a&c", "#a&b"), "should not contain parameter")
}

func TestSimplifyName(t *testing.T) {
	assert.Exactly(t, "samplename", simplifyName("Sample, Name!\r\n"), "should return simplified name")
	assert.NotEqual(t, "samplename2", simplifyName("Sample Name (+2)"), "should not discard the + symbol")
}

func TestRemap(t *testing.T) {
	dict := map[string]string{"A": "1", "B": "2", "C": "3"}
	assert.Exactly(t, "123D", remap("ABCD", dict), "should replace every char of input with proper value from dictonary")
}
