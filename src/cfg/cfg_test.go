package cfg

import (
	"m3u_merge_astra/util/copier"
	"m3u_merge_astra/util/file"
	"m3u_merge_astra/util/logger"
	"os"
	"path/filepath"
	"regexp"
	"testing"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/sirupsen/logrus"
	"github.com/stretchr/testify/assert"
)

func TestSimplifyAliases(t *testing.T) {
	cfg := General{
		NameAliasList: [][]string{
			{"Name 1", "Name 1 var 2", "Name 1 var 3"},
			{"Name_2", "Name_2_var_2"},
		},
	}
	cfgOriginal := copier.TestDeep(t, cfg)

	actual := cfg.SimplifyAliases()

	assert.NotSame(t, &cfg.NameAliasList, &actual, "should return copy of aliases")
	assert.Exactly(t, cfgOriginal, cfg, "should not modify the source config")

	expected := [][]string{
		{"name1", "name1var2", "name1var3"},
		{"name2", "name2var2"},
	}
	assert.Exactly(t, expected, actual, "should simplify aliases")
}

func TestDamagedConfigError(t *testing.T) {
	err := error(DamagedConfigError{MissingFields: []string{"a", "b"}})
	expected := "Existing program config is missing unexpected fields. " +
		"Create new config or add missing fields manually: a, b"
	assert.Exactly(t, expected, err.Error())
}

func TestBadRegexpError(t *testing.T) {
	err := error(BadRegexpError{Reason: "Bad", Regexp: *regexp.MustCompile(`.*`)})
	expected := "Bad; Regular expression: .*"
	assert.Exactly(t, expected, err.Error())
}

func TestInitDefault(t *testing.T) {
	log := logger.New(logrus.DebugLevel)

	path := filepath.Join(t.TempDir(), "m3u_merge_astra_init_test.yaml")

	// Test creation of the default config
	actual, isNewCfg, err := Init(log, path)

	assert.Exactly(t, Root{}, actual, "should return empty config")
	assert.True(t, isNewCfg, "should return true")
	assert.NoError(t, err, "should not return error")

	// Test reading of the default config
	actual, isNewCfg, err = Init(log, path)

	expected := NewDefCfg()

	assert.Exactly(t, expected, actual, "should return default config")
	assert.False(t, isNewCfg, "should return false")
	assert.NoError(t, err, "should not return error")
}

func TestInitAddMissing(t *testing.T) {
	log := logger.New(logrus.DebugLevel)

	path := filepath.Join(t.TempDir(), "m3u_merge_astra_init_test.yaml")

	// Test reading exising non-default config and adding missing fields
	err := file.Copy("init_input_test.yaml", path)
	assert.NoError(t, err, "should copy and overwrite previous test file")

	actual, isNewCfg, err := Init(log, path)

	expected := newTestConfig()

	assert.Exactly(t, expected, actual, "should return this config instance")
	assert.False(t, isNewCfg, "should return false")
	assert.NoError(t, err, "should not return error")

	// Check if missing fields were added to config file
	actualBytes, err := os.ReadFile(path)
	assert.NoError(t, err, "should read actual config bytes")

	expectedBytes, err := os.ReadFile("init_expected_test.yaml")
	assert.NoError(t, err, "should read expected config bytes")

	assert.Exactly(t, string(expectedBytes), string(actualBytes), "should add missing fields to config file")
}

func TestInitDamaged(t *testing.T) {
	log := logger.New(logrus.DebugLevel)

	path := filepath.Join(t.TempDir(), "m3u_merge_astra_init_test.yaml")

	// Test reading damaged config
	err := file.Copy("init_damaged_test.yaml", path)
	assert.NoError(t, err, "should copy and overwrite previous test file")

	_, isNewCfg, err := Init(log, path)

	expectedErr := DamagedConfigError{
		MissingFields: []string{ // All missing in default without known to be missing
			// "general.name_aliases", // <- Known
			"m3u.chann_name_blacklist",
			// "streams.add_groups_to_new", // <- Known
			"streams.input_weight_to_type_map",
			// "streams.remove_duplicated_inputs_by_rx_list", // <- Known
		},
	}

	assert.False(t, isNewCfg, "should return false")
	assert.Exactly(t, expectedErr, errors.UnwrapAll(err), "should return damaged config error")
}

