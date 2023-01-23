package shell

import "github.com/jedib0t/go-pretty/v6/table"

type Box table.Writer

func (cli *client) NewBox(header []any) Box {
	box := table.NewWriter()
	box.SetOutputMirror(cli.conf.Stdout)
	switch cli.conf.BoxStyle {
	default:
		box.SetStyle(table.StyleDefault)
	case "bold":
		box.SetStyle(table.StyleBold)
	case "double":
		box.SetStyle(table.StyleDouble)
	case "light":
		box.SetStyle(table.StyleLight)
	case "round":
		box.SetStyle(table.StyleRounded)
	}
	if cli.conf.Heading {
		box.AppendHeader(table.Row(header))
	}
	return box
}
