package analyzer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCheck(t *testing.T) {
	analyzerAddr := "127.0.0.1:8001"
	handshakeTimeout := time.Second * 3
	watchTime := time.Second * 5

	// MPEG-TS
	ctx, cancel := context.WithTimeout(context.Background(), watchTime)
	defer cancel()
	url := "https://tsduck.io/streams/brazil-isdb-tb/TS1globo.ts"
	result, err := Check(ctx, handshakeTimeout, analyzerAddr, url)
	assert.True(t, result.Bitrate > 0, "should have average bitrate more than 0")
	assert.True(t, result.HasAudio, "should have audio stream")
	assert.True(t, result.HasVideo, "should have video stream")
	assert.NoError(t, err)

	// HLS
	ctx, cancel = context.WithTimeout(context.Background(), watchTime)
	defer cancel()
	url = "https://cdn.theoplayer.com/video/big_buck_bunny/big_buck_bunny.m3u8"
	result, err = Check(ctx, handshakeTimeout, analyzerAddr, url)
	assert.True(t, result.Bitrate > 0, "should have average bitrate more than 0")
	assert.True(t, result.HasAudio, "should have audio stream")
	assert.True(t, result.HasVideo, "should have video stream")
	assert.NoError(t, err)

	// Regular file
	ctx, cancel = context.WithTimeout(context.Background(), watchTime)
	defer cancel()
	url = "https://speed.hetzner.de/1GB.bin"
	result, err = Check(ctx, handshakeTimeout, analyzerAddr, url)
	assert.True(t, result.Bitrate == 0, "should have average bitrate equal to 0")
	assert.False(t, result.HasAudio, "should not have audio stream")
	assert.False(t, result.HasVideo, "should not have video stream")
	assert.NoError(t, err)

	// Bad url to check
	ctx, cancel = context.WithTimeout(context.Background(), watchTime)
	defer cancel()
	url = "http://xxx"
	result, err = Check(ctx, handshakeTimeout, analyzerAddr, url)
	assert.True(t, result.Bitrate == 0, "should have average bitrate equal to 0")
	assert.False(t, result.HasAudio, "should not have audio stream")
	assert.False(t, result.HasVideo, "should not have video stream")
	assert.NoError(t, err)

	// Bad analyzer address and url to check
	ctx, cancel = context.WithTimeout(context.Background(), watchTime)
	defer cancel()
	url = "http://xxx"
	result, err = Check(ctx, handshakeTimeout, "256.256.256.256", url)
	assert.True(t, result.Bitrate == 0, "should have average bitrate equal to 0")
	assert.False(t, result.HasAudio, "should not have audio stream")
	assert.False(t, result.HasVideo, "should not have video stream")
	assert.Error(t, err, "should return error for bad analyzer address")
}
