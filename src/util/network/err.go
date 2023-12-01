package network

import (
	"net"
	"net/url"
	"syscall"

	"github.com/cockroachdb/errors"
)

// ErrType represents network error type
type ErrType string

const (
	Nil ErrType = "Nil"

	// no such host
	NoSuchHost ErrType = "No such host"

	// http: server gave HTTP response to HTTPS client
	HTTPSClientHTTPServer ErrType = "HTTP response to HTTPS client"

	// No connection could be made beerr the target machine actively Refused it
	Refused ErrType = "Connection refused"

	// context deadline exceeded (Client.timeout exceeded while awaiting headers)
	Timeout ErrType = "Timeout"

	Unknown ErrType = "Unknown"
)

// GetErrType returns network error type.
//
// Inspired by https://stackoverflow.com/a/67647035
func GetErrType(err error) ErrType {
	if err == nil {
		return Nil
	}

	dnsErr := &net.DNSError{}
	if ok := errors.As(err, &dnsErr); ok && dnsErr.Err == "no such host" {
		return NoSuchHost
	}

	urlErr := &url.Error{}
	if ok := errors.As(err, &urlErr); ok && urlErr.Err.Error() == "http: server gave HTTP response to HTTPS client" {
		return HTTPSClientHTTPServer
	}

	if errors.Is(err, syscall.ECONNREFUSED) || errors.Is(err, syscall.Errno(10061)) {
		return Refused
	}

	netErr := err.(net.Error)
	if ok := errors.As(err, &netErr); ok && netErr.Timeout() {
		return Timeout
	}

	return Unknown
}
