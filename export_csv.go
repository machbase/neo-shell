package shell

import (
	"encoding/csv"
	"io"
	"strconv"
	"unicode/utf8"

	"github.com/machbase/neo-grpc/machrpc"
	"golang.org/x/term"
)

func (cli *client) exportRowsCsv(writer io.Writer, rows *machrpc.Rows, cmd *SqlCmd) {
	cols, err := rows.Columns()
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}

	windowHeight := 0
	if cmd.Interactive && term.IsTerminal(0) {
		if _, height, err := term.GetSize(0); err == nil {
			windowHeight = height
		}
	}
	pageHeight := windowHeight - 1
	nextPauseRow := pageHeight
	if cmd.Heading {
		nextPauseRow--
	}

	csvWriter := csv.NewWriter(writer)
	csvWriter.Comma, _ = utf8.DecodeRuneInString(cmd.Delimiter)
	defer csvWriter.Flush()

	if cmd.Heading {
		names := cli.columnNames(cols, cmd.TimeLocation, cmd.Rownum)
		// TODO check return error, if csvWritter.Comma is not valid
		csvWriter.Write(names)
	}

	buf := makeBuffer(cols)
	nrow := 0
	for {
		if !rows.Next() {
			return
		}
		err := rows.Scan(buf...)
		if err != nil {
			cli.Println("ERR", err.Error())
			return
		}
		nrow++

		vs := makeCsvValues(buf, cmd.TimeLocation, cmd.TimeFormat, cmd.Precision)
		if cmd.Rownum {
			vs = append([]string{strconv.Itoa(nrow)}, vs...)
		}

		csvWriter.Write(vs)

		if nextPauseRow > 0 && nextPauseRow == nrow {
			nextPauseRow += pageHeight

			csvWriter.Flush()
			if !cli.pauseForMore() {
				return
			}
		}
	}
}
