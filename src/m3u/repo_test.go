package m3u

import (
	"testing"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"

	"m3u_merge_astra/cfg"
	"m3u_merge_astra/util/logger"
	"m3u_merge_astra/util/tw"
)

// newDefRepo returns new repository initialized with defaults
func newDefRepo() repo {
	return NewRepo(logger.New(logrus.DebugLevel), tw.New(), cfg.NewDefCfg())
}

func TestNewRepo(t *testing.T) {
	log := logger.New(logrus.DebugLevel)
	tw := tw.New()
	cfg := cfg.NewDefCfg()

	assert.Exactly(t, newDefRepo(), NewRepo(log, tw, cfg))
}

func TestLog(t *testing.T) {
	assert.Exactly(t, logger.New(logrus.DebugLevel), newDefRepo().Log())
}

func TestTW(t *testing.T) {
	assert.Exactly(t, tw.New(), newDefRepo().TW())
}

func TestCfg(t *testing.T) {
	assert.Exactly(t, cfg.NewDefCfg(), newDefRepo().Cfg())
}
