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
		Desc:   "describe table structure",
		Usage:  helpDescribe,
	})
}

const helpDescribe = `  desc [options] <table>
  options:
    --all,-a     show all hidden columns
`

type DescribeCmd struct {
	Table   string `arg:"" name:"table"`
	ShowAll bool   `name:"all" short:"a"`
	Help    bool   `kong:"-"`
}

func pcDescribe(c Client) readline.PrefixCompleterInterface {
	cli := c.(*client)
	return readline.PcItem("desc",
		readline.PcItemDynamic(cli.listTables()),
	)
}

func doDescribe(cli Client, line string) {
	cmd := &DescribeCmd{}

	parser, err := Kong(cmd, func() error { cli.Println(helpDescribe); cmd.Help = true; return nil })
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	_, err = parser.Parse(splitFields(line, false))
	if cmd.Help {
		return
	}
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}

	db := cli.Database()

	_desc, err := db.Describe(cmd.Table)
	if err != nil {
		cli.Println("unable to describe", cmd.Table, "; ERR", err.Error())
		return
	}
	desc := _desc.(*machrpc.TableDescription)

	cli.Println("TABLE   ", desc.Name)
	cli.Println("TYPE    ", desc.TypeString())
	if desc.Type == machrpc.TagTableType {
		tags := []string{}
		rows, err := db.Query(fmt.Sprintf("select name from _%s_META order by name", strings.ToUpper(cmd.Table)))
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

	nrow := 0
	box := cli.NewBox([]string{"#", "ID", "NAME", "TYPE", "LENGTH"})
	for _, col := range desc.Columns {
		if !cmd.ShowAll && strings.HasPrefix(col.Name, "_") {
			continue
		}
		nrow++
		colType := machrpc.ColumnTypeDescription(col.Type)
		box.AppendRow(nrow, col.Id, col.Name, colType, col.Length)
	}

	box.Render()
}
