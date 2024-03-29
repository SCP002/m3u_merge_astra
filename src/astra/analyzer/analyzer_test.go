package analyzer

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	analyzer := New("127.0.0.1", time.Second)
	assert.Exactly(t, "ws://127.0.0.1/api/", analyzer.url, "should set analyzer URL")
	assert.Exactly(t, time.Second, analyzer.dialer.HandshakeTimeout, "should set analyzer handshake timeout")
}

// Requires a running astra analyzer, use test_check.sh
func TestCheck(t *testing.T) {
	handshakeTimeout := time.Second * 3
	watchTime := time.Second * 20
	analyzer := New("127.0.0.1:8001", handshakeTimeout)

	var wg sync.WaitGroup

	// MPEG-TS
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), watchTime)
		defer cancel()
		url := "https://filesamples.com/samples/video/ts/sample_1280x720_surfing_with_audio.ts"
		result, err := analyzer.Check(ctx, url)
		assert.True(t, result.Bitrate > 0, "should have average bitrate more than 0")
		assert.True(t, result.HasAudio, "should have audio stream")
		assert.True(t, result.HasVideo, "should have video stream")
		assert.NoError(t, err)
	}()

	// HLS
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), watchTime)
		defer cancel()
		url := "https://cdn.theoplayer.com/video/big_buck_bunny/big_buck_bunny.m3u8"
		result, err := analyzer.Check(ctx, url)
		assert.True(t, result.Bitrate > 0, "should have average bitrate more than 0")
		assert.True(t, result.HasAudio, "should have audio stream")
		assert.True(t, result.HasVideo, "should have video stream")
		assert.NoError(t, err)
	}()

	// Regular file
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), watchTime)
		defer cancel()
		url := "https://speed.hetzner.de/1GB.bin"
		result, err := analyzer.Check(ctx, url)
		assert.True(t, result.Bitrate == 0, "should have average bitrate equal to 0")
		assert.False(t, result.HasAudio, "should not have audio stream")
		assert.False(t, result.HasVideo, "should not have video stream")
		assert.NoError(t, err)
	}()

	// Bad url to check
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), watchTime)
		defer cancel()
		url := "http://xxx"
		result, err := analyzer.Check(ctx, url)
		assert.True(t, result.Bitrate == 0, "should have average bitrate equal to 0")
		assert.False(t, result.HasAudio, "should not have audio stream")
		assert.False(t, result.HasVideo, "should not have video stream")
		assert.NoError(t, err)
	}()

	// Bad analyzer address and url to check
	wg.Add(1)
	go func() {
		defer wg.Done()
		ctx, cancel := context.WithTimeout(context.Background(), watchTime)
		defer cancel()
		url := "http://xxx"
		result, err := New("256.256.256.256", handshakeTimeout).Check(ctx, url)
		assert.True(t, result.Bitrate == 0, "should have average bitrate equal to 0")
		assert.False(t, result.HasAudio, "should not have audio stream")
		assert.False(t, result.HasVideo, "should not have video stream")
		assert.Error(t, err, "should return error for bad analyzer address")
	}()

	wg.Wait()
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
