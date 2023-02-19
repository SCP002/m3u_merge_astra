package url

import (
	"net/url"

	"github.com/cockroachdb/errors"
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
