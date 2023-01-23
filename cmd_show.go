package shell

import (
	"fmt"
	"time"

	"github.com/chzyer/readline"
	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/machbase/cemlib/util"
)

func init() {
	RegisterCmd(&Cmd{
		Name:    "show",
		Aliases: []string{},
		PcFunc:  pcShow,
		Action:  doShow,
		Desc:    "display information",
		Usage:   "show [tables | info | table <table_name>]",
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
			doShowTable(c, args[1], interactive)
		} else {
			cli.Println("Usage: show table <table_name>")
		}
	default:
		cli.Printfln("unknown show '%s'", args[0])
	}
}

func doShowTable(c Client, table string, interactive bool) {
	doWalk(c, fmt.Sprintf("select * from %s", table), interactive)
}

func (cli *client) doShowTables() {
	rows, err := cli.db.Query("select NAME, TYPE, FLAG from M$SYS_TABLES order by NAME")
	if err != nil {
		cli.Printfln("ERR select m$sys_tables fail; %s", err.Error())
		return
	}
	defer rows.Close()

	t := table.NewWriter()
	t.SetOutputMirror(cli.conf.Stdout)
	t.SetStyle(table.StyleLight)
	t.AppendHeader(table.Row{"#", "NAME", "TYPE", "DESC"})

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
		t.AppendRow([]any{nrow, name, typ, desc})
	}
	t.Render()
}

func (cli *client) doShowInfo() {
	nfo, err := cli.db.GetServerInfo()
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}

	t := table.NewWriter()
	t.SetOutputMirror(cli.conf.Stdout)
	t.SetStyle(table.StyleLight)
	if cli.conf.Heading {
		t.AppendHeader(table.Row{"NAME", "VALUE"})
	}

	t.AppendRow([]any{"build.version", fmt.Sprintf("v%d.%d.%d", nfo.Version.Major, nfo.Version.Minor, nfo.Version.Patch)})
	t.AppendRow([]any{"build.hash", fmt.Sprintf("#%s", nfo.Version.GitSHA)})
	t.AppendRow([]any{"build.timestamp", nfo.Version.BuildTimestamp})
	t.AppendRow([]any{"build.engine", nfo.Version.Engine})

	t.AppendRow([]any{"runtime.os", nfo.Runtime.OS})
	t.AppendRow([]any{"runtime.arch", nfo.Runtime.Arch})
	t.AppendRow([]any{"runtime.pid", nfo.Runtime.Pid})
	t.AppendRow([]any{"runtime.uptime", util.HumanizeDuration(time.Duration(nfo.Runtime.UptimeInSecond * int64(time.Second)))})
	t.AppendRow([]any{"runtime.goroutines", nfo.Runtime.Goroutines})

	t.AppendRow([]any{"mem.sys", cli.bytesUnit(nfo.Runtime.MemSys)})
	t.AppendRow([]any{"mem.heap.sys", cli.bytesUnit(nfo.Runtime.MemHeapSys)})
	t.AppendRow([]any{"mem.heap.alloc", cli.bytesUnit(nfo.Runtime.MemHeapAlloc)})
	t.AppendRow([]any{"mem.heap.in-use", cli.bytesUnit(nfo.Runtime.MemHeapInUse)})
	t.AppendRow([]any{"mem.stack.sys", cli.bytesUnit(nfo.Runtime.MemStackSys)})
	t.AppendRow([]any{"mem.stack.in-use", cli.bytesUnit(nfo.Runtime.MemStackInUse)})

	if cli.conf.Format == Formats.CSV {
		t.RenderCSV()
	} else {
		t.Render()
	}
}
