package cmd

import (
	"fmt"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/machbase/neo-shell/client"
	"github.com/machbase/neo-shell/do"
	"github.com/machbase/neo-shell/util"
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
      --all,-a       includes all hidden tables
    meta-tables      show meta tables
    virtual-tables   show virtual tables
`

type ShowCmd struct {
	Info   struct{} `cmd:""`
	Tables struct {
		ShowAll bool `name:"all" short:"a"`
	} `cmd:""`
	MetaTables    struct{} `cmd:""`
	VirtualTables struct{} `cmd:""`
	Help          bool     `kong:"-"`
}

func pcShow() readline.PrefixCompleterInterface {
	return readline.PcItem("show",
		readline.PcItem("info"),
		readline.PcItem("tables"),
		readline.PcItem("meta-tables"),
		readline.PcItem("virtual-tables"),
	)
}

func doShow(ctx *client.ActionContext) {
	cmd := &ShowCmd{}

	parser, err := client.Kong(cmd, func() error { ctx.Println(helpShow); cmd.Help = true; return nil })
	if err != nil {
		ctx.Println("ERR", err.Error())
		return
	}
	parserCtx, err := parser.Parse(util.SplitFields(ctx.Line, false))
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
		doShowTables(ctx, "M$SYS_TABLES", cmd.Tables.ShowAll)
	case "meta-tables":
		doShowTables(ctx, "M$TABLES", false)
	case "virtual-tables":
		doShowTables(ctx, "V$TABLES", false)
	default:
		ctx.Println(helpShow)
		return
	}
}

func doShowTables(ctx *client.ActionContext, tablesTable string, showAll bool) {
	rows, err := ctx.DB.Query(fmt.Sprintf("select NAME, TYPE, FLAG, ID from %s order by ID", tablesTable))
	if err != nil {
		ctx.Printfln("ERR select %s fail; %s", tablesTable, err.Error())
		return
	}
	defer rows.Close()

	t := ctx.NewBox([]string{"ROWNUM", "ID", "NAME", "TYPE"})

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

	box.AppendRow("mem.sys", util.BytesUnit(nfo.Runtime.MemSys, ctx.Lang))
	box.AppendRow("mem.heap.sys", util.BytesUnit(nfo.Runtime.MemHeapSys, ctx.Lang))
	box.AppendRow("mem.heap.alloc", util.BytesUnit(nfo.Runtime.MemHeapAlloc, ctx.Lang))
	box.AppendRow("mem.heap.in-use", util.BytesUnit(nfo.Runtime.MemHeapInUse, ctx.Lang))
	box.AppendRow("mem.stack.sys", util.BytesUnit(nfo.Runtime.MemStackSys, ctx.Lang))
	box.AppendRow("mem.stack.in-use", util.BytesUnit(nfo.Runtime.MemStackInUse, ctx.Lang))

	box.Render()
}
