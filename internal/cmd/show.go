package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/machbase/cemlib/util"
	"github.com/machbase/neo-shell/client"
	"github.com/machbase/neo-shell/do"
	neoutil "github.com/machbase/neo-shell/util"
	spi "github.com/machbase/neo-spi"
)

func init() {
	client.RegisterCmd(&client.Cmd{
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

func pcShow() readline.PrefixCompleterInterface {
	return readline.PcItem("show",
		readline.PcItem("tables"),
		readline.PcItem("info"),
	)
}

func doShow(ctx *client.ActionContext) {
	cmd := &ShowCmd{}

	parser, err := client.Kong(cmd, func() error { ctx.Println(helpShow); cmd.Help = true; return nil })
	if err != nil {
		ctx.Println("ERR", err.Error())
		return
	}
	parserCtx, err := parser.Parse(neoutil.SplitFields(ctx.Line, false))
	if cmd.Help {
		return
	}
	if err != nil {
		ctx.Println("ERR", err.Error())
		return
	}

	switch parserCtx.Command() {
	case "info":
		doShowInfo(ctx)
	case "tables":
		doShowTables(ctx, cmd.Tables.ShowAll)
	default:
		ctx.Println(helpShow)
		return
	}
}

func doShowTables(ctx *client.ActionContext, showAll bool) {
	rows, err := ctx.DB.Query("select NAME, TYPE, FLAG, ID from M$SYS_TABLES order by ID")
	if err != nil {
		ctx.Printfln("ERR select m$sys_tables fail; %s", err.Error())
		return
	}
	defer rows.Close()

	t := ctx.NewBox([]string{"#", "ID", "NAME", "TYPE"})

	nrow := 0
	for rows.Next() {
		var name string
		var typ int
		var flg int
		var id int
		err := rows.Scan(&name, &typ, &flg, &id)
		if err != nil {
			ctx.Println("ERR", err.Error())
			return
		}
		if !showAll && strings.HasPrefix(name, "_") {
			continue
		}
		nrow++

		desc := do.TableTypeDescription(spi.TableType(typ), flg)
		t.AppendRow(nrow, id, name, desc)
	}
	t.Render()
}

func doShowInfo(ctx *client.ActionContext) {
	nfo, err := ctx.DB.GetServerInfo()
	if err != nil {
		ctx.Println("ERR", err.Error())
		return
	}

	uptime := time.Duration(nfo.Runtime.UptimeInSecond) * time.Second

	box := ctx.NewBox([]string{"NAME", "VALUE"})

	box.AppendRow("build.version", fmt.Sprintf("v%d.%d.%d", nfo.Version.Major, nfo.Version.Minor, nfo.Version.Patch))
	box.AppendRow("build.hash", fmt.Sprintf("#%s", nfo.Version.GitSHA))
	box.AppendRow("build.timestamp", nfo.Version.BuildTimestamp)
	box.AppendRow("build.engine", nfo.Version.Engine)

	box.AppendRow("runtime.os", nfo.Runtime.OS)
	box.AppendRow("runtime.arch", nfo.Runtime.Arch)
	box.AppendRow("runtime.pid", nfo.Runtime.Pid)
	box.AppendRow("runtime.uptime", util.HumanizeDurationWithFormat(uptime, util.HumanizeDurationFormatSimple))
	box.AppendRow("runtime.goroutines", nfo.Runtime.Goroutines)

	box.AppendRow("mem.sys", neoutil.BytesUnit(nfo.Runtime.MemSys, ctx.Lang))
	box.AppendRow("mem.heap.sys", neoutil.BytesUnit(nfo.Runtime.MemHeapSys, ctx.Lang))
	box.AppendRow("mem.heap.alloc", neoutil.BytesUnit(nfo.Runtime.MemHeapAlloc, ctx.Lang))
	box.AppendRow("mem.heap.in-use", neoutil.BytesUnit(nfo.Runtime.MemHeapInUse, ctx.Lang))
	box.AppendRow("mem.stack.sys", neoutil.BytesUnit(nfo.Runtime.MemStackSys, ctx.Lang))
	box.AppendRow("mem.stack.in-use", neoutil.BytesUnit(nfo.Runtime.MemStackInUse, ctx.Lang))

	box.Render()
}