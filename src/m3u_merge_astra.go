package main

import (
	"fmt"
	"os"

	"m3u_merge_astra/astra"
	"m3u_merge_astra/cfg"
	"m3u_merge_astra/cli"
	"m3u_merge_astra/m3u"
	"m3u_merge_astra/merge"
	"m3u_merge_astra/util/logger"
	"m3u_merge_astra/util/network"
	"m3u_merge_astra/util/tw"

	goFlags "github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
	"github.com/utahta/go-openuri"
)

func main() {
	// Init logger
	log := logger.New(logrus.InfoLevel)

	// Parse command line arguments
	log.Debug("Parsing command line arguments\n")
	flags, err := cli.Parse()
	if flags.Version {
		fmt.Println("v1.3.1")
		os.Exit(0)
	}
	if cli.IsErrOfType(err, goFlags.ErrHelp) {
		// Help message will be prined by go-flags
		os.Exit(0)
	}
	if err != nil {
		log.Panic(err)
	}

	// Read program config
	cfg, isNewCfg, err := cfg.Init(log, flags.ProgramCfgPath)
	if err != nil {
		log.Panic(err)
	}
	if isNewCfg {
		log.Infof("New config is written to %v, please verify it and start this program again", flags.ProgramCfgPath)
		os.Exit(0)
	}

	// Read astra config
	log.Info("Reading astra config\n")
	astraCfg, err := astra.ReadCfg(flags.AstraCfgInput)
	if err != nil {
		log.Panic(err)
	}

	// Fetch M3U channels
	log.Info("Fetching M3U channels\n")
	httpClient := network.NewHttpClient(cfg.M3U.RespTimeout)
	m3uResp, err := openuri.Open(flags.M3UPath, openuri.WithHTTPClient(httpClient))
	if err != nil {
		log.Panic(err)
	}
	defer m3uResp.Close()

	// Init table writer
	tw := tw.New()

	// Parse and preprocess M3U channels
	m3uRepo := m3u.NewRepo(log, tw, cfg)

	m3uChannels := m3uRepo.Parse(m3uResp)
	m3uChannels = m3uRepo.Sort(m3uChannels)
	m3uChannels = m3uRepo.ReplaceGroups(m3uChannels)
	m3uChannels = m3uRepo.RemoveBlocked(m3uChannels)

	// Update astra streams with data from M3U channels and run extra operations such as sorting or disabling streams
	// without inputs
	astraRepo := astra.NewRepo(log, tw, cfg)
	mergeRepo := merge.NewRepo(log, tw, cfg)

	astraCfg.Streams = astraRepo.RemoveNamePrefixes(astraCfg.Streams)
	astraCfg.Streams = astraRepo.Sort(astraCfg.Streams)
	if cfg.Streams.Rename {
		astraCfg.Streams = mergeRepo.RenameStreams(astraCfg.Streams, m3uChannels)
	}
	astraCfg.Streams = astraRepo.RemoveBlockedInputs(astraCfg.Streams)
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
		astraCfg.Streams = astraRepo.RemoveDeadInputs(httpClient, astraCfg.Streams, true)
	}
	astraCfg.Streams = astraRepo.AddHashes(astraCfg.Streams)
	if cfg.Streams.RemoveWithoutInputs {
		astraCfg.Streams = astraRepo.RemoveWithoutInputs(astraCfg.Streams)
	} else if cfg.Streams.DisableWithoutInputs {
		astraCfg.Streams = astraRepo.DisableWithoutInputs(astraCfg.Streams)
	}
	astraCfg.Streams = astraRepo.AddNamePrefixes(astraCfg.Streams)

	// Write astra config
	log.Info("Writing astra config\n")
	err = astra.WriteCfg(astraCfg, flags.AstraCfgOutput)
	if err != nil {
		log.Panic(err)
	}
}
