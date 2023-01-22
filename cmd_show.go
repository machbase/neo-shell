package shell

import (
	"fmt"
	"time"

	"github.com/chzyer/readline"
	"github.com/machbase/cemlib/util"
)

func (cli *client) pcShow() *readline.PrefixCompleter {
	return readline.PcItem("show",
		readline.PcItem("tables"),
		readline.PcItem("info"),
		readline.PcItem("table",
			readline.PcItemDynamic(cli.listTables()),
		),
	)
}

func (cli *client) doShow(args []string) {
	switch args[0] {
	case "info":
		cli.doShowInfo()
	case "tables":
		cli.doShowTables()
	case "table":
		if len(args) == 2 {
			cli.doShowTable(args[1])
		} else {
			cli.Println("Usage: show table <table_name>")
		}
	default:
		cli.Printfln("unknown show '%s'", args[0])
	}
}

func (cli *client) doShowTable(table string) {
	cli.doWalk(fmt.Sprintf("select * from %s", table))
}

func (cli *client) doShowTables() {
	rows, err := cli.db.Query("select NAME, TYPE, FLAG from M$SYS_TABLES order by NAME")
	if err != nil {
		cli.Printfln("ERR select m$sys_tables fail; %s", err.Error())
		return
	}
	defer rows.Close()
	for rows.Next() {
		var name string
		var typ int
		var flg int
		rows.Scan(&name, &typ, &flg)
		desc := tableTypeDesc(typ, flg)
		cli.Printfln("%-24s %s", name, desc)
	}
}

func (cli *client) doShowInfo() {
	nfo, err := cli.db.GetServerInfo()
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	width := 18

	cli.Printfln("%-*s v%d.%d.%d #%s", width, "Server", nfo.Version.Major, nfo.Version.Minor, nfo.Version.Patch, nfo.Version.GitSHA)
	cli.Printfln("%-*s %s", width, "Engine", nfo.Version.Engine)

	cli.Printfln("%-*s %s %s", width, "os", nfo.Runtime.OS, nfo.Runtime.Arch)
	cli.Printfln("%-*s %d", width, "processes", nfo.Runtime.Processes)
	cli.Printfln("%-*s %d", width, "pid", nfo.Runtime.Pid)
	cli.Printfln("%-*s %s", width, "uptime", util.HumanizeDuration(time.Duration(nfo.Runtime.UptimeInSecond*int64(time.Second))))
	cli.Printfln("%-*s %d", width, "goroutines", nfo.Runtime.Goroutines)
	// total bytes of memory obtained from the OS
	// Sys measures the virtual address space reserved
	// by the Go runtime for the heap, stacks, and other internal data structures.
	cli.Printfln("%-*s %d MB", width, "mem sys", nfo.Runtime.MemSys/1024/1024)
	cli.Printfln("%-*s %d MB", width, "mem heap sys", nfo.Runtime.MemHeapSys/1024/1024)
	cli.Printfln("%-*s %d MB", width, "mem heap alloc", nfo.Runtime.MemHeapAlloc/1024/1024)
	cli.Printfln("%-*s %d MB", width, "mem heap in-use", nfo.Runtime.MemHeapInUse/1024/1024)
	cli.Printfln("%-*s %d KB", width, "mem stack sys", nfo.Runtime.MemStackSys/1024)
	cli.Printfln("%-*s %d KB", width, "mem stack in-use", nfo.Runtime.MemStackInUse/1024)
}
