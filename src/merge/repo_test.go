package merge

import (
	"reflect"
	"testing"

	"github.com/google/go-cmp/cmp"
	"github.com/stretchr/testify/assert"

	"m3u_merge_astra/cfg"
	"m3u_merge_astra/util/logger"
)

// newDefRepo returns new repository initialized with defaults
func newDefRepo() repo {
	return NewRepo(logger.New(logger.DebugLevel), cfg.NewDefCfg())
}

func TestNewRepo(t *testing.T) {
	log := logger.New(logger.DebugLevel)
	cfg := cfg.NewDefCfg()

	compareOpt := cmp.FilterPath(func(p cmp.Path) bool {
		return p.Last().String() == ".Formatter"
	}, cmp.Ignore())
	exportedOpt := cmp.Exporter(func(t reflect.Type) bool {
		return true
	})
	reposEqual := cmp.Equal(newDefRepo(), NewRepo(log, cfg), compareOpt, exportedOpt)
	assert.True(t, reposEqual)
}

func TestLog(t *testing.T) {
	compareOpt := cmp.FilterPath(func(p cmp.Path) bool {
		return p.Last().String() == ".Formatter"
	}, cmp.Ignore())
	exportedOpt := cmp.Exporter(func(t reflect.Type) bool {
		return true
	})
	loggersEqual := cmp.Equal(logger.New(logger.DebugLevel), newDefRepo().Log(), compareOpt, exportedOpt)
	assert.True(t, loggersEqual)
}

func TestCfg(t *testing.T) {
	assert.Exactly(t, cfg.NewDefCfg(), newDefRepo().Cfg())
}
