package shell

import (
	"fmt"
	"time"

	"github.com/chzyer/readline"
	"github.com/machbase/cemlib/util"
)

func init() {
	RegisterCmd(&Cmd{
		Name:    "show",
		Aliases: []string{},
		PcFunc:  pcShow,
		Action:  doShow,
		Desc:    "display information",
		Usage: `  show tables          list tables
  show table <table>   equiv. 'walk SELECT * FROM <table>'
  show info            runtime info of server`,
	})
}

func pcShow(c Client) readline.PrefixCompleterInterface {
	cli := c.(*client)
	return readline.PcItem("show",
		readline.PcItem("tables"),
		readline.PcItem("info"),
		readline.PcItem("table",
			readline.PcItemDynamic(cli.listTables()),
		),
	)
}

func doShow(c Client, line string, interactive bool) {
	cli := c.(*client)
	args := splitFields(line)
	switch args[0] {
	case "info":
		cli.doShowInfo()
	case "tables":
		cli.doShowTables()
	case "table":
		if len(args) == 2 {
			doWalk(c, fmt.Sprintf("select * from %s", args[1]), interactive)
		} else {
			cli.Println("Usage: show table <table_name>")
		}
	default:
		cli.Printfln("unknown show '%s'", args[0])
	}
}

func (cli *client) doShowTables() {
	rows, err := cli.db.Query("select NAME, TYPE, FLAG from M$SYS_TABLES order by NAME")
	if err != nil {
		cli.Printfln("ERR select m$sys_tables fail; %s", err.Error())
		return
	}
	defer rows.Close()

	t := cli.NewBox([]any{"#", "NAME", "TYPE", "DESC"}, false)

	nrow := 0
	for rows.Next() {
		var name string
		var typ int
		var flg int
		err := rows.Scan(&name, &typ, &flg)
		if err != nil {
			cli.Println("ERR", err.Error())
			return
		}
		nrow++

		desc := tableTypeDesc(typ, flg)
		t.AppendRow(nrow, name, typ, desc)
	}
	t.Render()
}

func (cli *client) doShowInfo() {
	nfo, err := cli.db.GetServerInfo()
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}

	uptime := time.Duration(nfo.Runtime.UptimeInSecond) * time.Second

	box := cli.NewBox([]any{"NAME", "VALUE"}, false)

	box.AppendRow("build.version", fmt.Sprintf("v%d.%d.%d", nfo.Version.Major, nfo.Version.Minor, nfo.Version.Patch))
	box.AppendRow("build.hash", fmt.Sprintf("#%s", nfo.Version.GitSHA))
	box.AppendRow("build.timestamp", nfo.Version.BuildTimestamp)
	box.AppendRow("build.engine", nfo.Version.Engine)

	box.AppendRow("runtime.os", nfo.Runtime.OS)
	box.AppendRow("runtime.arch", nfo.Runtime.Arch)
	box.AppendRow("runtime.pid", nfo.Runtime.Pid)
	box.AppendRow("runtime.uptime", util.HumanizeDurationWithFormat(uptime, util.HumanizeDurationFormatSimple))
	box.AppendRow("runtime.goroutines", nfo.Runtime.Goroutines)

	box.AppendRow("mem.sys", cli.bytesUnit(nfo.Runtime.MemSys))
	box.AppendRow("mem.heap.sys", cli.bytesUnit(nfo.Runtime.MemHeapSys))
	box.AppendRow("mem.heap.alloc", cli.bytesUnit(nfo.Runtime.MemHeapAlloc))
	box.AppendRow("mem.heap.in-use", cli.bytesUnit(nfo.Runtime.MemHeapInUse))
	box.AppendRow("mem.stack.sys", cli.bytesUnit(nfo.Runtime.MemStackSys))
	box.AppendRow("mem.stack.in-use", cli.bytesUnit(nfo.Runtime.MemStackInUse))

	box.Render()
}
