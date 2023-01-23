package shell

import (
	"fmt"
	"strings"

	"github.com/chzyer/readline"
	"github.com/jedib0t/go-pretty/v6/table"
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

	cli.Println("TABLE   ", tableName)
	cli.Println("TYPE    ", tableTypeDesc(tableType, tableFlag))
	cli.Println("COLUMNS ", colCount)
	if tableType == 6 {
		tags := []string{}
		rows, err := cli.db.Query(fmt.Sprintf("select name from _%s_META", strings.ToUpper(object)))
		if err != nil {
			cli.Println("ERR", err.Error())
			return
		}
		defer rows.Close()
		for rows.Next() {
			var name string
			if err := rows.Scan(&name); err != nil {
				cli.Println("ERR", err.Error())
				return
			}
			tags = append(tags, name)
		}
		cli.Println("TAGS    ", strings.Join(tags, ", "))
	}
	rows, err := cli.db.Query("select name, type, length from M$SYS_COLUMNS where table_id = ? order by id", tableId)
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	defer rows.Close()

	t := table.NewWriter()
	t.SetOutputMirror(cli.conf.Stdout)
	t.SetStyle(table.StyleLight)
	t.AppendHeader(table.Row{"#", "NAME", "TYPE", "LENGTH"})

	nrow := 0
	for rows.Next() {
		var nam string
		var typ int
		var len int

		err = rows.Scan(&nam, &typ, &len)
		if err != nil {
			cli.Println("ERR", err.Error())
			return
		}
		nrow++
		t.AppendRow([]any{nrow, nam, typ, len})
	}

	if cli.conf.Format == Formats.CSV {
		t.RenderCSV()
	} else {
		t.Render()
	}
}
