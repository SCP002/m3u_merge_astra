package cfg

import (
	_ "embed"
	"io/fs"
	"os"
	"reflect"
	"regexp"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"

	yamlUtil "m3u_merge_astra/util/yaml"
)

//go:embed default.yaml
var defCfgBytes []byte

// Root represents root settings of the program
type Root struct {
	General General `koanf:"general"`
	M3U     M3U     `koanf:"m3u"`
	Streams Streams `koanf:"streams"`
}

// General represents general settings of the program
type General struct {
	// FullTranslit specifies if name transliteration should be used to detect which M3U channel corresponds a stream
	FullTranslit bool `koanf:"full_translit"`

	// FullTranslitMap represents source to destination character mapping.
	//
	// All symbols are lowercase as comparsion function will convert every character in a name to lowercase.
	//
	// Key: From. Value: To.
	FullTranslitMap map[string]string `koanf:"full_translit_map"`

	// SimilarTranslit specifies if name transliteration between visually similar characters should be used to detect
	// which M3U channel corresponds a stream.
	SimilarTranslit bool `koanf:"similar_translit"`

	// SimilarTranslitMap represents source to destination character mapping.
	//
	// All symbols are lowercase as comparsion function will convert every character in a name to lowercase.
	//
	// Key: From. Value: To.
	SimilarTranslitMap map[string]string `koanf:"similar_translit_map"`
}

// M3U represents M3U related settings of the program
type M3U struct {
	// RespTimeout represents M3U playlist URL response timeout
	RespTimeout time.Duration `koanf:"resp_timeout"`

	// ChannNameBlacklist represens the list of regular expressions.
	//
	// If any expression match name of a channel, this channel will be removed from M3U input before merging.
	ChannNameBlacklist []regexp.Regexp `koanf:"chann_name_blacklist"`

	// ChannGroupBlacklist represens the list of regular expressions.
	//
	// If any expression match group of a channel, this channel will be removed from M3U input before merging.
	//
	// It runs after replacing groups by ChannGroupMap so enter the appropriate values.
	ChannGroupBlacklist []regexp.Regexp `koanf:"chann_group_blacklist"`

	// ChannURLBlacklist represens the list of regular expressions.
	//
	// If any expression match URL of a channel, this channel will be removed from M3U input before merging.
	ChannURLBlacklist []regexp.Regexp `koanf:"chann_url_blacklist"`

	// ChannGroupMap represents invalid to valid M3U channel group mapping.
	//
	// Key: From. Value: To.
	ChannGroupMap map[string]string `koanf:"chann_group_map"`
}

