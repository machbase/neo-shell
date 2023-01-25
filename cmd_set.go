package shell

import (
	"strings"
	"time"

	"github.com/chzyer/readline"
)

func init() {
	RegisterCmd(&Cmd{
		Name:    "set",
		Aliases: []string{},
		PcFunc:  pcSet,
		Action:  doSet,
		Desc:    "show/set shell settings",
		Usage: `  set vi-mode   [on|off]
  set heading   [on|off]
  set tz        [time-zone|UTC|Local]
  set box-style [simple|bold|double|light|round]
  set format    [-|csv]`,
	})
}

func pcSet(c Client) readline.PrefixCompleterInterface {
	return readline.PcItem("set",
		readline.PcItem("tz",
			readline.PcItem("UTC"),
			readline.PcItem("Local"),
		),
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

func doSet(c Client, line string) {
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
		box := cli.NewBox([]any{"NAME", "VALUE"}, false)
		box.AppendRow("tz", cli.conf.TimeLocation.String())
		box.AppendRow("vi-mode", onoff(cli.conf.VimMode))
		box.AppendRow("heading", onoff(cli.conf.Heading))
		box.AppendRow("box-style", cli.conf.BoxStyle)
		box.AppendRow("format", cli.conf.Format)
		box.Render()
		return
	}
	switch strings.ToLower(args[0]) {
	case "tz":
		if strings.ToLower(args[1]) == "local" {
			cli.conf.TimeLocation = time.Local
		} else {
			if tz, err := time.LoadLocation(args[1]); err == nil {
				cli.conf.TimeLocation = tz
			} else {
				cli.Println("ERR", err.Error())
			}
		}
		cli.Println("tz", cli.conf.TimeLocation.String())
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
