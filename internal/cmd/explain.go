package cmd

import (
	"github.com/chzyer/readline"
	"github.com/machbase/neo-shell/client"
)

func init() {
	client.RegisterCmd(&client.Cmd{
		Name:   "explain",
		PcFunc: pcExplain,
		Action: doExplain,
		Desc:   "Display execution plan of query",
		Usage: `  explain <query>
  arguments:
    query       query statement to display the execution plan
`,
	})
}

func pcExplain() readline.PrefixCompleterInterface {
	return readline.PcItem("explain")
}

func doExplain(ctx *client.ActionContext) {
	plan, err := ctx.DB.Explain(ctx.Line)
	if err != nil {
		ctx.Println(err.Error())
		return
	}
	ctx.Println(plan)
}