// Streams represents astra streams related settings of the program
type Streams struct {
	// AddedPrefix represents new stream name prefix
	AddedPrefix string `koanf:"added_prefix"`

	// AddNew specifies if new astra streams should be added if streams does not contain M3U channel name
	AddNew bool `koanf:"add_new"`

	// AddGroupsToNew specifies if groups should be added to new astra streams
	AddGroupsToNew bool `koanf:"add_groups_to_new"`

	// GroupsCategoryForNew represents category name to use for groups of new astra streams
	GroupsCategoryForNew string `koanf:"groups_category_for_new"`

	// AddNewWithKnownInputs specifies if new astra streams should be added if streams contain M3U channel URL
	AddNewWithKnownInputs bool `koanf:"add_new_with_known_inputs"`

	// MakeNewEnabled specifies if new streams should be enabled.
	MakeNewEnabled bool `koanf:"make_new_enabled"`

	// NewType represents new stream type, can be one of two types:
	//
	// SPTS - Single-Program Transport Stream. Streaming channels to the end users over IP network.
	//
	// MPTS - Multi-Program Transport Stream. Preparing multiplexes to DVB modulators.
	NewType StreamType `koanf:"new_type"`

	// DisabledPrefix represents disabled stream name prefix
	DisabledPrefix string `koanf:"disabled_prefix"`

	// RemoveWithoutInputs specifies if streams without inputs should be removed.
	//
	// It has priority over DisableWithoutInputs.
	RemoveWithoutInputs bool `koanf:"remove_without_inputs"`

	// DisableWithoutInputs specifies if streams without inputs should be disabled.
	DisableWithoutInputs bool `koanf:"disable_without_inputs"`

	// EnableOnInputUpdate specifies if streams should be enabled if they got new inputs or inputs were updated
	// (but not removed).
	EnableOnInputUpdate bool `koanf:"enable_on_input_update"`

	// Rename specifies if astra streams should be renamed as M3U channels if their standartized names are equal
	Rename bool `koanf:"rename"`

	// AddNewInputs specifies if new inputs of astra streams should be added if such found in M3U channels
	AddNewInputs bool `koanf:"add_new_inputs"`

	// UniteInputs specifies if inputs of streams with the same names should be moved to the first stream found
	UniteInputs bool `koanf:"unite_inputs"`

	// HashCheckOnAddNewInputs specifies if new inputs of astra streams should be added even if M3U channel and
	// stream input only differ by hash (everything after #).
	HashCheckOnAddNewInputs bool `koanf:"hash_check_on_add_new_inputs"`

	// SortInputs specifies if inputs of astra streams should be sorted
	SortInputs bool `koanf:"sort_inputs"`

	// InputWeightToTypeMap represents Mapping of how high stream input should appear in the list after sorting.
	//
	// Any unspecified input will have weight of maximum - 1 (right before the last entry).
	InputWeightToTypeMap map[int]regexp.Regexp `koanf:"input_weight_to_type_map"`

	// UnknownInputWeight represents Default weight of unknown inputs
	UnknownInputWeight int `koanf:"unknown_input_weight"`

	// InputBlacklist represens the list of regular expressions.
	//
	// If any expression match URL of a stream's input, this input will be removed from astra streams before adding new
	// ones.
	InputBlacklist []regexp.Regexp `koanf:"input_blacklist"`

	// RemoveDuplicatedInputs specifies if inputs of astra streams which already present in config should be
	// removed.
	RemoveDuplicatedInputs bool `koanf:"remove_duplicated_inputs"`

	// RemoveDeadInputs specifies if inputs of astra streams which do not respond should be removed.
	//
	// Currently supports only HTTP(S).
	RemoveDeadInputs bool `koanf:"remove_dead_inputs"`

	// DeadInputsCheckBlacklist represens the list of regular expressions.
	//
	// If any expression match URL of a stream's input, this input will not be checked for availability.
	DeadInputsCheckBlacklist []regexp.Regexp `koanf:"dead_inputs_check_blacklist"`

	// InputMaxConns represents maximum amount of simultaneous connections to validate inputs of astra streams.
	//
	// Use more than 1 with caution. It may result in false positives if server consider frequent requests as spam.
	InputMaxConns int `koanf:"input_max_conns"`

	// InputRespTimeout represents astra stream input response timeout
	InputRespTimeout time.Duration `koanf:"input_resp_timeout"`

	// InputUpdateMap represens list of regular expression pairs.
	//
	// If any <From> expression match URL of astra stream's input, it will be replaced with URL from according M3U
	// channel if it matches the <To> expression.
	//
	// In most cases specified <From> and <To> should be identical.
	//
	// Using InputBlacklist with AddNewInputs instead will have almost the same end result but since old
	// URL's will be removed beforehand, original hash (#...) will be lost. Also it will be less clear which input was
	// being replaced.
	InputUpdateMap []UpdateRecord `koanf:"input_update_map"`

	// UpdateInputs specifies if inputs of astra streams should be updated with M3U channels according to
	// InputUpdateMap.
	UpdateInputs bool `koanf:"update_inputs"`

	// KeepInputHash specifies if old URL hash should be kept on updating inputs of astra streams
	KeepInputHash bool `koanf:"keep_input_hash"`

	// RemoveInputsByUpdateMap specifies if inputs of astra streams which match at least one InputUpdateMap.From
	// expression but not in M3U channels should be removed.
	RemoveInputsByUpdateMap bool `koanf:"remove_inputs_by_update_map"`

	// NameToInputHashMap represents mapping of stream name regular expression to stream input hash which should
	// be added.
	NameToInputHashMap []HashAddRule `koanf:"name_to_input_hash_map"`

	// GroupToInputHashMap represents mapping of stream group regular expression to stream input hash which should
	// be added.
	GroupToInputHashMap []HashAddRule `koanf:"group_to_input_hash_map"`

	// InputToInputHashMap represents mapping of stream input regular expression to stream input hash which should
	// be added.
	InputToInputHashMap []HashAddRule `koanf:"input_to_input_hash_map"`
}

