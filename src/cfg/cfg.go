package cfg

import (
	_ "embed"
	"fmt"
	"io/fs"
	"os"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/cockroachdb/errors"
	"github.com/knadh/koanf"
	"github.com/knadh/koanf/parsers/yaml"
	"github.com/knadh/koanf/providers/file"
	"github.com/mitchellh/mapstructure"
	"github.com/samber/lo"

	"m3u_merge_astra/util/logger"
	"m3u_merge_astra/util/parse"
	"m3u_merge_astra/util/simplify"
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

	// NameAliases specifies if name aliases should be used to detect which M3U channel corresponds a stream
	NameAliases bool `koanf:"name_aliases"`

	// NameAliasList represents the list of lists.
	//
	// Names defined here will be considered identical to any other name in the same nested group.
	//
	// During comparsion, names will be simplified (lowercase, no special characters except the '+' sign), but not
	// transliterated.
	NameAliasList [][]string `koanf:"name_alias_list"`

	// NameAliasList represents simplified version of NameAliasList.
	//
	// This field will not be included into config and used to improve performance of util/compare.IsNameSame().
	SimpleNameAliasList [][]string
}

// SimplifyAliases returns simplified alias list in <c>.
//
// Made to improve performance of util/compare.IsNameSame().
func (c General) SimplifyAliases() (out [][]string) {
	for _, set := range c.NameAliasList {
		out = append(out, lo.Map(set, func(alias string, _ int) string {
			return simplify.Name(alias)
		}))
	}
	return
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

	// NewKeepActive represents delay before stop stream if no active connections for new streams.
	NewKeepActive int `koanf:"new_keep_active"`

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

	// RemoveDuplicatedInputsByRxList represens the list of regular expressions.
	//
	// If any first capture group (anything surrounded by the first '()') of regular expression match URL of input of a
	// stream, any other inputs of that stream which first capture group is the same will be removed from stream.
	//
	// This setting is not controlled by 'remove_duplicated_inputs'.
	RemoveDuplicatedInputsByRxList []regexp.Regexp `koanf:"remove_duplicated_inputs_by_rx_list"`

	// RemoveDeadInputs specifies if inputs of astra streams which do not respond or give invalid response should be
	// removed.
	//
	// Supports HTTP(S), enable 'use_analyzer' option for more.
	//
	// It has priority over DisableDeadInputs.
	RemoveDeadInputs bool `koanf:"remove_dead_inputs"`

	// DisableDeadInputs specifies if inputs of astra streams which do not respond or give invalid response should be
	// disabled.
	//
	// Supports HTTP(S), enable 'use_analyzer' option for more.
	DisableDeadInputs bool `koanf:"disable_dead_inputs"`

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

	// UseAnalyzer specifies if astra analyzer (astra --analyze -p <port>) should be used to check for dead inputs.
	//
	// Supports HTTP(S), UDP, RTP, RTSP.
	UseAnalyzer bool `koanf:"use_analyzer"`

	// AnalyzerAddr represents astra analyzer address in format of 'host:port'
	AnalyzerAddr string `koanf:"analyzer_addr"`

	// AnalyzerWatchTime represents amount of time astra analyzer should spend collecting results
	AnalyzerWatchTime time.Duration `koanf:"analyzer_watch_time"`

	// AnalyzerWatchTime represents average bitrate threshold in kbit/s for stream inputs.
	//
	// If astra analyzer will return bitrate lower than specified threshold, input will be cosidered dead.
	AnalyzerBitrateThreshold int `koanf:"analyzer_bitrate_threshold"`

	// AnalyzerVideoOnlyBitrateThreshold represents average bitrate threshold in kbit/s for stream inputs without audio.
	//
	// If astra analyzer will return bitrate lower than specified threshold, input will be cosidered dead.
	AnalyzerVideoOnlyBitrateThreshold int `koanf:"analyzer_video_only_bitrate_threshold"`

	// AnalyzerAudioOnlyBitrateThreshold represents average bitrate threshold in kbit/s for stream inputs without video.
	//
	// If astra analyzer will return bitrate lower than specified threshold, input will be cosidered dead.
	AnalyzerAudioOnlyBitrateThreshold int `koanf:"analyzer_audio_only_bitrate_threshold"`

	// AnalyzerCCErrorsThreshold represents average amount of CC errors for stream inputs.
	//
	// If astra analyzer will return amount of CC errors higher than specified threshold, input will be cosidered dead.
	//
	// Set to negative value to disable this check.
	AnalyzerCCErrorsThreshold int `koanf:"analyzer_cc_errors_threshold"`

	// AnalyzerPCRErrorsThreshold represents average amount of PCR errors for stream inputs.
	//
	// If astra analyzer will return amount of PCR errors higher than specified threshold, input will be cosidered dead.
	//
	// Set to negative value to disable this check.
	AnalyzerPCRErrorsThreshold int `koanf:"analyzer_pcr_errors_threshold"`

	// AnalyzerPESErrorsThreshold represents average amount of PES errors for stream inputs.
	//
	// If astra analyzer will return amount of PES errors higher than specified threshold, input will be cosidered dead.
	//
	// Set to negative value to disable this check.
	AnalyzerPESErrorsThreshold int `koanf:"analyzer_pes_errors_threshold"`

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
	// Stream groups should be defined to match expressions in the form of 'Category: Group'.
	GroupToInputHashMap []HashAddRule `koanf:"group_to_input_hash_map"`

	// InputToInputHashMap represents mapping of stream input regular expression to stream input hash which should
	// be added.
	InputToInputHashMap []HashAddRule `koanf:"input_to_input_hash_map"`

	// NameToKeepActiveMap represents mapping of stream name regular expression to 'keep active' setting of stream
	// which should be set.
	//
	// Only first matching rule applies per stream in the priority: By inputs -> By name -> By group.
	NameToKeepActiveMap []KeepActiveAddRule `koanf:"name_to_keep_active_map"`

	// GroupToKeepActiveMap represents mapping of stream group regular expression to 'keep active' setting of stream
	// which should be set.
	//
	// Only first matching rule applies per stream in the priority: By inputs -> By name -> By group.
	GroupToKeepActiveMap []KeepActiveAddRule `koanf:"group_to_keep_active_map"`

	// InputToKeepActiveMap represents mapping of stream input regular expression to 'keep active' setting of stream
	// which should be set.
	//
	// Only first matching rule applies per stream in the priority: By inputs -> By name -> By group.
	//
	// Setting will be set if at least one input matches the <By> expression.
	InputToKeepActiveMap []KeepActiveAddRule `koanf:"input_to_keep_active_map"`
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

// KeepActiveAddRule represents astra stream 'keep active' adding rule
type KeepActiveAddRule struct {
	By         regexp.Regexp `koanf:"by"`
	KeepActive int           `koanf:"keep_active"`
}

// StreamType represents astra stream type
type StreamType string

const (
	SPTS StreamType = "spts"
	MPTS StreamType = "mpts"
)

// DamagedConfigError represents error thrown if program config is missing unexpected fields
type DamagedConfigError struct {
	MissingFields []string
}

// Error is used to satisfy golang error interface
func (e DamagedConfigError) Error() string {
	msg := "Existing program config is missing unexpected fields. Create new config or add missing fields manually"
	return fmt.Sprintf("%v: %v", msg, strings.Join(e.MissingFields, ", "))
}

// BadRegexpError represents error thrown if program config has invalid regular expression
type BadRegexpError struct {
	Reason string
	Regexp regexp.Regexp
}

// Error is used to satisfy golang error interface
func (e BadRegexpError) Error() string {
	return fmt.Sprintf("%v; Regular expression: %v", e.Reason, e.Regexp.String())
}

// Init returns config instance and false if config at <cfgFilePath> already exist.
//
// If config does not exist, creates a default, returns empty instance and true.
//
// Builds simplified version of name aliases to Root.General.SimpleNameAliasList.
//
// Can return errors defined in this package: DamagedConfigError, BadRegexpError.
func Init(log *logger.Logger, cfgFilePath string) (Root, bool, error) {
	log.Info("Reading program config")

	ko := koanf.New(".")

	loadConfig := func() error {
		return ko.Load(file.Provider(cfgFilePath), yaml.Parser())
	}

	writeDefConfig := func() error {
		return os.WriteFile(cfgFilePath, defCfgBytes, 0644)
	}

	// Load config file into koanf or create a new if not exist
	var root Root
	if err := loadConfig(); err != nil {
		if errors.Is(err, fs.ErrNotExist) {
			log.Info("Config file not found, creating a default")
			if err := writeDefConfig(); err != nil {
				return root, false, errors.Wrap(err, "Write default config")
			}
			return root, true, nil
		} else {
			return root, false, errors.Wrap(err, "Load config")
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
			DecodeHook:           decoder,
			ErrorUnused:          true,
			IgnoreUntaggedFields: true,
			Metadata:             &metadata,
			Result:               &root,
			WeaklyTypedInput:     true,
			ZeroFields:           true,
		},
	})
	if err != nil {
		return root, false, errors.Wrap(err, "Decode config")
	}

	// Check if config is damaged (there are more missing fields than was added since v1.0.0)
	knownFields := []string{
		/*  0 */ "streams.add_groups_to_new",
		/*  1 */ "streams.groups_category_for_new",
		/*  2 */ "streams.enable_on_input_update",
		/*  3 */ "general.name_aliases",
		/*  4 */ "general.name_alias_list",
		/*  5 */ "streams.remove_duplicated_inputs_by_rx_list",
		/*  6 */ "streams.new_keep_active",
		/*  7 */ "streams.name_to_keep_active_map",
		/*  8 */ "streams.group_to_keep_active_map",
		/*  9 */ "streams.input_to_keep_active_map",
		/* 10 */ "streams.use_analyzer",
		/* 11 */ "streams.analyzer_addr",
		/* 12 */ "streams.analyzer_watch_time",
		/* 13 */ "streams.analyzer_bitrate_threshold",
		/* 14 */ "streams.analyzer_video_only_bitrate_threshold",
		/* 15 */ "streams.analyzer_audio_only_bitrate_threshold",
		/* 16 */ "streams.analyzer_cc_errors_threshold",
		/* 17 */ "streams.analyzer_pcr_errors_threshold",
		/* 18 */ "streams.analyzer_pes_errors_threshold",
		/* 19 */ "streams.disable_dead_inputs",
	}
	missingFields, _ := lo.Difference(metadata.Unset, knownFields)
	internalFields := []string{
		"general.SimpleNameAliasList",
	}
	// Remove internal fields from missing to prevent false positive DamagedConfigError
	missingFields, _ = lo.Difference(missingFields, internalFields)
	if len(missingFields) > 0 {
		err := DamagedConfigError{MissingFields: missingFields}
		return root, false, errors.Wrap(err, "Check config")
	}

	// Add missing known fields
	cfgBytes, err := os.ReadFile(cfgFilePath) // Broken if read with ko.Bytes("")
	if err != nil {
		return root, false, errors.Wrap(err, "Read config")
	}
	defCfg := NewDefCfg()

	// v1.0.0 to v1.1.0
	knownField := knownFields[0]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.AddGroupsToNew
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment:  []string{"Add groups to new astra streams?"},
			Data:         yamlUtil.Scalar{Key: parse.LastPathItem(knownField, "."), Value: strconv.FormatBool(defVal)},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.add_new", false, node); err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.Streams.AddGroupsToNew = defVal
	}
	// v1.0.0 to v1.1.0
	knownField = knownFields[1]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.GroupsCategoryForNew
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment:  []string{"Category name to use for groups of new astra streams."},
			Data:         yamlUtil.Scalar{Key: parse.LastPathItem(knownField, "."), Value: fmt.Sprintf("'%v'", defVal)},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.add_groups_to_new", false, node); err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.Streams.GroupsCategoryForNew = defVal
	}
	// v1.1.0 to v1.2.0
	knownField = knownFields[2]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.EnableOnInputUpdate
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment:  []string{"Enable streams if they got new inputs or inputs were updated (but not removed)?"},
			Data:         yamlUtil.Scalar{Key: parse.LastPathItem(knownField, "."), Value: strconv.FormatBool(defVal)},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.disable_without_inputs", false, node); err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.Streams.EnableOnInputUpdate = defVal
	}
	// v1.2.0 to v1.3.0
	knownField = knownFields[3]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.General.NameAliases
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			HeadComment: []string{"Use name aliases list to detect which M3U channel corresponds a stream?"},
			Data:        yamlUtil.Scalar{Key: parse.LastPathItem(knownField, "."), Value: strconv.FormatBool(defVal)},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "general.similar_translit_map", true, node); err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.General.NameAliases = defVal
	}
	// v1.2.0 to v1.3.0
	knownField = knownFields[4]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.General.NameAliasList
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment: []string{
				"List of lists.",
				"Names defined here will be considered identical to any other name in the same nested group.",
				"During comparsion, names will be simplified (lowercase, no special characters except the '+' sign),",
				"but not transliterated.",
			},
			Data: yamlUtil.NestedList{
				Key: parse.LastPathItem(knownField, "."),
				Tree: yamlUtil.ValueTree{
					Children: []yamlUtil.ValueTree{
						{
							Value: yamlUtil.Value{Value: "'Sample'", Commented: true},
							Children: []yamlUtil.ValueTree{
								{Value: yamlUtil.Value{Value: "'Sample TV'", Commented: true}},
								{Value: yamlUtil.Value{Value: "'Sample Television Channel'", Commented: true}},
							},
						},
						{
							Value: yamlUtil.Value{Value: "'Discovery ID'", Commented: true},
							Children: []yamlUtil.ValueTree{
								{Value: yamlUtil.Value{Value: "'Discovery Investigation'", Commented: true}},
							},
						},
					},
				},
			},
			EndNewline: true,
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "general.name_aliases", false, node); err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.General.NameAliasList = defVal
	}
	// v1.3.0 to v1.4.0
	knownField = knownFields[5]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.RemoveDuplicatedInputsByRxList
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment: []string{
				"List of regular expressions.",
				"If any first capture group (anything surrounded by the first '()') of regular expression match URL " +
					"of input of a",
				"stream, any other inputs of that stream which first capture group is the same will be removed from " +
					"stream.",
				"",
				"This setting is not controlled by 'remove_duplicated_inputs'.",
			},
			Data: yamlUtil.List{
				Key: parse.LastPathItem(knownField, "."),
				Values: []yamlUtil.Value{
					{Value: `'^.*:\/\/([^#?/]*)' # By host`, Commented: true},
					{Value: `'^.*:\/\/.*?\/([^#?]*)' # By path`, Commented: true},
				}},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.remove_duplicated_inputs", false, node); err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.Streams.RemoveDuplicatedInputsByRxList = defVal
	}
	// v1.3.0 to v1.4.0
	knownField = knownFields[6]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.NewKeepActive
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment:  []string{"Delay before stop stream if no active connections for new streams."},
			Data:         yamlUtil.Scalar{Key: parse.LastPathItem(knownField, "."), Value: strconv.Itoa(defVal)},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.new_type", false, node); err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.Streams.NewKeepActive = defVal
	}
	// v1.3.0 to v1.4.0
	knownField = knownFields[7]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.NameToKeepActiveMap
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment: []string{
				"Mapping of stream name regular expression to 'keep active' setting of stream which should be set.",
				"",
				"Only first matching rule applies per stream in the priority: By inputs -> By name -> By group.",
			},
			Data: yamlUtil.Sequence{
				Key: parse.LastPathItem(knownField, "."),
				Sets: [][]yamlUtil.Pair{
					{
						{Key: "by", Value: "'[- _]HD$'", Commented: true},
						{Key: "keep_active", Value: "10", Commented: true},
					},
					{
						{Key: "by", Value: "'[- _]FM$'", Commented: true},
						{Key: "keep_active", Value: "0", Commented: true},
					},
				},
			},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.input_to_input_hash_map", true, node); err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.Streams.NameToKeepActiveMap = defVal
	}
	// v1.3.0 to v1.4.0
	knownField = knownFields[8]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.GroupToKeepActiveMap
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment: []string{
				"Mapping of stream group regular expression to 'keep active' setting of stream which should be set.",
				"",
				"Only first matching rule applies per stream in the priority: By inputs -> By name -> By group.",
			},
			Data: yamlUtil.Sequence{
				Key: parse.LastPathItem(knownField, "."),
				Sets: [][]yamlUtil.Pair{
					{
						{Key: "by", Value: "'(?i)All: HD Channels$'", Commented: true},
						{Key: "keep_active", Value: "10", Commented: true},
					},
					{
						{Key: "by", Value: "'(?i).*RADIO$'", Commented: true},
						{Key: "keep_active", Value: "0", Commented: true},
					},
				},
			},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.name_to_keep_active_map", true, node); err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.Streams.GroupToKeepActiveMap = defVal
	}
	// v1.3.0 to v1.4.0
	knownField = knownFields[9]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.InputToKeepActiveMap
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment: []string{
				"Mapping of stream input regular expression to 'keep active' setting of stream which should be set.",
				"",
				"Only first matching rule applies per stream in the priority: By inputs -> By name -> By group.",
				"",
				"Setting will be set if at least one input matches the 'by' expression.",
			},
			Data: yamlUtil.Sequence{
				Key: parse.LastPathItem(knownField, "."),
				Sets: [][]yamlUtil.Pair{
					{
						{Key: "by", Value: "':8080'", Commented: true},
						{Key: "keep_active", Value: "10", Commented: true},
					},
					{
						{Key: "by", Value: `'^rts?p:\/\/'`, Commented: true},
						{Key: "keep_active", Value: "0", Commented: true},
					},
				},
			},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.group_to_keep_active_map", true, node); err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.Streams.InputToKeepActiveMap = defVal
	}
	// v1.4.0 to v1.5.0
	knownField = knownFields[10]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.UseAnalyzer
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment: []string{
				"Use astra analyzer (astra --analyze -p <port>) to check for dead inputs?",
				"",
				"Supports HTTP(S), UDP, RTP, RTSP.",
			},
			Data: yamlUtil.Scalar{Key: parse.LastPathItem(knownField, "."), Value: strconv.FormatBool(defVal)},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.input_resp_timeout", false, node); err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.Streams.UseAnalyzer = defVal
	}
	// v1.4.0 to v1.5.0
	knownField = knownFields[11]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.AnalyzerAddr
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment:  []string{"Astra analyzer address in format of 'host:port'."},
			Data:         yamlUtil.Scalar{Key: parse.LastPathItem(knownField, "."), Value: fmt.Sprintf("'%v'", defVal)},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.use_analyzer", false, node); err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.Streams.AnalyzerAddr = defVal
	}
	// v1.4.0 to v1.5.0
	knownField = knownFields[12]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.AnalyzerWatchTime
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment:  []string{"Amount of time astra analyzer should spend collecting results."},
			Data:         yamlUtil.Scalar{Key: parse.LastPathItem(knownField, "."), Value: fmt.Sprintf("'%v'", defVal)},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.analyzer_addr", false, node); err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.Streams.AnalyzerWatchTime = defVal
	}
	// v1.4.0 to v1.5.0
	knownField = knownFields[13]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.AnalyzerBitrateThreshold
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment: []string{
				"Average bitrate threshold in kbit/s for stream inputs.",
				"",
				"If astra analyzer will return bitrate lower than specified threshold, input will be cosidered dead.",
			},
			Data: yamlUtil.Scalar{Key: parse.LastPathItem(knownField, "."), Value: strconv.Itoa(defVal)},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.analyzer_watch_time", false, node); err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.Streams.AnalyzerBitrateThreshold = defVal
	}
	// v1.4.0 to v1.5.0
	knownField = knownFields[14]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.AnalyzerVideoOnlyBitrateThreshold
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment: []string{
				"Average bitrate threshold in kbit/s for stream inputs without audio.",
				"",
				"If astra analyzer will return bitrate lower than specified threshold, input will be cosidered dead.",
			},
			Data: yamlUtil.Scalar{Key: parse.LastPathItem(knownField, "."), Value: strconv.Itoa(defVal)},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.analyzer_bitrate_threshold", false, node); err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.Streams.AnalyzerVideoOnlyBitrateThreshold = defVal
	}
	// v1.4.0 to v1.5.0
	knownField = knownFields[15]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.AnalyzerAudioOnlyBitrateThreshold
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment: []string{
				"Average bitrate threshold in kbit/s for stream inputs without video.",
				"",
				"If astra analyzer will return bitrate lower than specified threshold, input will be cosidered dead.",
			},
			Data: yamlUtil.Scalar{Key: parse.LastPathItem(knownField, "."), Value: strconv.Itoa(defVal)},
		}
		cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.analyzer_video_only_bitrate_threshold", false, node)
		if err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.Streams.AnalyzerAudioOnlyBitrateThreshold = defVal
	}
	// v1.4.0 to v1.5.0
	knownField = knownFields[16]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.AnalyzerCCErrorsThreshold
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment: []string{
				"Average amount of CC errors for stream inputs.",
				"",
				"If astra analyzer will return amount of CC errors higher than specified threshold, " +
					"input will be cosidered dead.",
				"",
				"Set to negative value to disable this check.",
			},
			Data: yamlUtil.Scalar{Key: parse.LastPathItem(knownField, "."), Value: strconv.Itoa(defVal)},
		}
		cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.analyzer_audio_only_bitrate_threshold", false, node)
		if err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.Streams.AnalyzerCCErrorsThreshold = defVal
	}
	// v1.4.0 to v1.5.0
	knownField = knownFields[17]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.AnalyzerPCRErrorsThreshold
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment: []string{
				"Average amount of PCR errors for stream inputs.",
				"",
				"If astra analyzer will return amount of PCR errors higher than specified threshold, " +
					"input will be cosidered dead.",
				"",
				"Set to negative value to disable this check.",
			},
			Data: yamlUtil.Scalar{Key: parse.LastPathItem(knownField, "."), Value: strconv.Itoa(defVal)},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.analyzer_cc_errors_threshold", false, node); err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.Streams.AnalyzerPCRErrorsThreshold = defVal
	}
	// v1.4.0 to v1.5.0
	knownField = knownFields[18]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.AnalyzerPESErrorsThreshold
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment: []string{
				"Average amount of PES errors for stream inputs.",
				"",
				"If astra analyzer will return amount of PES errors higher than specified threshold, " +
					"input will be cosidered dead.",
				"",
				"Set to negative value to disable this check.",
			},
			Data: yamlUtil.Scalar{Key: parse.LastPathItem(knownField, "."), Value: strconv.Itoa(defVal)},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.analyzer_pcr_errors_threshold", false, node); err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.Streams.AnalyzerPESErrorsThreshold = defVal
	}
	// v1.4.0 to v1.5.0
	knownField = knownFields[19]
	if lo.Contains(metadata.Unset, knownField) {
		defVal := defCfg.Streams.DisableDeadInputs
		log.Infof("Adding missing field to config: %v: %v", knownField, defVal)
		node := yamlUtil.Node{
			StartNewline: true,
			HeadComment: []string{
				"Disable inputs of astra streams which do not respond or give invalid response?",
				"Supports HTTP(S), enable 'use_analyzer' option for more.",
			},
			Data: yamlUtil.Scalar{Key: parse.LastPathItem(knownField, "."), Value: strconv.FormatBool(defVal)},
		}
		if cfgBytes, err = yamlUtil.Insert(cfgBytes, "streams.remove_dead_inputs", false, node); err != nil {
			return root, false, errors.Wrap(err, "Add missing field to config")
		}
		root.Streams.DisableDeadInputs = defVal
	}

	// Validate amount of capture groups
	for _, rx := range root.Streams.RemoveDuplicatedInputsByRxList {
		if rx.NumSubexp() < 1 {
			msg := "Expecting at least one capture group"
			return root, false, errors.Wrap(BadRegexpError{Regexp: rx, Reason: msg}, "Validate config")
		}
	}

	// Write modified config
	if err = os.WriteFile(cfgFilePath, cfgBytes, 0644); err != nil {
		return root, false, errors.Wrap(err, "Write modified config")
	}

	// Build simple aliases list
	if root.General.NameAliases {
		root.General.SimpleNameAliasList = root.General.SimplifyAliases()
	}

	return root, false, nil
}

