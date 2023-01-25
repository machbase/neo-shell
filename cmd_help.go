package shell

import (
	"sort"
	"strings"

	"github.com/chzyer/readline"
)

func init() {
	RegisterCmd(&Cmd{
		Name:    "help",
		Aliases: []string{`\h`},
		PcFunc:  pcHelp,
		Action:  doHelp,
		Desc:    "display this message, use 'help [command]'",
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

	fields := splitFields(line)
	if cmd, ok := commands[fields[0]]; ok {
		cli.Println(cmd.Desc)

		ali := strings.Join(cmd.Aliases, ", ")
		if len(ali) > 0 {
			cli.Println("Alias:")
			cli.Println("  ", ali)
		}
		cli.Println("Usage:")
		if len(cmd.Usage) > 0 {
			lines := strings.Split(cmd.Usage, "\n")
			for _, l := range lines {
				cli.Println(l)
			}
		}
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
			cli.Printfln("    %-*s %s", 10, cmd.Name, cmd.Desc)
		}
		cli.Printfln("    %-*s %s", 10, "exit", "exit shell")
	}
}
