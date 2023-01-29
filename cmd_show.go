package shell

import (
	"fmt"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/machbase/cemlib/util"
	"github.com/machbase/neo-grpc/machrpc"
)

func init() {
	RegisterCmd(&Cmd{
		Name:   "show",
		PcFunc: pcShow,
		Action: doShow,
		Desc:   "Display information",
		Usage:  helpShow,
	})
}

const helpShow = `  show [options] <command>
  commands:
    info             show server info
    tables           list tables
      --all,-a       includes all hidden tables`

type ShowCmd struct {
	Info   struct{} `cmd:""`
	Tables struct {
		ShowAll bool `name:"all" short:"a"`
	} `cmd:""`
	Help bool `kong:"-"`
}

func pcShow(c Client) readline.PrefixCompleterInterface {
	return readline.PcItem("show",
		readline.PcItem("tables"),
		readline.PcItem("info"),
	)
}

func doShow(cc Client, line string) {
	cmd := &ShowCmd{}

	parser, err := Kong(cmd, func() error { cc.Println(helpShow); cmd.Help = true; return nil })
	if err != nil {
		cc.Println("ERR", err.Error())
		return
	}
	ctx, err := parser.Parse(splitFields(line, false))
	if cmd.Help {
		return
	}
	if err != nil {
		cc.Println("ERR", err.Error())
		return
	}

	switch ctx.Command() {
	case "info":
		cli := cc.(*client)
		cli.doShowInfo()
	case "tables":
		cli := cc.(*client)
		cli.doShowTables(cmd.Tables.ShowAll)
	default:
		cc.Println(helpShow)
		return
	}
}

func (cli *client) doShowTables(showAll bool) {
	rows, err := cli.db.Query("select NAME, TYPE, FLAG, ID from M$SYS_TABLES order by ID")
	if err != nil {
		cli.Printfln("ERR select m$sys_tables fail; %s", err.Error())
		return
	}
	defer rows.Close()

	t := cli.NewBox([]string{"#", "ID", "NAME", "TYPE"})

	nrow := 0
	for rows.Next() {
		var name string
		var typ int
		var flg int
		var id int
		err := rows.Scan(&name, &typ, &flg, &id)
		if err != nil {
			cli.Println("ERR", err.Error())
			return
		}
		if !showAll && strings.HasPrefix(name, "_") {
			continue
		}
		nrow++

		desc := machrpc.TableTypeDescription(machrpc.TableType(typ), flg)
		t.AppendRow(nrow, id, name, desc)
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

	box := cli.NewBox([]string{"NAME", "VALUE"})

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
