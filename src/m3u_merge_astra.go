package main

import (
	"fmt"
	"os"

	"m3u_merge_astra/astra"
	"m3u_merge_astra/astra/analyzer"
	"m3u_merge_astra/cfg"
	"m3u_merge_astra/cli"
	"m3u_merge_astra/m3u"
	"m3u_merge_astra/merge"
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
		fmt.Println("v1.5.1")
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

	// Read astra config
	log.Info("Reading astra config")
	astraCfg, err := astra.ReadCfg(flags.AstraCfgInput)
	if err != nil {
		log.Fatal(err)
	}

	// Fetch M3U channels
	log.Info("Fetching M3U channels")
	httpClient := network.NewHttpClient(cfg.M3U.RespTimeout)
	m3uResp, err := openuri.Open(flags.M3UPath, openuri.WithHTTPClient(httpClient))
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

	astraCfg.Streams = astraRepo.RemoveNamePrefixes(astraCfg.Streams)
	astraCfg.Streams = astraRepo.Sort(astraCfg.Streams)
	if cfg.Streams.Rename {
		astraCfg.Streams = mergeRepo.RenameStreams(astraCfg.Streams, m3uChannels)
	}
	if len(cfg.Streams.InputBlacklist) > 0 {
		astraCfg.Streams = astraRepo.RemoveBlockedInputs(astraCfg.Streams)
	}
	if cfg.Streams.RemoveDuplicatedInputs {
		astraCfg.Streams = astraRepo.RemoveDuplicatedInputs(astraCfg.Streams)
	}
	if len(cfg.Streams.RemoveDuplicatedInputsByRxList) > 0 {
		astraCfg.Streams = astraRepo.RemoveDuplicatedInputsByRx(astraCfg.Streams)
	}
	if cfg.Streams.UpdateInputs {
		astraCfg.Streams = mergeRepo.UpdateInputs(astraCfg.Streams, m3uChannels)
	}
	if cfg.Streams.RemoveInputsByUpdateMap {
		astraCfg.Streams = mergeRepo.RemoveInputsByUpdateMap(astraCfg.Streams, m3uChannels)
	}
	if cfg.Streams.AddNewInputs {
		astraCfg.Streams = mergeRepo.AddNewInputs(astraCfg.Streams, m3uChannels)
	}
	if cfg.Streams.UniteInputs {
		astraCfg.Streams = astraRepo.UniteInputs(astraCfg.Streams)
	}
	if cfg.Streams.SortInputs {
		astraCfg.Streams = astraRepo.SortInputs(astraCfg.Streams)
	}
	if cfg.Streams.AddNew {
		astraCfg.Streams = mergeRepo.AddNewStreams(astraCfg.Streams, m3uChannels)
	}
	astraCfg.Categories = astraRepo.AddNewGroups(astraCfg.Categories, astraCfg.Streams)
	if cfg.Streams.RemoveDeadInputs {
		httpClient := network.NewHttpClient(cfg.Streams.InputRespTimeout)
		analyzer := analyzer.New(cfg.Streams.AnalyzerAddr, cfg.Streams.InputRespTimeout)
		astraCfg.Streams = astraRepo.RemoveDeadInputs(httpClient, analyzer, astraCfg.Streams)
	} else if cfg.Streams.DisableDeadInputs {
		httpClient := network.NewHttpClient(cfg.Streams.InputRespTimeout)
		analyzer := analyzer.New(cfg.Streams.AnalyzerAddr, cfg.Streams.InputRespTimeout)
		astraCfg.Streams = astraRepo.DisableDeadInputs(httpClient, analyzer, astraCfg.Streams)
	}
	if !slice.IsAllEmpty(cfg.Streams.NameToInputHashMap, cfg.Streams.GroupToInputHashMap,
		cfg.Streams.InputToInputHashMap) {
		astraCfg.Streams = astraRepo.AddHashes(astraCfg.Streams)
	}
	if !slice.IsAllEmpty(cfg.Streams.NameToKeepActiveMap, cfg.Streams.GroupToKeepActiveMap,
		cfg.Streams.InputToKeepActiveMap) {
		astraCfg.Streams = astraRepo.SetKeepActive(astraCfg.Streams)
	}
	if cfg.Streams.RemoveWithoutInputs {
		astraCfg.Streams = astraRepo.RemoveWithoutInputs(astraCfg.Streams)
	} else if cfg.Streams.DisableWithoutInputs {
		astraCfg.Streams = astraRepo.DisableWithoutInputs(astraCfg.Streams)
	}
	astraCfg.Streams = astraRepo.AddNamePrefixes(astraCfg.Streams)

	// Write astra config
	log.Info("Writing astra config")
	err = astra.WriteCfg(astraCfg, flags.AstraCfgOutput)
	if err != nil {
		log.Fatal(err)
	}
}
