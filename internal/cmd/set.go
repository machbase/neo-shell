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
		Usage: `  set <key>       <value>
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
	// onoff := func(t bool) string {
	// 	if t {
	// 		return "on"
	// 	} else {
	// 		return "off"
	// 	}
	// }
	// parseflag := func() bool {
	// 	b := "-"
	// 	if len(args) == 2 {
	// 		b = strings.ToLower(args[1])
	// 	}
	// 	var flag bool
	// 	if b == "on" {
	// 		flag = true
	// 	} else if b == "off" {
	// 		flag = false
	// 	}
	// 	ctx.Println(args[0], onoff(flag))
	// 	return flag
	// }
	// parseBoxStyle := func(s string) string {
	// 	switch s {
	// 	case "simple", "bold", "double", "light", "round":
	// 	default:
	// 		s = "light"
	// 	}
	// 	ctx.Println(args[0], s)
	// 	return s
	// }

	pref := ctx.Pref()
	if len(args) == 0 {
		box := ctx.NewBox([]string{"NAME", "VALUE", "DESCRIPTION"})
		itms := pref.Items()
		for _, itm := range itms {
			box.AppendRow(itm.Name, itm.Value(), itm.Description())
		}
		box.Render()
		return
	}

	if len(args) == 2 {
		itm := pref.Item(strings.ToLower(args[0]))
		if itm == nil {
			ctx.Println("unknown set key '%s'", args[0])
		} else {
			if err := itm.SetValue(args[1]); err != nil {
				ctx.Println("ERR", err.Error())
			} else {
				pref.Save()
			}
		}
	}
}
