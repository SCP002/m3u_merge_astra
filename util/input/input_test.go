package input

import (
	"strings"
	"testing"

	"m3u_merge_astra/util/logger"

	"github.com/stretchr/testify/assert"
)

func TestAskYesNo(t *testing.T) {
	log := logger.New(logger.DebugLevel)

	in := strings.NewReader("Y\n")
	answer := AskYesNo(log, in, "prompt\n")
	assert.True(t, answer)

	in.Reset("y\n")
	answer = AskYesNo(log, in, "prompt\n")
	assert.True(t, answer)

	in.Reset("N\n")
	answer = AskYesNo(log, in, "prompt\n")
	assert.False(t, answer)

	in.Reset("n\n")
	answer = AskYesNo(log, in, "prompt\n")
	assert.False(t, answer)
}

func TestAsk(t *testing.T) {
	log := logger.New(logger.DebugLevel)

	in := strings.NewReader(" 0 \n")
	answer := ask(log, in, false, "prompt\n", func(s string) bool {
		assert.Exactly(t, " 0 \n", s, "callback argument should have this value")
		return false
	})
	assert.Exactly(t, " 0 \n", answer, "should read this value")

	in.Reset(" 0 \n")
	answer = ask(log, in, true, "prompt\n", func(s string) bool {
		assert.Exactly(t, "0", s, "callback argument should have trimmed value")
		return false
	})
	assert.Exactly(t, "0", answer, "should read and trim input")
}
