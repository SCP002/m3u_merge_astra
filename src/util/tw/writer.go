package tw

import (
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
)

// Writer represents table writer
type Writer struct {
	table.Writer
}

// New returns new configured table writer
func New() Writer {
	tw := table.NewWriter()
	tw.SetOutputMirror(os.Stderr)
	tw.SetStyle(table.StyleLight)
	tw.SetColumnConfigs([]table.ColumnConfig{
		{Name: "From name", WidthMax: 30},
		{Name: "Category", WidthMax: 30},
		{Name: "Group", WidthMax: 30},
		{Name: "Hash", WidthMax: 30},
		{Name: "Input", WidthMax: 70},
		{Name: "Name", WidthMax: 30},
		{Name: "New group", WidthMax: 30},
		{Name: "New name", WidthMax: 30},
		{Name: "New URL", WidthMax: 70},
		{Name: "Note", WidthMax: 30},
		{Name: "Old name", WidthMax: 30},
		{Name: "Old URL", WidthMax: 70},
		{Name: "Original group", WidthMax: 30},
		{Name: "Reason", WidthMax: 40},
		{Name: "Result", WidthMax: 30},
		{Name: "To name", WidthMax: 30},
		{Name: "URL", WidthMax: 70},
	})

	return Writer{tw}
}

// Render renders table and resets it
func (w Writer) Render() {
	w.Writer.Render()
	w.ResetHeaders()
	w.ResetRows()
	w.ResetFooters()
}
