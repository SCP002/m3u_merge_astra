package tw

import (
	"os"

	"github.com/jedib0t/go-pretty/v6/table"
	"golang.org/x/term"
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
	return Writer{tw}
}

// AppendHeader appends the row to the List of headers to render.
//
// Only the first item in the "config" will be tagged against this row.
//
// Sets maximum width of every column evenly distributed across the terminal width.
func (w Writer) AppendHeader(row table.Row, config ...table.RowConfig) {
	// getTermWidth returns terminal width in characters or <fallback> on error.
	//
	// Always fails to get proper width in unit tests.
	getTermWidth := func(fallback int) int {
		width, _, err := term.GetSize(int(os.Stdout.Fd()))
		if err != nil {
			return fallback
		}
		return width
	}

	widthPerColumn := getTermWidth(120) / len(row) - 2 * 2

	columnConfigs := []table.ColumnConfig{}
	for colNum := 1; colNum <= len(row); colNum++ {
		columnConfigs = append(columnConfigs, table.ColumnConfig{
			Number:   colNum,
			WidthMax: widthPerColumn,
		})
	}
	w.SetColumnConfigs(columnConfigs)

	w.Writer.AppendHeader(row, config...)
}

// Render renders table and resets it
func (w Writer) Render() {
	w.Writer.Render()
	w.ResetHeaders()
	w.ResetRows()
	w.ResetFooters()
}
