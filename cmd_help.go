package shell

import (
	"sort"
	"strings"

	"github.com/chzyer/readline"
)

func init() {
	RegisterCmd(&Cmd{
		Name:    "help",
		Aliases: []string{"\\h"},
		PcFunc:  pcHelp,
		Action:  doHelp,
		Desc:    "display this message, help [command]",
	})
}

func pcHelp(c Client) readline.PrefixCompleterInterface {
	return readline.PcItem("help", readline.PcItemDynamic(func(line string) []string {
		lst := make([]string, 0)
		for k := range commands {
			lst = append(lst, k)
		}
		return lst
	}))
}

func doHelp(c Client, line string, interactive bool) {
	cli := c.(*client)

	if cmd, ok := commands[line]; ok {
		cli.Println("command:", cmd.Name)
		cli.Println(cmd.Usage)
	} else {
		cli.Println("commands")
		keys := make([]string, 0, len(commands))
		for k := range commands {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(i, j int) bool {
			if keys[i] == "help" {
				return false
			} else if keys[j] == "help" {
				return true
			}
			return keys[i] < keys[j]
		})
		for _, k := range keys {
			cmd := commands[k]
			cli.Printfln("    %-*s %s", 12, cmd.Name, cmd.Desc)
			if len(cmd.Usage) > 0 {
				lines := strings.Split(cmd.Usage, "\n")
				for _, l := range lines {
					cli.Printfln("%s %s", strings.Repeat(" ", 16), l)
				}
			}
		}
		cli.Printfln("    %-*s %s", 12, "exit", "exit machsql shell")
	}
}