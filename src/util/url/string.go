package url

import (
	"net/url"
	"strings"

	"github.com/cockroachdb/errors"
	"github.com/samber/lo"
)

// Equal returns true if <lURLStr> equal <rURLStr> and error is parsing failed.
//
// If <withHash> is false, compare ignoring hashes (everything after #).
func Equal(lURLStr string, rURLStr string, withHash bool) (bool, error) {
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

// hasParameter returns true if (all) "&" separated parameter(s) of <search> exist in "&" separated parameter(s) of
// <fragment>.
func hasParameter(search string, fragment string) bool {
	fragParams := strings.Split(strings.TrimLeft(fragment, "#"), "&")
	searchParams := strings.Split(strings.TrimLeft(search, "#"), "&")

	return lo.Every(fragParams, searchParams)
}
