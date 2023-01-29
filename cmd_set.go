package shell

import (
	"strings"

	"github.com/chzyer/readline"
)

func init() {
	RegisterCmd(&Cmd{
		Name:   "set",
		PcFunc: pcSet,
		Action: doSet,
		Desc:   "show/set shell settings",
		Usage: `  set vi-mode   [on|off]
  set box-style [simple|bold|double|light|round]`,
	})
}

func pcSet(c Client) readline.PrefixCompleterInterface {
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

func doSet(c Client, line string) {
	cli := c.(*client)
	args := splitFields(line, true)
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
		cli.Println(args[0], onoff(*flag))
	}

	if len(args) == 0 {
		box := cli.NewBox([]string{"NAME", "VALUE"})
		box.AppendRow("vi-mode", onoff(cli.conf.VimMode))
		box.AppendRow("box-style", cli.conf.BoxStyle)
		box.Render()
		return
	}
	switch strings.ToLower(args[0]) {
	case "vi-mode":
		parseflag(&cli.conf.VimMode)
	case "box-style":
		cli.conf.BoxStyle = parseBoxStyle(args[1])
		cli.Println("box-style", cli.conf.BoxStyle)
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
