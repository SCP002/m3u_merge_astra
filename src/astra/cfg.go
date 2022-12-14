package astra

import (
	"fmt"
	"io"
	"os"

	"m3u_merge_astra/cli"
	"m3u_merge_astra/util/copier"
	"m3u_merge_astra/util/slice"

	"github.com/SCP002/clipboard"
	json "github.com/SCP002/jsonexraw"
	"github.com/cockroachdb/errors"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/samber/lo"
)

// Cfg represents astra config
type Cfg struct {
	Categories []Category     `json:"categories"`
	Streams    []Stream       `json:"make_stream"`
	Unknown    map[string]any `json:"-" jsonex:"true"` // All unknown fields go here.
}

// Category represents category for groups of astra streams
type Category struct {
	Name   string  `json:"name"`
	Groups []Group `json:"groups"`
}

// Group represents group of astra streams
type Group struct {
	Name string `json:"name"`
}

// AddNewGroups returns copy of categories <cats> with new categories and groups from <streams>
func (r repo) AddNewGroups(cats []Category, streams []Stream) []Category {
	r.log.Info("Adding new groups\n")
	r.tw.AppendHeader(table.Row{"Category", "Group"})

	cats = copier.PDeep(cats)

	// Transform []Stream into []Category
	sCats := lo.FlatMap(streams, func(s Stream, _ int) []Category {
		return lo.MapToSlice(s.Groups, func(catName string, groupName any) Category {
			return Category{Name: catName, Groups: lo.WithoutEmpty([]Group{
				{Name: groupName.(string)},
			})}
		})
	})

	// Update input categories with categories from []Streams
	for _, sCat := range sCats {
		var idx int
		cats, _, idx = slice.FindIndexOrElse(cats, Category{Name: sCat.Name}, func(c Category) bool {
			return c.Name == sCat.Name
		})
		cats[idx].Groups = slice.AppendNew(cats[idx].Groups, func(g Group) {
			r.tw.AppendRow(table.Row{sCat.Name, g.Name})
		}, sCat.Groups...)
	}

	r.tw.Render()
	fmt.Fprint(os.Stderr, "\n")
	return cats
}

// ReadCfg returns serialized astra config from <source>.
//
// <source> can be 'clipboard', 'stdio' or file path.
func ReadCfg(source string) (Cfg, error) {
	var cfgRaw []byte
	var cfg Cfg
	var err error

	switch source {
	case string(cli.Clipboard):
		cfgRawStr, err := clipboard.ReadAll()
		if err != nil {
			return cfg, errors.Wrap(err, "read astra config from clipboard")
		}
		cfgRaw = []byte(cfgRawStr)
	case string(cli.Stdio):
		if cfgRaw, err = io.ReadAll(os.Stdin); err != nil {
			return cfg, errors.Wrap(err, "read astra config from stdin")
		}
	default:
		if cfgRaw, err = os.ReadFile(source); err != nil {
			return cfg, errors.Wrap(err, "read astra config from file")
		}
	}

	if err = json.Unmarshal([]byte(cfgRaw), &cfg); err != nil {
		return cfg, errors.Wrap(err, "serialize astra config")
	}

	return cfg, err
}

// WriteCfg writes <cfg> to <dest>.
//
// <dest> can be 'clipboard', 'stdio' or file path.
func WriteCfg(cfg Cfg, dest string) error {
	cfgRaw, err := json.MarshalIndent(cfg, "", "    ")
	if err != nil {
		return errors.Wrap(err, "deserialize astra config")
	}

	switch dest {
	case string(cli.Clipboard):
		if err = clipboard.WriteAll(string(cfgRaw)); err != nil {
			return errors.Wrap(err, "write astra config to clipboard")
		}
	case string(cli.Stdio):
		if _, err := os.Stdout.Write(cfgRaw); err != nil {
			return errors.Wrap(err, "write astra config to stdout")
		}
	default:
		if err := os.WriteFile(dest, cfgRaw, 0644); err != nil {
			return errors.Wrap(err, "write astra config to file")
		}
	}

	return nil
}
