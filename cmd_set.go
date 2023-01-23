package shell

import (
	"strings"

	"github.com/chzyer/readline"
)

func init() {
	RegisterCmd(&Cmd{
		Name:    "set",
		Aliases: []string{},
		PcFunc:  pcSet,
		Action:  doSet,
		Desc:    "show/set machsql shell settings",
		Usage:   "set [local-time | vi-mode | heading] [on|off]",
	})
}

func pcSet(c Client) readline.PrefixCompleterInterface {
	return readline.PcItem("set",
		readline.PcItem("local-time",
			readline.PcItem("on"),
			readline.PcItem("off"),
		),
		readline.PcItem("box-style",
			readline.PcItem("default"),
			readline.PcItem("bold"),
			readline.PcItem("double"),
			readline.PcItem("light"),
			readline.PcItem("round"),
		),
		readline.PcItem("vi-mode",
			readline.PcItem("on"),
			readline.PcItem("off"),
		),
		readline.PcItem("heading",
			readline.PcItem("on"),
			readline.PcItem("off"),
		),
		readline.PcItem("format",
			readline.PcItem(Formats.Default),
			readline.PcItem(Formats.CSV),
		),
	)
}

func doSet(c Client, line string, interactive bool) {
	cli := c.(*client)
	args := splitFields(line)
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
		cli.Println("local-time", onoff(cli.conf.LocalTime))
		cli.Println("vi-mode   ", onoff(cli.conf.VimMode))
		cli.Println("box-style ", cli.conf.BoxStyle)
		cli.Println("heading   ", onoff(cli.conf.Heading))
		cli.Println("format    ", cli.conf.Format)
		return
	}
	switch strings.ToLower(args[0]) {
	case "local-time":
		parseflag(&cli.conf.LocalTime)
	case "vi-mode":
		parseflag(&cli.conf.VimMode)
	case "heading":
		parseflag(&cli.conf.Heading)
	case "box-style":
		cli.conf.BoxStyle = parseBoxStyle(args[1])
		cli.Println("box-style", cli.conf.BoxStyle)
	case "format":
		cli.conf.Format = Formats.Parse(args[1])
		cli.Println("format", cli.conf.Format)
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
