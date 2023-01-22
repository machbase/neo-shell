package shell

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func (cli *client) pcDescribe() *readline.PrefixCompleter {
	return readline.PcItem("describe",
		readline.PcItemDynamic(cli.listTables()),
	)
}

func (cli *client) doDescribe(object string) {
	object = strings.TrimSpace(object)
	if len(object) == 0 {
		cli.Writeln("Usage: describe <table_name>")
		return
	}

	var tableName string
	var tableType int
	var tableFlag int
	var tableId int
	var colCount int

	r := cli.db.QueryRow("select name, type, flag, id, colcount from M$SYS_TABLES where name = ?", strings.ToUpper(object))
	if err := r.Scan(&tableName, &tableType, &tableFlag, &tableId, &colCount); err != nil {
		cli.Writeln("unable to describe", object)
		return
	}

	title := fmt.Sprintf(" %s (%s) - ESC to quit ", tableName, tableTypeDesc(tableType, tableFlag))
	labels := []string{"NAME", "TYPE", "LENGTH"}

	app := tview.NewApplication()
	table := tview.NewTable()
	table.SetBorder(true).SetTitle(title).SetTitleAlign(tview.AlignLeft)
	table.SetFixed(1, 1)
	for i, l := range labels {
		table.SetCell(0, i, tview.NewTableCell(l).SetTextColor(tcell.ColorYellow).SetAlign(tview.AlignCenter))
	}

	rows, err := cli.db.Query("select name, type, length from M$SYS_COLUMNS where table_id = ? order by id", tableId)
	if err != nil {
		cli.Writeln("ERR", err.Error())
		return
	}
	defer rows.Close()

	nrow := 0
	for rows.Next() {
		var colName string
		var colType int
		var colLen int
		err = rows.Scan(&colName, &colType, &colLen)
		if err != nil {
			cli.Writeln("ERR", err.Error())
			return
		}
		nrow++

		table.SetCell(nrow, 0, tview.NewTableCell(colName))
		table.SetCell(nrow, 1, tview.NewTableCell(strconv.Itoa(colType)))
		table.SetCell(nrow, 2, tview.NewTableCell(strconv.Itoa(colLen)))
	}

	table.SetDoneFunc(func(key tcell.Key) {
		app.Stop()
	})
	if err := app.SetRoot(table, true).SetFocus(table).Run(); err != nil {
		cli.Writeln("ERR", err.Error())
		return
	}
}
