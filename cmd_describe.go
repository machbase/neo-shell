package shell

import (
	"fmt"
	"strings"

	"github.com/chzyer/readline"
	"github.com/machbase/neo-grpc/machrpc"
)

func init() {
	RegisterCmd(&Cmd{
		Name:   "desc",
		PcFunc: pcDescribe,
		Action: doDescribe,
		Desc:   "desc <table>",
	})
}

func pcDescribe(c Client) readline.PrefixCompleterInterface {
	cli := c.(*client)
	return readline.PcItem("desc",
		readline.PcItemDynamic(cli.listTables()),
	)
}

func doDescribe(cli Client, line string) {
	object := line
	if len(line) == 0 {
		cli.Println("Usage: desc <table_name>")
		return
	}

	db := cli.Database()

	_desc, err := db.Describe(line)
	if err != nil {
		cli.Println("unable to describe", object, err.Error())
		return
	}
	desc := _desc.(*machrpc.TableDescription)

	cli.Println("TABLE   ", desc.Name)
	cli.Println("TYPE    ", desc.TypeString())
	if desc.Type == machrpc.TagTableType {
		tags := []string{}
		rows, err := db.Query(fmt.Sprintf("select name from _%s_META order by name", strings.ToUpper(object)))
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

	box := cli.NewBox([]string{"#", "NAME", "TYPE", "LENGTH"})
	for i, col := range desc.Columns {
		colType := machrpc.ColumnTypeDescription(col.Type)
		box.AppendRow(i+1, col.Name, colType, col.Length)
	}

	box.Render()
}
