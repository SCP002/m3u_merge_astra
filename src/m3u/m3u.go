package m3u

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"regexp"
	"strings"

	"m3u_merge_astra/cfg"
	"m3u_merge_astra/deps"
	"m3u_merge_astra/util/conv"
	"m3u_merge_astra/util/slice"
	"m3u_merge_astra/util/tw"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/samber/lo"
	"github.com/sirupsen/logrus"
)

// repo represents dependencies holder for this package
type repo struct {
	log *logrus.Logger
	tw  tw.Writer
	cfg cfg.Root
}

// NewRepo returns new dependencies holder for this package
func NewRepo(log *logrus.Logger, tw tw.Writer, cfg cfg.Root) repo {
	return repo{log: log, tw: tw, cfg: cfg}
}

// Log used to satisfy deps.Global interface
func (r repo) Log() *logrus.Logger {
	return r.log
}

// TW used to satisfy deps.Global interface
func (r repo) TW() tw.Writer {
	return r.tw
}

// Cfg used to satisfy deps.Global interface
func (r repo) Cfg() cfg.Root {
	return r.cfg
}

// Channel represents M3U channel object
type Channel struct {
	Name  string
	Group string
	URL   string
}

// GetName used to satisfy util/slice.Named interface
func (ch Channel) GetName() string {
	return ch.Name
}

// replaceGroup returns new channel with group taken from config in <r>
func (ch Channel) replaceGroup(r deps.Global) Channel {
	newGroup := r.Cfg().M3U.ChannGroupMap[ch.Group]
	if ch.Group != newGroup && newGroup != "" {
		r.TW().AppendRow(table.Row{ch.Name, ch.Group, newGroup})
		ch.Group = newGroup
	}
	return ch
}

// Parse parses raw M3U channels into []Channel
func (r repo) Parse(rawChannels io.ReadCloser) (out []Channel) {
	r.log.Info("Parsing M3U channels\n")

	nameRx := regexp.MustCompile(`^#EXTINF:.*?,(.*)`)
	groupTitleRx := regexp.MustCompile(`^#EXTINF:.*group-title="(.*?)"`)
	extGrpRx := regexp.MustCompile(`^#EXTGRP:(.*)`)

	name := ""
	groupTitle := ""
	lastExtGrp := ""

	scanner := bufio.NewScanner(rawChannels)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())

		if line == "" {
			continue
		}
		if matchList := nameRx.FindStringSubmatch(line); len(matchList) > 1 {
			name = strings.TrimSpace(matchList[1])
		}
		if matchList := groupTitleRx.FindStringSubmatch(line); len(matchList) > 1 {
			groupTitle = strings.TrimSpace(matchList[1])
		}
		if matchList := extGrpRx.FindStringSubmatch(line); len(matchList) > 1 {
			lastExtGrp = strings.TrimSpace(matchList[1])
			continue
		}
		if !strings.HasPrefix(line, "#") && name != "" {
			ch := Channel{
				Name: name,
				// group-title have a priority over #EXTGRP
				Group: lo.Ternary(groupTitle != "", groupTitle, lastExtGrp),
				URL:   line,
			}
			out = append(out, ch)
			name = ""
			groupTitle = ""
			// #EXTGRP applies to every subsequent channel until overriden. Not clearing lastExtGrp.
		}
	}

	return
}

// ReplaceGroups returns copy of <channels> with groups taken from map in config
func (r repo) ReplaceGroups(channels []Channel) (out []Channel) {
	r.log.Info("Replacing groups of M3U channels\n")
	r.tw.AppendHeader(table.Row{"Name", "Original group", "New group"})

	for _, ch := range channels {
		out = append(out, ch.replaceGroup(r))
	}

	r.tw.Render()
	fmt.Fprint(os.Stderr, "\n")
	return
}

// RemoveBlocked returns copy of <channels> without blocked ones
func (r repo) RemoveBlocked(channels []Channel) (out []Channel) {
	r.log.Info("Removing blocked channels\n")
	r.tw.AppendHeader(table.Row{"Name", "Group", "URL"})

	out = lo.Reject(channels, func(ch Channel, _ int) bool {
		reject := slice.RxAnyMatch(r.cfg.M3U.ChannNameBlacklist, ch.Name) ||
			slice.RxAnyMatch(r.cfg.M3U.ChannGroupBlacklist, ch.Group) ||
			slice.RxAnyMatch(r.cfg.M3U.ChannURLBlacklist, ch.URL)
		if reject {
			r.tw.AppendRow(table.Row{ch.Name, ch.Group, ch.URL})
		}
		return reject
	})

	r.tw.Render()
	fmt.Fprint(os.Stderr, "\n")
	return
}

// HasURL returns true if <channels> contain <url>.
//
// If <withHash> is false, return true even if channel url and <url> are the same but have different hashes.
func (r repo) HasURL(channels []Channel, url string, withHash bool) bool {
	return lo.ContainsBy(channels, func(ch Channel) bool {
		equal, err := conv.LinksEqual(ch.URL, url, withHash)
		if err != nil {
			r.log.Debug(err)
		}
		return equal
	})
}
