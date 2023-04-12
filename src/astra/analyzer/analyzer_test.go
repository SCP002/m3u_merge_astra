package analyzer

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCheck(t *testing.T) {
	// MPEG-TS
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	result, err := Check(ctx, "127.0.0.1:8001", "https://tsduck.io/streams/brazil-isdb-tb/TS1globo.ts")
	assert.True(t, result.Bitrate > 0, "should have average bitrate more than 0")
	assert.NoError(t, err)

	// HLS
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	result, err = Check(ctx, "127.0.0.1:8001", "https://cdn.theoplayer.com/video/big_buck_bunny/big_buck_bunny.m3u8")
	assert.True(t, result.Bitrate > 0, "should have average bitrate more than 0")
	assert.NoError(t, err)

	// Bad url to check
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	result, err = Check(ctx, "127.0.0.1:8001", "http://xxx")
	assert.True(t, result.Bitrate == 0, "should have average bitrate equal to 0")
	assert.NoError(t, err)

	// Bad analyzer address and url to check
	ctx, cancel = context.WithTimeout(context.Background(), time.Second*5)
	defer cancel()
	result, err = Check(ctx, "256.256.256.256", "http://xxx")
	assert.True(t, result.Bitrate == 0, "should have average bitrate equal to 0")
	assert.Error(t, err, "should return error for bad analyzer address")
}
