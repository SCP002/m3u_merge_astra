package m3u

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"m3u_merge_astra/cfg"
	"m3u_merge_astra/util/logger"
)

// newDefRepo returns new repository initialized with defaults
func newDefRepo() repo {
	return NewRepo(logger.New(logrus.DebugLevel), cfg.NewDefCfg())
}

func TestNewRepo(t *testing.T) {
	log := logger.New(logrus.DebugLevel)
	cfg := cfg.NewDefCfg()

	assert.Exactly(t, newDefRepo(), NewRepo(log, cfg))
}

func TestLog(t *testing.T) {
	assert.Exactly(t, logger.New(logrus.DebugLevel), newDefRepo().Log())
}

func TestCfg(t *testing.T) {
	assert.Exactly(t, cfg.NewDefCfg(), newDefRepo().Cfg())
}
