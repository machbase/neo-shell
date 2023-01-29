package shell

import (
	"sort"
	"strings"

	"github.com/chzyer/readline"
)

func init() {
	RegisterCmd(&Cmd{
		Name:   "help",
		PcFunc: pcHelp,
		Action: doHelp,
		Desc:   "Display this message, use 'help [command]'",
	})
}

func pcHelp(cli Client) readline.PrefixCompleterInterface {
	return readline.PcItem("help", readline.PcItemDynamic(func(line string) []string {
		lst := make([]string, 0)
		for k := range commands {
			lst = append(lst, k)
		}
		return lst
	}))
}

func doHelp(cli Client, line string) {
	fields := splitFields(line, true)
	if len(fields) > 0 {
		if cmd, ok := commands[fields[0]]; ok {
			cli.Println(cmd.Desc)

			if len(cmd.Usage) > 0 {
				cli.Println("Usage:")
				lines := strings.Split(cmd.Usage, "\n")
				for _, l := range lines {
					cli.Println(l)
				}
			}
			return
		}
		switch fields[0] {
		case "timeformat":
			helpTimeFormat(cli)
			return
		}
	}
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
	cli.Printfln("    %-*s %s", 10, "exit", "Exit shell")
}

func helpTimeFormat(cli Client) {
	cli.Println(`
  timeformat  
    abbreviations
      Default,-      2006-01-02 15:04:05.999
      Numeric        01/02 03:04:05PM '06 -0700
      Ansic          Mon Jan _2 15:04:05 2006
      Unix           Mon Jan _2 15:04:05 MST 2006
      Ruby           Mon Jan 02 15:04:05 -0700 2006
      RFC822         02 Jan 06 15:04 MST
      RFC822Z        02 Jan 06 15:04 -0700
      RFC850         Monday, 02-Jan-06 15:04:05 MST
      RFC1123        Mon, 02 Jan 2006 15:04:05 MST
      RFC1123Z       Mon, 02 Jan 2006 15:04:05 -0700
      RFC3339        2006-01-02T15:04:05Z07:00
      RFC3339Nano    2006-01-02T15:04:05.999999999Z07:00
      Kitchen        3:04:05PM
      Stamp          Jan _2 15:04:05
      StampMili      Jan _2 15:04:05.000
      StampMicro     Jan _2 15:04:05.000000
      StampNano      Jan _2 15:04:05.000000000
    custom format
       year   2006
       month  01
       day    02
       hour   03 or 15
       minute 04
       second 05 or with sub-seconds '05.999999'
`)
}
