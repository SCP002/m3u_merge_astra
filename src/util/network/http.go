package network

import (
	"crypto/tls"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
)

// NewHttpServer starts http server at <httpPort> and https server at <httpsPort> with request handlers from <mux>.
//
// Accepts <onErr> callback for errors.
//
// Returns both running servers, <httpSrv> and <httpsSrv>.
func NewHttpServer(mux http.Handler, httpPort int, httpsPort int, onErr func(err error)) (
	httpSrv, httpsSrv *http.Server) {
	httpSrv = &http.Server{
		Addr:    fmt.Sprintf(":%v", httpPort),
		Handler: mux,
	}

	// Get certificates from httptest TLS server
	certSrv := httptest.NewTLSServer(nil)
	certs := certSrv.TLS.Certificates
	certSrv.Close()

	httpsSrv = &http.Server{
		Addr:    fmt.Sprintf(":%v", httpsPort),
		Handler: mux,
		TLSConfig: &tls.Config{
			Certificates: certs,
		},
	}

	go func() {
		err := httpSrv.ListenAndServe()
		onErr(errors.Wrap(err, "HTTP"))
	}()
	go func() {
		err := httpsSrv.ListenAndServeTLS("", "")
		onErr(errors.Wrap(err, "HTTPS"))
	}()

	return
}

// NewHttpClient returns new HTTP client.
//
// <timeout> is a time limit for requests made by returned client.
func NewHttpClient(timeout time.Duration) *http.Client {
	tlsCfg := &tls.Config{
		InsecureSkipVerify: true,
	}
	client := &http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: tlsCfg,
		},
	}
	return client
}

// NewFakeHttpClient returns new HTTP client which make connections to localhost regardless of target host specified.
//
// <timeout> is a time limit for requests made by returned client.
func NewFakeHttpClient(timeout time.Duration) *http.Client {
	client := NewHttpClient(timeout)
	tlsCfg := &tls.Config{
		InsecureSkipVerify: true,
	}
	client.Transport = &http.Transport{
		TLSClientConfig: tlsCfg,
		Dial: func(network, addr string) (net.Conn, error) {
			// Replace request destination host with localhost:port (same as :port)
			port := strings.Split(addr, ":")[1]
			return net.Dial(network, fmt.Sprintf(":%v", port))
		},
	}
	return client
}
