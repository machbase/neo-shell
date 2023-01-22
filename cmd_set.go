package shell

import (
	"strings"

	"github.com/chzyer/readline"
)

func (cli *client) pcSet() *readline.PrefixCompleter {
	return readline.PcItem("set",
		readline.PcItem("key",
			readline.PcItem("vi"),
			readline.PcItem("emacs"),
		),
		readline.PcItem("heading",
			readline.PcItem("on"),
			readline.PcItem("off"),
		),
	)
}

func (cli *client) doSet(args ...string) {
	if len(args) <= 2 || strings.ToLower(args[0]) != "set" {
		return
	}
	switch strings.ToLower(args[1]) {
	case "key":
		if strings.ToLower(args[2]) == "vi" {
			cli.conf.VimMode = true
			cli.Println("vi key mode")
		} else {
			cli.conf.VimMode = false
			cli.Println("emacs key mode")
		}
	case "heading":
		if strings.ToLower(args[2]) == "on" {
			cli.conf.Heading = true
			cli.Println("heading on")
		} else {
			cli.conf.Heading = false
			cli.Println("heading off")
		}
	}
}
