package cmd

import (
	"strings"

	"github.com/chzyer/readline"
	"github.com/machbase/neo-shell/client"
	"github.com/machbase/neo-shell/util"
)

func init() {
	client.RegisterCmd(&client.Cmd{
		Name:   "set",
		PcFunc: pcSet,
		Action: doSet,
		Desc:   "show/set shell settings",
		Usage: `  set <key> <value>
  set vi-mode     [on|off]
  set box-style   [simple|bold|double|light|round]
`,
		ClientAction: true,
	})
}

func pcSet() readline.PrefixCompleterInterface {
	return readline.PcItem("set",
		// readline.PcItem("tz",
		// 	readline.PcItem("UTC"),
		// 	readline.PcItem("Local"),
		// ),
		readline.PcItem("box-style",
			readline.PcItem("simple"),
			readline.PcItem("bold"),
			readline.PcItem("double"),
			readline.PcItem("light"),
			readline.PcItem("round"),
		),
		readline.PcItem("vi-mode",
			readline.PcItem("on"),
			readline.PcItem("off"),
		),
		// readline.PcItem("heading",
		// 	readline.PcItem("on"),
		// 	readline.PcItem("off"),
		// ),
		// readline.PcItem("format",
		// 	readline.PcItem(Formats.Default),
		// 	readline.PcItem(Formats.CSV),
		// 	readline.PcItem(Formats.JSON),
		// ),
	)
}

func doSet(ctx *client.ActionContext) {
	args := util.SplitFields(ctx.Line, true)
	onoff := func(t bool) string {
		if t {
			return "on"
		} else {
			return "off"
		}
	}
	parseflag := func(flag *bool) {
		b := "-"
		if len(args) == 2 {
			b = strings.ToLower(args[1])
		}
		if b == "on" {
			*flag = true
		} else if b == "off" {
			*flag = false
		}
		ctx.Println(args[0], onoff(*flag))
	}

	conf := ctx.Config()
	if len(args) == 0 {
		box := ctx.NewBox([]string{"NAME", "VALUE"})
		box.AppendRow("vi-mode", onoff(conf.VimMode))
		box.AppendRow("box-style", conf.BoxStyle)
		box.Render()
		return
	}
	switch strings.ToLower(args[0]) {
	case "vi-mode":
		parseflag(&conf.VimMode)
	case "box-style":
		conf.BoxStyle = parseBoxStyle(args[1])
		ctx.Println("box-style", conf.BoxStyle)
	}
}

func parseBoxStyle(s string) string {
	switch s {
	case "simple", "bold", "double", "light", "round":
		return s
	default:
		return "light"
	}
}
