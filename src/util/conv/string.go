package conv

import (
	"m3u_merge_astra/cfg"
	"net/url"
	"regexp"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
)

// convRegEx represents every character to remove from name before comparsion
var convRegEx regexp.Regexp = *regexp.MustCompile("[" +
	// All control characters such as \x00
	"[:cntrl:]" +
	// All blank (whitespace) characters
	"[:space:]" +
	// All punctuation characters (except the + symbol: prevents 'Name 2' == 'Name (+2)')
	"-!\"#$%&'()*,./:;<=>?@[\\]^_`{|}~" +
	"]+")

// IsNameSame returns true if standardized <lName> is equal to standardized <rName> using transliteration settings from
// <cfg>.
func IsNameSame(cfg cfg.General, lName, rName string) bool {
	lName = simpleName(lName)
	rName = simpleName(rName)
	if lName == rName {
		return true
	}
	if cfg.SimilarTranslit && remap(lName, cfg.SimilarTranslitMap) == remap(rName, cfg.SimilarTranslitMap) {
		return true
	}
	if cfg.FullTranslit && remap(lName, cfg.FullTranslitMap) == remap(rName, cfg.FullTranslitMap) {
		return true
	}
	if cfg.NameAliases && firstSimpleAlias(lName, cfg.NameAliasList) == firstSimpleAlias(rName, cfg.NameAliasList) {
		return true
	}
	return false
}

// LinksEqual returns true if <lURLStr> equal <rURLStr> and error is parsing failed.
//
// If <withHash> is false, compare ignoring hashes (everything after #).
func LinksEqual(lURLStr string, rURLStr string, withHash bool) (bool, error) {
	if withHash {
		return lURLStr == rURLStr, nil
	}

	errMsg := "Can't parse URL to compare but hash omit is required, fallback to strict comparsion"

	lURL, err := url.Parse(lURLStr)
	if err != nil {
		return lURLStr == rURLStr, errors.Wrap(err, errMsg)
	}
	rURL, err := url.Parse(rURLStr)
	if err != nil {
		return lURLStr == rURLStr, errors.Wrap(err, errMsg)
	}
	lURL.Fragment = ""
	rURL.Fragment = ""
	return lURL.String() == rURL.String(), nil
}

// GetHash returns hash of <urlStr> and error is parsing failed.
func GetHash(urlStr string) (string, error) {
	url, err := url.Parse(urlStr)
	if err != nil {
		return "", errors.Wrap(err, "Can't parse URL to get hash")
	}
	return url.Fragment, nil
}

// AddHash returns <urlStr> with <hash>, true if <urlStr> has been changed and error is parsing failed.
func AddHash(hash string, urlStr string) (string, bool, error) {
	if hash == "" {
		return urlStr, false, nil
	}
	hash = strings.TrimLeft(hash, "#")

	url, err := url.Parse(urlStr)
	if err != nil {
		return urlStr, false, errors.Wrap(err, "Can't parse URL to append hash, leaving URL unmodified")
	}
	if hasParameter(hash, url.Fragment) {
		return urlStr, false, nil
	}
	if url.Fragment == "" {
		return urlStr + "#" + hash, true, nil
	} else {
		return urlStr + "&" + hash, true, nil
	}
}

// ContainsAny returns true if <inp> contains any of <elms>
func ContainsAny(inp string, elms ...string) bool {
	return lo.SomeBy(elms, func(elm string) bool {
		return strings.Contains(inp, elm)
	})
}

// hasParameter returns true if (all) "&" separated parameter(s) of <search> exist in "&" separated parameter(s) of
// <fragment>.
func hasParameter(search string, fragment string) bool {
	fragParams := strings.Split(strings.TrimLeft(fragment, "#"), "&")
	searchParams := strings.Split(strings.TrimLeft(search, "#"), "&")

	return lo.Every(fragParams, searchParams)
}

// simpleName returns simplified <inp> (lowercase, no special characters)
func simpleName(inp string) string {
	out := strings.ToLower(inp)
	return convRegEx.ReplaceAllString(out, "")
}

// remap returns remapped <inp> using <dict>
func remap(inp string, dict map[string]string) (out string) {
	for _, oldChar := range inp {
		newChar := dict[string(oldChar)]
		if newChar != "" {
			out += newChar
		} else {
			out += string(oldChar)
		}
	}
	return
}

// firstSimpleAlias returns first simplified alias for <name> from <aliases> or simple <name> if not found
func firstSimpleAlias(name string, aliases [][]string) string {
	name = simpleName(name)
	for _, set := range aliases {
		simpleSet := lo.Map(set, func(alias string, _ int) string {
			return simpleName(alias)
		})
		if lo.Contains(simpleSet, name) {
			return simpleSet[0]
		}
	}
	return name
}
