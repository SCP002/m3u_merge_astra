package main

import (
	"fmt"
	"os"

	"m3u_merge_astra/astra"
	"m3u_merge_astra/astra/analyzer"
	"m3u_merge_astra/astra/api"
	"m3u_merge_astra/cfg"
	"m3u_merge_astra/cli"
	"m3u_merge_astra/m3u"
	"m3u_merge_astra/merge"
	"m3u_merge_astra/util/copier"
	"m3u_merge_astra/util/input"
	"m3u_merge_astra/util/logger"
	"m3u_merge_astra/util/network"
	"m3u_merge_astra/util/slice"

	"github.com/adampresley/sigint"
	goFlags "github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
	"github.com/utahta/go-openuri"
)

func main() {
	// Init default logger
	log := logger.New(logrus.FatalLevel)

	// Parse command line arguments
	flags, err := cli.Parse()
	if flags.Version {
		fmt.Println("v2.1.0")
		os.Exit(0)
	}
	if cli.IsErrOfType(err, goFlags.ErrHelp) {
		// Help message will be prined by go-flags
		os.Exit(0)
	}
	if err != nil {
		log.Fatal(err)
	}

	// Set log level
	log.SetLevel(flags.LogLevel)

	// Register SIGINT and SIGTERM event handler
	sigint.Listen(func() {
		log.Info("SIGINT or SIGTERM signal received, shutting down")
		os.Exit(0)
	})

	// Read program config
	cfg, isNewCfg, err := cfg.Init(log, flags.ProgramCfgPath)
	if err != nil {
		log.Fatal(err)
	}
	if isNewCfg {
		log.Infof("New config is written to %v, please verify it and start this program again", flags.ProgramCfgPath)
		os.Exit(0)
	}

	// Fetch astra config
	log.Info("Fetching astra config")
	apiHttpClient := network.NewHttpClient(cfg.General.AstraAPIRespTimeout)
	apiHandler := api.NewHandler(log, apiHttpClient, flags.AstraAddr, flags.AstraUser, flags.AstraPwd)
	astraCfg, err := apiHandler.FetchCfg()
	if err != nil {
		log.Fatal(err)
	}

	// Fetch M3U channels
	log.Info("Fetching M3U channels")
	m3uHttpClient := network.NewHttpClient(cfg.M3U.RespTimeout)
	m3uResp, err := openuri.Open(flags.M3UPath, openuri.WithHTTPClient(m3uHttpClient))
	if err != nil {
		log.Fatal(err)
	}
	defer m3uResp.Close()

	// Parse and preprocess M3U channels
	m3uRepo := m3u.NewRepo(log, cfg)

	m3uChannels := m3uRepo.Parse(m3uResp)
	m3uChannels = m3uRepo.Sort(m3uChannels)
	if len(cfg.M3U.ChannGroupMap) > 0 {
		m3uChannels = m3uRepo.ReplaceGroups(m3uChannels)
	}
	if !slice.IsAllEmpty(cfg.M3U.ChannNameBlacklist, cfg.M3U.ChannGroupBlacklist, cfg.M3U.ChannURLBlacklist) {
		m3uChannels = m3uRepo.RemoveBlocked(m3uChannels)
	}

	// Update astra streams with data from M3U channels and run extra operations such as sorting or disabling streams
	// without inputs
	astraRepo := astra.NewRepo(log, cfg)
	mergeRepo := merge.NewRepo(log, cfg)

	modifiedStreams := copier.MustDeep(astraCfg.Streams)
	modifiedStreams = astraRepo.RemoveNamePrefixes(modifiedStreams)
	modifiedStreams = astraRepo.Sort(modifiedStreams)
	if cfg.Streams.Rename {
		modifiedStreams = mergeRepo.RenameStreams(modifiedStreams, m3uChannels)
	}
	if len(cfg.Streams.InputBlacklist) > 0 {
		modifiedStreams = astraRepo.RemoveBlockedInputs(modifiedStreams)
	}
	if cfg.Streams.RemoveDuplicatedInputs {
		modifiedStreams = astraRepo.RemoveDuplicatedInputs(modifiedStreams)
	}
	if len(cfg.Streams.RemoveDuplicatedInputsByRxList) > 0 {
		modifiedStreams = astraRepo.RemoveDuplicatedInputsByRx(modifiedStreams)
	}
	if cfg.Streams.UpdateInputs {
		modifiedStreams = mergeRepo.UpdateInputs(modifiedStreams, m3uChannels)
	}
	if cfg.Streams.RemoveInputsByUpdateMap {
		modifiedStreams = mergeRepo.RemoveInputsByUpdateMap(modifiedStreams, m3uChannels)
	}
	if cfg.Streams.AddNewInputs {
		modifiedStreams = mergeRepo.AddNewInputs(modifiedStreams, m3uChannels)
	}
	if cfg.Streams.UniteInputs {
		modifiedStreams = astraRepo.UniteInputs(modifiedStreams)
	}
	if cfg.Streams.SortInputs {
		modifiedStreams = astraRepo.SortInputs(modifiedStreams)
	}
	if cfg.Streams.AddNew {
		modifiedStreams = mergeRepo.AddNewStreams(modifiedStreams, m3uChannels)
	}
	if len(cfg.Streams.DisableAllButOneInputByRxList) > 0 {
		modifiedStreams = astraRepo.DisableAllButOneInputByRx(modifiedStreams)
	}
	if cfg.Streams.RemoveDeadInputs {
		httpClient := network.NewHttpClient(cfg.Streams.InputRespTimeout)
		analyzer := analyzer.New(log, cfg.Streams.AnalyzerAddr, cfg.Streams.InputRespTimeout)
		modifiedStreams = astraRepo.RemoveDeadInputs(httpClient, analyzer, modifiedStreams)
	} else if cfg.Streams.DisableDeadInputs {
		httpClient := network.NewHttpClient(cfg.Streams.InputRespTimeout)
		analyzer := analyzer.New(log, cfg.Streams.AnalyzerAddr, cfg.Streams.InputRespTimeout)
		modifiedStreams = astraRepo.DisableDeadInputs(httpClient, analyzer, modifiedStreams)
	}
	if !slice.IsAllEmpty(cfg.Streams.NameToInputHashMap, cfg.Streams.GroupToInputHashMap,
		cfg.Streams.InputToInputHashMap) {
		modifiedStreams = astraRepo.AddHashes(modifiedStreams)
	}
	if !slice.IsAllEmpty(cfg.Streams.NameToKeepActiveMap, cfg.Streams.GroupToKeepActiveMap,
		cfg.Streams.InputToKeepActiveMap) {
		modifiedStreams = astraRepo.SetKeepActive(modifiedStreams)
	}
	if cfg.Streams.RemoveWithoutInputs {
		modifiedStreams = astraRepo.RemoveWithoutInputs(modifiedStreams)
	} else if cfg.Streams.DisableWithoutInputs {
		modifiedStreams = astraRepo.DisableWithoutInputs(modifiedStreams)
	}
	modifiedStreams = astraRepo.AddNamePrefixes(modifiedStreams)

	// Update astra categories
	modifiedCats := copier.MustDeep(astraCfg.Categories)
	if cfg.General.MergeCategories {
		modifiedCats = astraRepo.MergeCategories(modifiedCats)
	}
	modifiedCats = astraRepo.UpdateCategories(modifiedCats, modifiedStreams)

	// Search for changes
	changedCatMap := astraRepo.ChangedCategories(astraCfg.Categories, modifiedCats)
	changedStreams := astraRepo.ChangedStreams(astraCfg.Streams, modifiedStreams)

	// Sending changes to astra
	sendChangesAllowed := true
	if !flags.Noninteractive {
		sendChangesAllowed = input.AskYesNo(log, os.Stdin, "Send changes to astra (Y/N)? ")
	}
	if sendChangesAllowed {
		apiHandler.SetCategories(changedCatMap)
		apiHandler.SetStreams(changedStreams)
	}
}
