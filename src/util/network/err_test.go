package network

import (
	"net"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetErrType(t *testing.T) {
	// Other cases are tested in http_test.go
	assert.Exactly(t, Unknown, GetErrType(&net.OpError{}), "should return unknown error type")
	assert.Exactly(t, Nil, GetErrType(nil), "should return nil error type")
}