// UpdateRecord represents astra stream input update rule
type UpdateRecord struct {
	From regexp.Regexp `koanf:"from"`
	To   regexp.Regexp `koanf:"to"`
}

// HashAddRule represents astra stream input hash adding rule
type HashAddRule struct {
	By   regexp.Regexp `koanf:"by"`
	Hash string        `koanf:"hash"`
}

// StreamType represents astra stream type
type StreamType string

const (
	SPTS StreamType = "spts"
	MPTS StreamType = "mpts"
)

// Init returns config instance and false if config at <cfgFilePath> already exist.
//
// If config does not exist, creates a default, returns empty instance and true.
func Init(log *logrus.Logger, cfgFilePath string) (Root, bool) {
	log.Info("Reading program config\n")

	ko := koanf.New(".")

	loadConfig := func() error {
		err := ko.Load(file.Provider(cfgFilePath), yaml.Parser())
		return errors.Wrap(err, "Load config")
	}

	writeDefConfig := func() error {
		err := os.WriteFile(cfgFilePath, defCfgBytes, 0644)
		return errors.Wrap(err, "Write default config")
	}

	// Load config file into koanf or create a new if not exist
	var root Root
	if err := loadConfig(); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			log.Info("Config file not found, creating a default\n")
			if err := writeDefConfig(); err != nil {
				log.Fatal(err)
			}
			return root, true
		} else {
			log.Fatal(err)
		}
	}

	// Decode loaded config file into structure
	decoder := mapstructure.ComposeDecodeHookFunc(
		// Compile regular expressions
		func(from, to reflect.Type, fromData any) (any, error) {
			if to == reflect.TypeOf(regexp.Regexp{}) {
				rxStr := reflect.ValueOf(fromData).String()
				return regexp.Compile(rxStr)
			}
			return fromData, nil
		},
		// Default decoders
		mapstructure.StringToTimeDurationHookFunc(),
		mapstructure.StringToSliceHookFunc(","),
	)
	metadata := mapstructure.Metadata{}
	err := ko.UnmarshalWithConf("", &root, koanf.UnmarshalConf{
		DecoderConfig: &mapstructure.DecoderConfig{
			DecodeHook:       decoder,
			ErrorUnused:      true,
			Metadata:         &metadata,
			Result:           &root,
			WeaklyTypedInput: true,
			ZeroFields:       true,
		},
	})
	if err := errors.Wrap(err, "Decode config"); err != nil {
		log.Fatal(err)
	}

	// Add known missing fields
	cfgBytes, err := os.ReadFile(cfgFilePath) // Broken if read with ko.Bytes("")
	if err != nil {
		log.Fatal(errors.Wrap(err, "Read config"))
	}
	defCfg := NewDefCfg()
	// v1.0.0 to v1.1.0
	knownField := "streams.add_groups_to_new"
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.AddGroupsToNew
		log.Infof("Adding missing field to config: %v: %v\n", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment:  []string{"Add groups to new astra streams?"},
			Key:          "add_groups_to_new",
			ValType:      yamlUtil.Scalar,
			Values:       []string{"false"},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.add_new", false, node); err != nil {
			log.Fatal(errors.Wrap(err, "Add missing field to config"))
		}
		root.Streams.AddGroupsToNew = defVal
	}
	// v1.0.0 to v1.1.0
	knownField = "streams.groups_category_for_new"
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.GroupsCategoryForNew
		log.Infof("Adding missing field to config: %v: %v\n", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment:  []string{"Category name to use for groups of new astra streams."},
			Key:          "groups_category_for_new",
			ValType:      yamlUtil.Scalar,
			Values:       []string{"'All'"},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.add_groups_to_new", false, node); err != nil {
			log.Fatal(errors.Wrap(err, "Add missing field to config"))
		}
		root.Streams.GroupsCategoryForNew = defVal
	}
	// v1.1.0 to v1.2.0
	knownField = "streams.enable_on_input_update"
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.EnableOnInputUpdate
		log.Infof("Adding missing field to config: %v: %v\n", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment:  []string{"Enable streams if they got new inputs or inputs were updated (but not removed)?"},
			Key:          "enable_on_input_update",
			ValType:      yamlUtil.Scalar,
			Values:       []string{"false"},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.disable_without_inputs", false, node); err != nil {
			log.Fatal(errors.Wrap(err, "Add missing field to config"))
		}
		root.Streams.EnableOnInputUpdate = defVal
	}
	if err = os.WriteFile(cfgFilePath, cfgBytes, 0644); err != nil {
		log.Fatal(errors.Wrap(err, "Write modified config"))
	}

	return root, false
}

