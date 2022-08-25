package cli

import (
	"os"
	"testing"

	goFlags "github.com/jessevdk/go-flags"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	os.Args = []string{""}
	_, err := Parse()
	assert.NoError(t, err, "should not return error")

	os.Args = []string{"", "--help"}
	_, err = Parse()
	assert.True(t, IsErrOfType(err, goFlags.ErrHelp), "should return help error")

	os.Args = []string{"", "--version"}
	flags, err := Parse()
	assert.NoError(t, err, "should not return error")
	assert.True(t, flags.Version, "flag should be specified")

	os.Args = []string{"", "--programCfgPath=/cfg/path", "--m3uPath=/m3u/path", "--astraCfgInput=stdio",
		"--astraCfgOutput=/astra/output"}
	flags, err = Parse()
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, "/cfg/path", flags.ProgramCfgPath, "flag should have this value")
	assert.Exactly(t, "/m3u/path", flags.M3UPath, "flag should have this value")
	assert.Exactly(t, string(Stdio), flags.AstraCfgInput, "flag should have this value")
	assert.Exactly(t, "/astra/output", flags.AstraCfgOutput, "flag should have this value")
}

func TestIsErrOfType(t *testing.T) {
	assert.True(t, IsErrOfType(&goFlags.Error{Type: goFlags.ErrUnknown}, goFlags.ErrUnknown))
	assert.False(t, IsErrOfType(&goFlags.Error{Type: goFlags.ErrUnknown}, goFlags.ErrHelp))
}
