package analyzer

import (
	"sync"
	"testing"
	"time"

	"m3u_merge_astra/util/logger"

	"github.com/stretchr/testify/assert"
)

func TestNew(t *testing.T) {
	log := logger.New(logger.DebugLevel)
	analyzer := New(log, "127.0.0.1", time.Second)
	assert.Exactly(t, log, analyzer.log, "should set logger for analyzer")
	assert.Exactly(t, "ws://127.0.0.1/api/", analyzer.url, "should set analyzer URL")
	assert.Exactly(t, time.Second, analyzer.dialer.HandshakeTimeout, "should set analyzer handshake timeout")
}

// Requires a running astra analyzer
func TestCheck(t *testing.T) {
	log := logger.New(logger.DebugLevel)
	handshakeTimeout := time.Second * 3
	watchTime := time.Second * 15
	maxAttempts := 3
	analyzer := New(log, "127.0.0.1:8001", handshakeTimeout)

	var wg sync.WaitGroup

	// MPEG-TS
	wg.Add(1)
	go func() {
		defer wg.Done()
		url := "https://filesamples.com/samples/video/ts/sample_1280x720_surfing_with_audio.ts"
		result, err := analyzer.Check(watchTime, maxAttempts, url)
		assert.True(t, result.Bitrate > 0, "should have average bitrate more than 0")
		assert.True(t, result.HasAudio, "should have audio stream")
		assert.True(t, result.HasVideo, "should have video stream")
		assert.NoError(t, err)
	}()

	// // HLS (results are not predictable, commented until analyzer supports HLS)
	// wg.Add(1)
	// go func() {
	// 	defer wg.Done()
	// 	url := "https://cdn.theoplayer.com/video/big_buck_bunny/big_buck_bunny.m3u8"
	// 	result, err := analyzer.Check(watchTime, maxAttempts, url)
	// 	assert.True(t, result.Bitrate > 0, "should have average bitrate more than 0")
	// 	assert.True(t, result.HasAudio, "should have audio stream")
	// 	assert.True(t, result.HasVideo, "should have video stream")
	// 	assert.NoError(t, err)
	// }()

	// Regular file
	wg.Add(1)
	go func() {
		defer wg.Done()
		url := "http://ipv4.download.thinkbroadband.com/1GB.zip"
		result, err := analyzer.Check(watchTime, maxAttempts, url)
		assert.True(t, result.Bitrate == 0, "should have average bitrate equal to 0")
		assert.False(t, result.HasAudio, "should not have audio stream")
		assert.False(t, result.HasVideo, "should not have video stream")
		assert.NoError(t, err)
	}()

	// Bad url to check
	wg.Add(1)
	go func() {
		defer wg.Done()
		url := "http://xxx"
		result, err := analyzer.Check(watchTime, maxAttempts, url)
		assert.True(t, result.Bitrate == 0, "should have average bitrate equal to 0")
		assert.False(t, result.HasAudio, "should not have audio stream")
		assert.False(t, result.HasVideo, "should not have video stream")
		assert.NoError(t, err)
	}()

	// Bad analyzer address and url to check
	wg.Add(1)
	go func() {
		defer wg.Done()
		url := "http://xxx"
		result, err := New(log, "256.256.256.256", handshakeTimeout).Check(watchTime, maxAttempts, url)
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

	result, err := analyzer.Check(time.Second, 1, "url1")
	assert.Exactly(t, Result{Bitrate: 1}, result, "should return that result")
	assert.NoError(t, err, "should not return error")
}