func TestInitValidateCaptureGroups(t *testing.T) {
	log := logger.New(logrus.DebugLevel)

	path := filepath.Join(t.TempDir(), "m3u_merge_astra_init_test.yaml")

	// Test reading config with 'remove_duplicated_inputs_by_rx_list'
	err := file.Copy("init_validate_capture_groups_test.yaml", path)
	assert.NoError(t, err, "should copy and overwrite previous test file")

	_, isNewCfg, err := Init(log, path)

	expectedErr := BadRegexpError{
		Regexp: *regexp.MustCompile(`rx_without_capture_group`),
		Reason: "Expecting at least one capture group",
	}

	assert.False(t, isNewCfg, "should return false")
	assert.Exactly(t, expectedErr, errors.UnwrapAll(err), "should return bad regexp error")
}

func TestInitSimplifyAliases(t *testing.T) {
	log := logger.New(logrus.DebugLevel)

	path := filepath.Join(t.TempDir(), "m3u_merge_astra_init_test.yaml")

	// Test reading config with name aliases and simplification of them
	err := file.Copy("init_simplify_aliases_test.yaml", path)
	assert.NoError(t, err, "should copy and overwrite previous test file")

	actual, isNewCfg, err := Init(log, path)

	expected := [][]string{
		{"sample", "sampletv", "sampletelevisionchannel"},
		{"discoveryid", "discoveryinvestigation"},
	}

	assert.Exactly(t, expected, actual.General.SimpleNameAliasList, "actual config should contain these aliases")
	assert.False(t, isNewCfg, "should return false")
	assert.NoError(t, err, "should not return error")
}

func newTestConfig() Root {
	return Root{
		General: General{
			FullTranslit:        true,
			FullTranslitMap:     map[string]string{"ş": "ш", "\\n": ""},
			SimilarTranslit:     false,
			SimilarTranslitMap:  map[string]string(nil),
			NameAliases:         true,            // New field in v1.3.0
			NameAliasList:       [][]string(nil), // New field in v1.3.0
			SimpleNameAliasList: [][]string(nil), // Field for internal use
		},
		M3U: M3U{
			RespTimeout:         time.Second * 10,
			ChannNameBlacklist:  []regexp.Regexp{*regexp.MustCompile(`Nonsense TV`), *regexp.MustCompile(`(?i)^Test$`)},
			ChannGroupBlacklist: nil,
			ChannURLBlacklist: []regexp.Regexp{
				*regexp.MustCompile(`https?:\/\/filter_me\.com`),
				*regexp.MustCompile(`192\.168\.88\.14\/play`),
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
			NewKeepActive:           0, // New field in v1.4.0
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
				*regexp.MustCompile(`https?:\/\/filter_me\.com`),
				*regexp.MustCompile(`192\.168\.88\.14\/play`),
			},
			RemoveDuplicatedInputs:         true,
			RemoveDuplicatedInputsByRxList: []regexp.Regexp(nil), // New field in v1.4.0
			RemoveDeadInputs:               false,
			DeadInputsCheckBlacklist: []regexp.Regexp{
				*regexp.MustCompile(`https?:\/\/dont-check\.com\/play`),
				*regexp.MustCompile(`192\.168\.88\.`),
			},
			InputMaxConns:                     10,
			InputRespTimeout:                  time.Minute,
			UseAnalyzer:                       false,            // New field in v1.5.0
			AnalyzerAddr:                      "127.0.0.1:8001", // New field in v1.5.0
			AnalyzerWatchTime:                 time.Second * 10, // New field in v1.5.0
			AnalyzerBitrateThreshold:          1,                // New field in v1.5.0
			AnalyzerVideoOnlyBitrateThreshold: 1,                // New field in v1.5.0
			AnalyzerAudioOnlyBitrateThreshold: 1,                // New field in v1.5.0
			AnalyzerCCErrorsThreshold:         -1,               // New field in v1.5.0
			AnalyzerPCRErrorsThreshold:        -1,               // New field in v1.5.0
			AnalyzerPESErrorsThreshold:        -1,               // New field in v1.5.0
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
				{By: *regexp.MustCompile(`(?i)All: HD Channels$`), Hash: "buffer_time=10"},
				{By: *regexp.MustCompile(`(?i).*RADIO$`), Hash: "no_sync"},
			},
			InputToInputHashMap: []HashAddRule{
				{By: *regexp.MustCompile(`:8080`), Hash: "ua=VLC/3.0.9 LibVLC/3.0.9"},
				{By: *regexp.MustCompile(`^rts?p:\/\/`), Hash: "no_reload"},
			},
			NameToKeepActiveMap:  []KeepActiveAddRule(nil), // New field in v1.4.0
			GroupToKeepActiveMap: []KeepActiveAddRule(nil), // New field in v1.4.0
			InputToKeepActiveMap: []KeepActiveAddRule(nil), // New field in v1.4.0
		},
	}
}
