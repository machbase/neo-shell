package shell

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/chzyer/readline"
	"github.com/gdamore/tcell/v2"
	"github.com/rivo/tview"
)

func init() {
	RegisterCmd(&Cmd{
		Name:    "describe",
		Aliases: []string{"desc"},
		PcFunc:  pcDescribe,
		Action:  doDescribe,
		Usage:   "describe <table_name>",
		Desc:    "display table structure",
	})
}

func pcDescribe(c Client) readline.PrefixCompleterInterface {
	cli := c.(*client)
	return readline.PcItem("describe",
		readline.PcItemDynamic(cli.listTables()),
	)
}

func doDescribe(c Client, line string, interactive bool) {
	object := line
	cli := c.(*client)
	if len(line) == 0 {
		cli.Println("Usage: describe <table_name>")
		return
	}

	var tableName string
	var tableType int
	var tableFlag int
	var tableId int
	var colCount int

	r := cli.db.QueryRow("select name, type, flag, id, colcount from M$SYS_TABLES where name = ?", strings.ToUpper(object))
	if err := r.Scan(&tableName, &tableType, &tableFlag, &tableId, &colCount); err != nil {
		cli.Println("unable to describe", object)
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
	table.SetDoneFunc(func(key tcell.Key) {
		if key == tcell.KeyESC {
			app.Stop()
		}
	})

	rows, err := cli.db.Query("select name, type, length from M$SYS_COLUMNS where table_id = ? order by id", tableId)
	if err != nil {
		cli.Println("ERR", err.Error())
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
			cli.Println("ERR", err.Error())
			return
		}
		nrow++

		table.SetCell(nrow, 0, tview.NewTableCell(colName))
		table.SetCell(nrow, 1, tview.NewTableCell(strconv.Itoa(colType)))
		table.SetCell(nrow, 2, tview.NewTableCell(strconv.Itoa(colLen)))
	}

	if err := app.SetRoot(table, true).SetFocus(table).Run(); err != nil {
		cli.Println("ERR", err.Error())
		return
	}
}
