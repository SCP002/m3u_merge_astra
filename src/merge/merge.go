package merge

import (
	"m3u_merge_astra/astra"
	"m3u_merge_astra/m3u"
	"m3u_merge_astra/util/rnd"
	"m3u_merge_astra/util/slice/find"

	"github.com/samber/lo"
)

// RenameStreams returns shallow copy of <streams> with names taken from <channels> if their standardized names are
// equal.
func (r repo) RenameStreams(streams []astra.Stream, channels []m3u.Channel) (out []astra.Stream) {
	r.log.Info("Renaming streams")

	for _, s := range streams {
		ch, _, chFound := find.Named(r.cfg.General, channels, s.Name)
		if chFound && s.Name != ch.Name {
			r.log.InfoFi("Renaming stream", "ID", s.ID, "old name", s.Name, "new name", ch.Name,
				"group", s.FirstGroup())
			s.Name = ch.Name
		}
		out = append(out, s)
	}

	return
}

// UpdateInputs returns shallow copy of <streams> with every first matching input of every stream replaced with matching
// URL's of m3u channels according to cfg.Streams.InputUpdateMap.
//
// If cfg.Streams.EnableOnInputUpdate is enabled in config, it also enables every stream on update.
func (r repo) UpdateInputs(streams []astra.Stream, channels []m3u.Channel) (out []astra.Stream) {
	r.log.Info("Updating inputs of streams")

	for _, s := range streams {
		find.EverySimilar(r.cfg.General, channels, s.Name, 0, func(ch m3u.Channel, _ int) {
			if !s.HasInput(r.log, ch.URL, true) {
				var updated bool
				s, updated = s.UpdateInput(r, ch.URL, func(oldURL string) {
					r.log.InfoFi("Updating input of stream", "ID", s.ID, "name", s.Name, "old URL", oldURL,
						"new URL", ch.URL, "note", s.InputsUpdateNote(r.cfg.Streams))
				})
				if r.cfg.Streams.EnableOnInputUpdate && updated && !s.Enabled {
					r.log.InfoFi("Enabling the stream (updating inputs of streams, enable_on_input_update is on)",
						"ID", s.ID, "name", s.Name)
					s = s.Enable()
				}
			}
		})
		out = append(out, s)
	}

	return
}

// RemoveInputsByUpdateMap returns shallow copy of <streams> without inputs which match at least one
// cfg.Streams.InputUpdateMap.From expression but none found in <channels>.
func (r repo) RemoveInputsByUpdateMap(streams []astra.Stream, channels []m3u.Channel) (out []astra.Stream) {
	r.log.Info("Removing absent inputs from streams according the update map")

	m3uRepo := m3u.NewRepo(r.log, r.cfg)

	for _, s := range streams {
		similarChannels := find.GetSimilar(r.cfg.General, channels, s.Name)
		for _, knownInp := range s.KnownInputs(r.cfg.Streams) {
			if !m3uRepo.HasURL(similarChannels, knownInp, false) {
				s = s.RemoveInputsCb(knownInp, func() {
					r.log.InfoFi("Removing absent input from stream according the update map", "ID", s.ID,
						"name", s.Name, "group", s.FirstGroup(), "input", knownInp)
				})
			}
		}
		out = append(out, s)
	}

	return
}

// AddNewInputs returns shallow copy of <streams> with new inputs if such found in <channels>.
//
// If cfg.Streams.EnableOnInputUpdate is enabled in config, it also enables every stream with new inputs.
func (r repo) AddNewInputs(streams []astra.Stream, channels []m3u.Channel) (out []astra.Stream) {
	r.log.Info("Adding new inputs to streams")

	for _, s := range streams {
		find.EverySimilar(r.cfg.General, channels, s.Name, 0, func(ch m3u.Channel, _ int) {
			if !s.HasInput(r.log, ch.URL, r.cfg.Streams.HashCheckOnAddNewInputs) {
				r.log.InfoFi("Adding new input to stream", "ID", s.ID, "name", s.Name, "group", s.FirstGroup(),
					"URL", ch.URL, "note", s.InputsUpdateNote(r.cfg.Streams))
				s = s.AddInput(ch.URL)
				if r.cfg.Streams.EnableOnInputUpdate && !s.Enabled {
					r.log.InfoFi("Enabling the stream (adding new inputs to streams, enable_on_input_update is on)",
						"ID", s.ID, "name", s.Name)
					s = s.Enable()
				}
			}
		})
		out = append(out, s)
	}

	return
}

// AddNewStreams returns <streams> with new streams generated from <channels> if no such found in <streams>
func (r repo) AddNewStreams(streams []astra.Stream, channels []m3u.Channel) []astra.Stream {
	r.log.Info("Adding new streams")

	astraRepo := astra.NewRepo(r.log, r.cfg)

	for _, ch := range channels {
		if !r.cfg.Streams.AddNewWithKnownInputs && astraRepo.HasInput(streams, ch.URL, false) {
			continue
		}
		if !find.HasAnySimilar(r.cfg.General, streams, ch.Name) {
			id := generateUID(streams)
			stream := astra.NewStream(r.cfg.Streams, id, ch.Name, ch.Group, []string{ch.URL})
			r.log.InfoFi("Adding new stream", "ID", id, "name", ch.Name, "group", stream.FirstGroup(), "input", ch.URL)
			streams = append(streams, stream)
		}
	}

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
