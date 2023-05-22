package analyzer

import (
	"context"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	dialer := websocket.DefaultDialer
	dialer.HandshakeTimeout = time.Second
	expected := &analyzer{
		url:    "ws://127.0.0.1/api/",
		dialer: dialer,
	}
	assert.Exactly(t, expected, New("127.0.0.1", time.Second), "should initialize analyzer")
}

// Requires a running astra analyzer
func TestCheck(t *testing.T) {
	handshakeTimeout := time.Second * 3
	watchTime := time.Second * 5
	analyzer := New("127.0.0.1:8001", handshakeTimeout)

	// MPEG-TS
	ctx, cancel := context.WithTimeout(context.Background(), watchTime)
	defer cancel()
	result, err := analyzer.Check(ctx, "https://tsduck.io/streams/brazil-isdb-tb/TS1globo.ts")
	assert.True(t, result.Bitrate > 0, "should have average bitrate more than 0")
	assert.True(t, result.HasAudio, "should have audio stream")
	assert.True(t, result.HasVideo, "should have video stream")
	assert.NoError(t, err)

	// HLS
	ctx, cancel = context.WithTimeout(context.Background(), watchTime)
	defer cancel()
	result, err = analyzer.Check(ctx, "https://cdn.theoplayer.com/video/big_buck_bunny/big_buck_bunny.m3u8")
	assert.True(t, result.Bitrate > 0, "should have average bitrate more than 0")
	assert.True(t, result.HasAudio, "should have audio stream")
	assert.True(t, result.HasVideo, "should have video stream")
	assert.NoError(t, err)

	// Regular file
	ctx, cancel = context.WithTimeout(context.Background(), watchTime)
	defer cancel()
	result, err = analyzer.Check(ctx, "https://speed.hetzner.de/1GB.bin")
	assert.True(t, result.Bitrate == 0, "should have average bitrate equal to 0")
	assert.False(t, result.HasAudio, "should not have audio stream")
	assert.False(t, result.HasVideo, "should not have video stream")
	assert.NoError(t, err)

	// Bad url to check
	ctx, cancel = context.WithTimeout(context.Background(), watchTime)
	defer cancel()
	result, err = analyzer.Check(ctx, "http://xxx")
	assert.True(t, result.Bitrate == 0, "should have average bitrate equal to 0")
	assert.False(t, result.HasAudio, "should not have audio stream")
	assert.False(t, result.HasVideo, "should not have video stream")
	assert.NoError(t, err)

	// Bad analyzer address and url to check
	ctx, cancel = context.WithTimeout(context.Background(), watchTime)
	defer cancel()
	result, err = New("256.256.256.256", handshakeTimeout).Check(ctx, "http://xxx")
	assert.True(t, result.Bitrate == 0, "should have average bitrate equal to 0")
	assert.False(t, result.HasAudio, "should not have audio stream")
	assert.False(t, result.HasVideo, "should not have video stream")
	assert.Error(t, err, "should return error for bad analyzer address")
}

func TestNewFake(t *testing.T) {
	assert.Exactly(t, &fakeAnalyzer{urlResultMap: map[string]Result{}}, NewFake(), "should initialize fake analyzer")
}

func TestAddResult(t *testing.T) {
	analyzer := NewFake()
	analyzer.AddResult("url1", Result{Bitrate: 1})
	analyzer.AddResult("url2", Result{Bitrate: 2})

	expected := map[string]Result{"url1": {Bitrate: 1}, "url2": {Bitrate: 2}}
	assert.Exactly(t, expected, analyzer.urlResultMap, "should add results to analyzer")
}

func TestFakeCheck(t *testing.T) {
	analyzer := NewFake()
	analyzer.AddResult("url1", Result{Bitrate: 1})

	result, err := analyzer.Check(context.Background(), "url1")
	assert.Exactly(t, Result{Bitrate: 1}, result, "should return that result")
	assert.NoError(t, err, "should not return error")
}
