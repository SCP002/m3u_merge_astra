package merge

import (
	"fmt"
	"os"

	"m3u_merge_astra/astra"
	"m3u_merge_astra/m3u"
	"m3u_merge_astra/util/rnd"
	"m3u_merge_astra/util/slice/find"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/samber/lo"
)

// RenameStreams returns copy of <streams> with names taken from <channels> if their standardized names are equal
func (r repo) RenameStreams(streams []astra.Stream, channels []m3u.Channel) (out []astra.Stream) {
	r.log.Info("Renaming streams\n")
	r.tw.AppendHeader(table.Row{"Old name", "New name", "Group"})

	for _, s := range streams {
		ch, _, chFound := find.Named(r.cfg.General, channels, s.Name)
		if chFound && s.Name != ch.Name {
			r.tw.AppendRow(table.Row{s.Name, ch.Name, s.FirstGroup()})
			s.Name = ch.Name
		}
		out = append(out, s)
	}

	r.tw.Render()
	fmt.Fprint(os.Stderr, "\n")
	return
}

// UpdateInputs returns copy of <streams> with every first matching input of every stream replaced with matching URL's
// of m3u channels according to cfg.Streams.InputUpdateMap.
//
// If cfg.Streams.EnableOnInputUpdate is enabled in config, it also enables every stream on update.
func (r repo) UpdateInputs(streams []astra.Stream, channels []m3u.Channel) (out []astra.Stream) {
	r.log.Info("Updating inputs\n")
	r.tw.AppendHeader(table.Row{"Name", "Old URL", "New URL", "Note"})

	for _, s := range streams {
		find.EverySimilar(r.cfg.General, channels, s.Name, 0, func(ch m3u.Channel, _ int) {
			if !s.HasInput(r, ch.URL, true) {
				s = s.UpdateInput(r, ch.URL, func(oldURL string) {
					r.tw.AppendRow(table.Row{s.Name, oldURL, ch.URL, s.InputsUpdateNote(r.cfg.Streams)})
				})
				if r.cfg.Streams.EnableOnInputUpdate {
					s = s.Enable()
				}
			}
		})
		out = append(out, s)
	}

	r.tw.Render()
	fmt.Fprint(os.Stderr, "\n")
	return
}

// RemoveInputsByUpdateMap returns copy of <streams> without inputs which match at least one
// cfg.Streams.InputUpdateMap.From expression but none found in <channels>.
func (r repo) RemoveInputsByUpdateMap(streams []astra.Stream, channels []m3u.Channel) (out []astra.Stream) {
	r.log.Info("Removing absent inputs according the update map\n")
	r.tw.AppendHeader(table.Row{"Name", "Group", "Input"})
	m3uRepo := m3u.NewRepo(r.log, r.tw, r.cfg)

	for _, s := range streams {
		similarChannels := find.GetSimilar(r.cfg.General, channels, s.Name)
		for _, knownInp := range s.KnownInputs(r.cfg.Streams) {
			if !m3uRepo.HasURL(similarChannels, knownInp, false) {
				s = s.RemoveInputsCb(knownInp, func() {
					r.tw.AppendRow(table.Row{s.Name, s.FirstGroup(), knownInp})
				})
			}
		}
		out = append(out, s)
	}

	r.tw.Render()
	fmt.Fprint(os.Stderr, "\n")
	return
}

// AddNewInputs returns copy of <streams> with new inputs if such found in <channels>.
//
// If cfg.Streams.EnableOnInputUpdate is enabled in config, it also enables every stream with new inputs.
func (r repo) AddNewInputs(streams []astra.Stream, channels []m3u.Channel) (out []astra.Stream) {
	r.log.Info("Adding new inputs\n")
	r.tw.AppendHeader(table.Row{"Name", "Group", "URL", "Note"})

	for _, s := range streams {
		find.EverySimilar(r.cfg.General, channels, s.Name, 0, func(ch m3u.Channel, _ int) {
			if !s.HasInput(r, ch.URL, r.cfg.Streams.HashCheckOnAddNewInputs) {
				r.tw.AppendRow(table.Row{s.Name, s.FirstGroup(), ch.URL, s.InputsUpdateNote(r.cfg.Streams)})
				s = s.AddInput(ch.URL)
				if r.cfg.Streams.EnableOnInputUpdate {
					s = s.Enable()
				}
			}
		})
		out = append(out, s)
	}

	r.tw.Render()
	fmt.Fprint(os.Stderr, "\n")
	return
}

// AddNewStreams returns copy of <streams> with new streams generated from <channels> if no such found in <streams>
func (r repo) AddNewStreams(streams []astra.Stream, channels []m3u.Channel) []astra.Stream {
	r.log.Info("Adding new streams\n")
	r.tw.AppendHeader(table.Row{"Name", "Group", "Input"})
	astraRepo := astra.NewRepo(r.log, r.tw, r.cfg)

	for _, ch := range channels {
		if !r.cfg.Streams.AddNewWithKnownInputs && astraRepo.HasInput(streams, ch.URL, false) {
			continue
		}
		if !find.HasAnySimilar(r.cfg.General, streams, ch.Name) {
			id := generateUID(streams)
			stream := astra.NewStream(r.cfg.Streams, id, ch.Name, ch.Group, []string{ch.URL})
			r.tw.AppendRow(table.Row{ch.Name, stream.FirstGroup(), ch.URL})
			streams = append(streams, stream)
		}
	}

	r.tw.Render()
	fmt.Fprint(os.Stderr, "\n")
	return streams
}

// generateUID returns 4 symbols long ID unique for <streams>
func generateUID(streams []astra.Stream) string {
	for {
		uid := rnd.String(4, false, true)
		duplicate := lo.ContainsBy(streams, func(s astra.Stream) bool {
			return s.ID == uid
		})
		if duplicate {
			continue
		}
		return uid
	}
}
