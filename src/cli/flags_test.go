package cli

import (
	"os"
	"testing"

	goFlags "github.com/jessevdk/go-flags"
	pLog "github.com/phuslu/log"
	"github.com/stretchr/testify/assert"
)

func TestParse(t *testing.T) {
	os.Args = []string{""}
	flags, err := Parse()
	assert.NoError(t, err, "should not return error")
	assert.False(t, flags.Noninteractive, "noninteracive flag should be false if not specified")
	assert.Empty(t, flags.LogFile, "logFile flag should be empty if not specified")

	os.Args = []string{"", "--help"}
	_, err = Parse()
	assert.True(t, IsErrOfType(err, goFlags.ErrHelp), "should return help error")

	os.Args = []string{"", "--version"}
	flags, err = Parse()
	assert.NoError(t, err, "should not return error")
	assert.True(t, flags.Version, "flag should be specified")

	os.Args = []string{"", "--logLevel=-1"}
	_, err = Parse()
	assert.Error(t, err, "should return error for negative log level")
	assert.True(t, IsErrOfType(err, goFlags.ErrMarshal), "should return marshal error")

	os.Args = []string{"", "--logLevel=2"}
	flags, err = Parse()
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, pLog.DebugLevel, flags.LogLevel, "flag should have this value")

	os.Args = []string{"", "--logLevel=999"}
	flags, err = Parse()
	assert.NoError(t, err, "should not return error")
	assert.Exactly(t, pLog.Level(999), flags.LogLevel, "flag should have this value")

	os.Args = []string{"", "--noninteractive", "--logFile=/log", "--programCfgPath=/cfg/path", "--m3uPath=/m3u/path",
		"--astraAddr=http://127.0.0.1:8005", "--astraUser=admin", "--astraPwd=admin"}
	flags, err = Parse()
	assert.NoError(t, err, "should not return error")
	assert.True(t, flags.Noninteractive, "flag should have this value")
	assert.Exactly(t, pLog.InfoLevel, flags.LogLevel, "flag should have this value")
	assert.Exactly(t, "/log", flags.LogFile, "flag should have this value")
	assert.Exactly(t, "/cfg/path", flags.ProgramCfgPath, "flag should have this value")
	assert.Exactly(t, "/m3u/path", flags.M3UPath, "flag should have this value")
	assert.Exactly(t, "http://127.0.0.1:8005", flags.AstraAddr, "flag should have this value")
	assert.Exactly(t, "admin", flags.AstraUser, "flag should have this value")
	assert.Exactly(t, "admin", flags.AstraPwd, "flag should have this value")
}

func TestIsErrOfType(t *testing.T) {
	assert.True(t, IsErrOfType(&goFlags.Error{Type: goFlags.ErrUnknown}, goFlags.ErrUnknown))
	assert.False(t, IsErrOfType(&goFlags.Error{Type: goFlags.ErrUnknown}, goFlags.ErrHelp))
}
