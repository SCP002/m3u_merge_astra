package cfg

import (
	"m3u_merge_astra/util/file"
	"m3u_merge_astra/util/logger"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestInit(t *testing.T) {
	log := logger.New(logrus.DebugLevel)

	path := filepath.Join(os.TempDir(), "m3u_merge_astra_init_test.yaml")
	defer os.Remove(path)

	// Test creation of the default config
	actual, isNewCfg := Init(log, path)

	assert.Exactly(t, Root{}, actual, "should return empty config")
	assert.True(t, isNewCfg, "should return true")

	// Test reading of the default config
	actual, isNewCfg = Init(log, path)

	expected := NewDefCfg()

	assert.Exactly(t, expected, actual, "should return default config")
	assert.False(t, isNewCfg, "should return false")

	// Test reading exising non-default config and adding missing fields
	err := file.Copy("init_input_test.yaml", path)
	assert.NoError(t, err, "should copy and overwrite previous test file")

	actual, isNewCfg = Init(log, path)

	expected = Root{
		General: General{
			FullTranslit:       true,
			FullTranslitMap:    map[string]string{"ş": "ш", "\\n": ""},
			SimilarTranslit:    false,
			SimilarTranslitMap: map[string]string(nil),
		},
		M3U: M3U{
			RespTimeout:         time.Second * 10,
			ChannNameBlacklist:  []regexp.Regexp{*regexp.MustCompile(`Nonsense TV`), *regexp.MustCompile(`^Test.*`)},
			ChannGroupBlacklist: nil,
			ChannURLBlacklist: []regexp.Regexp{
				*regexp.MustCompile(`https?://filter_me\.com`),
				*regexp.MustCompile(`192\.168\.88\.14/play`),
			},
			ChannGroupMap: map[string]string{"": "General", "-": "General", "For kids": "Kids"},
		},
		Streams: Streams{
			AddedPrefix:             "",
			AddNew:                  true,
			AddGroupsToNew:          false, // New field in v1.1.0
			GroupsCategoryForNew:    "All", // New field in v1.1.0
			AddNewWithKnownInputs:   false,
			MakeNewEnabled:          true,
			NewType:                 MPTS,
			DisabledPrefix:          "_'DISABLED': ",
			RemoveWithoutInputs:     true,
			DisableWithoutInputs:    false,
			EnableOnInputUpdate:     false, // New field in v1.2.0
			Rename:                  false,
			AddNewInputs:            true,
			UniteInputs:             false,
			HashCheckOnAddNewInputs: true,
			SortInputs:              false,
			InputWeightToTypeMap: map[int]regexp.Regexp{
				-1: *regexp.MustCompile(`192.\168\.88\.`),
				99: *regexp.MustCompile(`least_reliable\.tv`),
			},
			UnknownInputWeight: 50,
			InputBlacklist: []regexp.Regexp{
				*regexp.MustCompile(`https?://filter_me\.com`),
				*regexp.MustCompile(`192\.168\.88\.14/play`),
			},
			RemoveDuplicatedInputs: true,
			RemoveDeadInputs:       false,
			DeadInputsCheckBlacklist: []regexp.Regexp{
				*regexp.MustCompile(`https?://dont-check\.com/play`),
				*regexp.MustCompile(`192\.168\.88\.`),
			},
			InputMaxConns:    10,
			InputRespTimeout: time.Minute,
			InputUpdateMap: []UpdateRecord{
				{From: *regexp.MustCompile(`127\.0\.0\.1`), To: *regexp.MustCompile(`127\.0\.0\.1`)},
				{From: *regexp.MustCompile(`some_url\.com`), To: *regexp.MustCompile(`some_url\.com`)},
			},
			UpdateInputs:            true,
			KeepInputHash:           false,
			RemoveInputsByUpdateMap: true,
			NameToInputHashMap: []HashAddRule{
				{By: *regexp.MustCompile(`[- _]HD$`), Hash: "buffer_time=10"},
				{By: *regexp.MustCompile(`[- _]FM$`), Hash: "no_sync"},
			},
			GroupToInputHashMap: []HashAddRule{
				{By: *regexp.MustCompile(`HD Channels`), Hash: "buffer_time=10"},
				{By: *regexp.MustCompile(`Radio`), Hash: "no_sync"},
			},
			InputToInputHashMap: []HashAddRule{
				{By: *regexp.MustCompile(`:8080`), Hash: "ua=VLC/3.0.9 LibVLC/3.0.9"},
				{By: *regexp.MustCompile(`^rts?p://`), Hash: "no_reload"},
			},
		},
	}

	assert.Exactly(t, expected, actual, "should return this config instance")
	assert.False(t, isNewCfg, "should return false")

	// Check if missing fields were added to config file
	actualBytes, err := os.ReadFile(path)
	assert.NoError(t, err, "should read actual config bytes")

	expectedBytes, err := os.ReadFile("init_expected_test.yaml")
	assert.NoError(t, err, "should read expected config bytes")

	assert.Exactly(t, expectedBytes, actualBytes, "should add missing fields to config file")
}
