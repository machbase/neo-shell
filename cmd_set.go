package shell

import (
	"strings"

	"github.com/chzyer/readline"
)

func (cli *client) pcSet() *readline.PrefixCompleter {
	return readline.PcItem("set",
		readline.PcItem("local-time",
			readline.PcItem("on"),
			readline.PcItem("off"),
		),
		readline.PcItem("vi-mode",
			readline.PcItem("on"),
			readline.PcItem("off"),
		),
		readline.PcItem("heading",
			readline.PcItem("on"),
			readline.PcItem("off"),
		),
	)
}

func (cli *client) doSet(args []string) {
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
		cli.Writeln(args[0], onoff(*flag))
	}

	if len(args) == 0 {
		cli.Writeln("local-time", onoff(cli.conf.LocalTime))
		cli.Writeln("vi-mode   ", onoff(cli.conf.VimMode))
		cli.Writeln("heading   ", onoff(cli.conf.Heading))
		return
	}
	switch strings.ToLower(args[0]) {
	case "local-time":
		parseflag(&cli.conf.LocalTime)
	case "vi-mode":
		parseflag(&cli.conf.VimMode)
	case "heading":
		parseflag(&cli.conf.Heading)
	}
}
