package shell

import (
	"io"

	"github.com/machbase/neo-grpc/machrpc"
	"golang.org/x/term"
)

func (cli *client) exportRowsNone(writer io.Writer, rows *machrpc.Rows, cmd *SqlCmd) {
	cols, err := rows.Columns()
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	buf := makeBuffer(cols)
	names := cli.columnNames(cols, cmd.Rownum)

	box := cli.newBox(names, !cmd.Interactive, writer)
	windowHeight := 0
	if term.IsTerminal(0) {
		if _, height, err := term.GetSize(0); err == nil {
			windowHeight = height
		}
	}

	pageHeight := windowHeight - 4
	if cli.conf.Heading {
		pageHeight--
	}

	nrow := 0
	for {
		if !rows.Next() {
			box.Render()
			box.ResetRows()
			return
		}
		err := rows.Scan(buf...)
		if err != nil {
			cli.Println("ERR", err.Error())
			return
		}
		nrow++
		vs := makeValues(buf, cli.TimeLocation(), cmd.Precision)
		values := make([]any, len(vs)+1)
		values[0] = nrow
		for i := range vs {
			values[i+1] = vs[i]
		}
		box.AppendRow(values...)

		if cmd.Interactive {
			if pageHeight > 0 && nrow%pageHeight == 0 {
				box.Render()
				box.ResetRows()
				if !cli.pauseForMore() {
					return
				}
			}
		} else {
			if nrow%1000 == 0 {
				box.Render()
				box.ResetRows()
				box.ResetHeaders()
			}
		}
	}
}
