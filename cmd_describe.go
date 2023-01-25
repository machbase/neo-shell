package shell

import (
	"fmt"
	"strings"

	"github.com/chzyer/readline"
)

func init() {
	RegisterCmd(&Cmd{
		Name:    "desc",
		Aliases: []string{},
		PcFunc:  pcDescribe,
		Action:  doDescribe,
		Desc:    "desc <table>",
	})
}

func pcDescribe(c Client) readline.PrefixCompleterInterface {
	cli := c.(*client)
	return readline.PcItem("desc",
		readline.PcItemDynamic(cli.listTables()),
	)
}

func doDescribe(c Client, line string) {
	object := line
	cli := c.(*client)
	if len(line) == 0 {
		cli.Println("Usage: desc <table_name>")
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
	if tableType == 6 {
		tags := []string{}
		rows, err := cli.db.Query(fmt.Sprintf("select name from _%s_META order by name", strings.ToUpper(object)))
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
	cli.Println("COLUMNS ", colCount)

	rows, err := cli.db.Query("select name, type, length from M$SYS_COLUMNS where table_id = ? order by id", tableId)
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	defer rows.Close()

	box := cli.NewBox([]any{"#", "NAME", "TYPE", "LENGTH"}, false)
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
		box.AppendRow(nrow, nam, typ, len)
	}

	box.Render()
}

func tableTypeDesc(typ int, flg int) string {
	desc := "undef"
	switch typ {
	case 0:
		desc = "Log Table"
	case 1:
		desc = "Fixed Table"
	case 3:
		desc = "Volatile Table"
	case 4:
		desc = "Lookup Table"
	case 5:
		desc = "KeyValue Table"
	case 6:
		desc = "Tag Table"
	}
	switch flg {
	case 1:
		desc += " (data)"
	case 2:
		desc += " (rollup)"
	case 4:
		desc += " (meta)"
	case 8:
		desc += " (stat)"
	}
	return desc
}