// DefFullTranslitMap returns default full transliteration map.
//
// Used in tests.
func DefFullTranslitMap() map[string]string {
	return map[string]string{
		"а": "a", "б": "b", "в": "v", "г": "g", "д": "d", "е": "e", "ё": "yo", "ж": "zh", "з": "z", "и": "i",
		"й": "j", "к": "k", "л": "l", "м": "m", "н": "n", "о": "o", "п": "p", "р": "r", "с": "s", "т": "t",
		"у": "u", "ф": "f", "х": "x", "ц": "c", "ч": "ch", "ш": "sh", "щ": "shh", "ъ": "", "ы": "y", "ь": "",
		"э": "eh", "ю": "yu", "я": "ya",
	}
}

// DefSimilarTranslitMap returns default similar transliteration map.
//
// Used in tests.
func DefSimilarTranslitMap() map[string]string {
	return map[string]string{
		"а": "a", "б": "6", "в": "b", "е": "e", "з": "3", "к": "k", "м": "m", "н": "h", "о": "o", "р": "p",
		"с": "c", "т": "t", "у": "y", "х": "x",
	}
}

// NewDefCfg returns default config as written in "default.yaml" file
func NewDefCfg() Root {
	return Root{
		General: General{
			FullTranslit:       true,
			FullTranslitMap:    DefFullTranslitMap(),
			SimilarTranslit:    true,
			SimilarTranslitMap: DefSimilarTranslitMap(),
		},
		M3U: M3U{
			RespTimeout:         time.Second * 10,
			ChannNameBlacklist:  []regexp.Regexp(nil),
			ChannGroupBlacklist: []regexp.Regexp(nil),
			ChannURLBlacklist:   []regexp.Regexp(nil),
			ChannGroupMap:       map[string]string(nil),
		},
		Streams: Streams{
			AddedPrefix:              "_ADDED: ",
			AddNew:                   true,
			AddGroupsToNew:           false,
			GroupsCategoryForNew:     "All",
			AddNewWithKnownInputs:    false,
			MakeNewEnabled:           false,
			NewType:                  SPTS,
			DisabledPrefix:           "_DISABLED: ",
			RemoveWithoutInputs:      false,
			DisableWithoutInputs:     true,
			EnableOnInputUpdate:      false,
			Rename:                   false,
			AddNewInputs:             true,
			UniteInputs:              true,
			HashCheckOnAddNewInputs:  false,
			SortInputs:               true,
			InputWeightToTypeMap:     map[int]regexp.Regexp(nil),
			UnknownInputWeight:       50,
			InputBlacklist:           []regexp.Regexp(nil),
			RemoveDuplicatedInputs:   true,
			RemoveDeadInputs:         false,
			DeadInputsCheckBlacklist: []regexp.Regexp(nil),
			InputMaxConns:            1,
			InputRespTimeout:         time.Second * 10,
			InputUpdateMap:           []UpdateRecord(nil),
			UpdateInputs:             false,
			KeepInputHash:            true,
			RemoveInputsByUpdateMap:  false,
			NameToInputHashMap:       []HashAddRule(nil),
			GroupToInputHashMap:      []HashAddRule(nil),
			InputToInputHashMap:      []HashAddRule(nil),
		},
	}
}
