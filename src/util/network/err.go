package network

import (
	"net"
	"net/url"
	"syscall"
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
	for {
		if err, ok := err.(*net.DNSError); ok && err.Err == "no such host" {
			return NoSuchHost
		}
		errMsg := "http: server gave HTTP response to HTTPS client"
		if err, ok := err.(*url.Error); ok && err.Err.Error() == errMsg {
			return HTTPSClientHTTPServer
		}
		if err, ok := err.(syscall.Errno); ok {
			if err == 10061 || err == syscall.ECONNREFUSED {
				return Refused
			}
		}
		if err, ok := err.(net.Error); ok && err.Timeout() {
			return Timeout
		}
		if unwrap, ok := err.(interface{ Unwrap() error }); ok {
			err = unwrap.Unwrap()
		}
		if err == nil {
			return Unknown
		}
	}
}
