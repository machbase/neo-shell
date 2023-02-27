//go:build neo_shell

package cmd

import (
	"github.com/chzyer/readline"
	"github.com/machbase/neo-shell/client"
	"github.com/machbase/neo-shell/server/psqlsvr"
	"github.com/machbase/neo-shell/util"
)

func init() {
	if !psqlsvr.Enabled() {
		return
	}

	client.RegisterCmd(&client.Cmd{
		Name:   "listen-psql",
		PcFunc: pcListenPsql,
		Action: doListenPsql,
		Desc:   "listen address for postgresql wire protocol",
		Usage:  helpListenPsql,
	})
}

const helpListenPsql = `  listen-psql <addr>
  arguments:
    addr               tcp or unix domain socket address to listen
`

type ListenPsqlCmd struct {
	Addr string `arg:"" name:"addr"`
	Help bool   `kong:"-"`
}

func pcListenPsql() readline.PrefixCompleterInterface {
	return readline.PcItem("listen")
}

func doListenPsql(ctx *client.ActionContext) {
	cmd := &ListenPsqlCmd{}
	parser, err := client.Kong(cmd, func() error { ctx.Println(helpListenPsql); cmd.Help = true; return nil })
	if err != nil {
		ctx.Println("ERR", err.Error())
		return
	}

	_, err = parser.Parse(util.SplitFields(ctx.Line, false))
	if cmd.Help {
		return
	}
	if err != nil {
		ctx.Println("ERR", err.Error())
		return
	}

	lsnr, err := psqlsvr.New(ctx.DB, &psqlsvr.Config{Address: cmd.Addr})
	if err != nil {
		ctx.Println("ERR", err.Error())
		return
	}
	ctx.Printfln("psql server listen %s", cmd.Addr)
	err = lsnr.Start()
	if err != nil {
		ctx.Println("ERR", err.Error())
	}
}
