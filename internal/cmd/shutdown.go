package cmd

import (
	"github.com/chzyer/readline"
	"github.com/machbase/neo-shell/client"
)

func init() {
	client.RegisterCmd(&client.Cmd{
		Name:   "shutdown",
		PcFunc: pcShutdown,
		Action: doShutdown,
		Desc:   "Shutdown server",
		Usage:  helpShutdown,
	})
}

const helpShutdown string = `  shutdown`

type ShutdownCmd struct {
	Interactive bool `kong:"-"`
	Help        bool `kong:"-"`
}

func pcShutdown() readline.PrefixCompleterInterface {
	return readline.PcItem("shutdown")
}

func doShutdown(ctx *client.ActionContext) {
	f := ctx.ShutdownServerFunc()
	if f == nil {
		ctx.Println("ERR", "server shutdown is not allowed")
	} else {
		err := f()
		if err != nil {
			ctx.Println("ERR", err.Error())
			return
		}
		ctx.Println("server shutting down...")
	}
}
