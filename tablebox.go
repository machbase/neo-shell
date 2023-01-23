package shell

import "github.com/jedib0t/go-pretty/v6/table"

type Box interface {
	AppendRow(row []any)
	ResetRows()
	ResetHeaders()
	Render() string
}

func (cli *client) NewBox(header []any, compact bool) Box {
	b := &box{
		w:      table.NewWriter(),
		format: cli.conf.Format,
	}
	b.w.SetOutputMirror(cli.conf.Stdout)

	style := table.StyleDefault
	switch cli.conf.BoxStyle {
	case "bold":
		style = table.StyleBold
	case "double":
		style = table.StyleDouble
	case "light":
		style = table.StyleLight
	case "round":
		style = table.StyleRounded
	}
	if compact {
		style.Options.SeparateColumns = false
		style.Options.DrawBorder = false
	} else {
		style.Options.SeparateColumns = true
		style.Options.DrawBorder = true
	}
	b.w.SetStyle(style)

	if cli.conf.Heading {
		b.w.AppendHeader(table.Row(header))
	}
	return b
}

type box struct {
	w      table.Writer
	format string
}

func (b *box) AppendRow(row []any) {
	b.w.AppendRow(row)
}

func (b *box) ResetRows() {
	b.w.ResetRows()
}

func (b *box) ResetHeaders() {
	b.w.ResetHeaders()
}

func (b *box) Render() string {
	if b.format == Formats.CSV {
		return b.w.RenderCSV()
	} else {
		return b.w.Render()
	}
}