// DefFullTranslitMap returns default full transliteration map.
//
// Used in tests.
func DefFullTranslitMap() map[string]string {
	return map[string]string{
		"а": "a", "б": "b", "в": "v", "г": "g", "д": "d", "е": "e", "ё": "yo", "ж": "zh", "з": "z", "и": "i",
		"й": "j", "к": "k", "л": "l", "м": "m", "н": "n", "о": "o", "п": "p", "р": "r", "с": "s", "т": "t",
		"у": "u", "ф": "f", "х": "h", "ц": "c", "ч": "ch", "ш": "sh", "щ": "shh", "ъ": "", "ы": "y", "ь": "",
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
			FullTranslit:        true,
			FullTranslitMap:     DefFullTranslitMap(),
			SimilarTranslit:     true,
			SimilarTranslitMap:  DefSimilarTranslitMap(),
			NameAliases:         true,
			NameAliasList:       [][]string(nil),
			SimpleNameAliasList: [][]string(nil),
		},
		M3U: M3U{
			RespTimeout:         time.Second * 10,
			ChannNameBlacklist:  []regexp.Regexp(nil),
			ChannGroupBlacklist: []regexp.Regexp(nil),
			ChannURLBlacklist:   []regexp.Regexp(nil),
			ChannGroupMap:       map[string]string(nil),
		},
		Streams: Streams{
			AddedPrefix:                       "_ADDED: ",
			AddNew:                            true,
			AddGroupsToNew:                    false,
			GroupsCategoryForNew:              "All",
			AddNewWithKnownInputs:             false,
			MakeNewEnabled:                    false,
			NewType:                           SPTS,
			NewKeepActive:                     0,
			DisabledPrefix:                    "_DISABLED: ",
			RemoveWithoutInputs:               false,
			DisableWithoutInputs:              true,
			EnableOnInputUpdate:               false,
			Rename:                            false,
			AddNewInputs:                      true,
			UniteInputs:                       true,
			HashCheckOnAddNewInputs:           false,
			SortInputs:                        true,
			InputWeightToTypeMap:              map[int]regexp.Regexp(nil),
			UnknownInputWeight:                50,
			InputBlacklist:                    []regexp.Regexp(nil),
			RemoveDuplicatedInputs:            true,
			RemoveDuplicatedInputsByRxList:    []regexp.Regexp(nil),
			RemoveDeadInputs:                  false,
			DisableDeadInputs:                 false,
			DeadInputsCheckBlacklist:          []regexp.Regexp(nil),
			InputMaxConns:                     1,
			InputRespTimeout:                  time.Second * 10,
			UseAnalyzer:                       false,
			AnalyzerAddr:                      "127.0.0.1:8001",
			AnalyzerWatchTime:                 time.Second * 20,
			AnalyzerBitrateThreshold:          1,
			AnalyzerVideoOnlyBitrateThreshold: 1,
			AnalyzerAudioOnlyBitrateThreshold: 1,
			AnalyzerCCErrorsThreshold:         -1,
			AnalyzerPCRErrorsThreshold:        -1,
			AnalyzerPESErrorsThreshold:        -1,
			InputUpdateMap:                    []UpdateRecord(nil),
			UpdateInputs:                      false,
			KeepInputHash:                     true,
			RemoveInputsByUpdateMap:           false,
			NameToInputHashMap:                []HashAddRule(nil),
			GroupToInputHashMap:               []HashAddRule(nil),
			InputToInputHashMap:               []HashAddRule(nil),
			NameToKeepActiveMap:               []KeepActiveAddRule(nil),
			GroupToKeepActiveMap:              []KeepActiveAddRule(nil),
			InputToKeepActiveMap:              []KeepActiveAddRule(nil),
		},
	}
}
