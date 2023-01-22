package shell

import (
	"io"
	"log"
	"strings"

	"github.com/chzyer/readline"
)

func (cli *client) doPrompt() {
	prompt := "\033[31mmachsql»\033[0m "
	rl, err := readline.NewEx(&readline.Config{
		Prompt:                 prompt,
		HistoryFile:            "/tmp/readline.tmp",
		DisableAutoSaveHistory: true,
		AutoComplete:           cli.completer(),
		InterruptPrompt:        "^C",
		EOFPrompt:              "exit",
		Stdin:                  cli.conf.Stdin,
		Stdout:                 cli.conf.Stdout,
		Stderr:                 cli.conf.Stderr,
		HistorySearchFold:      true,
		FuncFilterInputRune:    filterInput,
	})
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	rl.CaptureExitSignal()
	rl.SetVimMode(cli.conf.VimMode)

	log.SetOutput(rl.Stderr())

	var parts []string
	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		if line == "exit" || line == "exit;" {
			goto exit
		} else if strings.HasPrefix(line, "help") {
			if !strings.HasSuffix(line, ";") {
				line = line + ";"
			}
		}
		parts = append(parts, line)
		if !strings.HasSuffix(line, ";") {
			rl.SetPrompt("         ")
			continue
		}

		line = strings.Join(parts, " ")
		rl.SaveHistory(line)

		line = strings.TrimSuffix(line, ";")
		parts = parts[:0]
		rl.SetPrompt(prompt)
		cli.Run(line, true)
	}
exit:
}

func usage(w io.Writer, completer *readline.PrefixCompleter, cmd string) {
	io.WriteString(w, "commands:\n")
	io.WriteString(w, completer.Tree("    "))
}

func filterInput(r rune) (rune, bool) {
	switch r {
	case readline.CharCtrlZ: // block CtrlZ feature
		return r, false
	}
	return r, true
}

func (cli *client) completer() *readline.PrefixCompleter {
	var completer = readline.NewPrefixCompleter(
		cli.pcShow(),
		cli.pcChart(),
		cli.pcSet(),
		cli.pcExplain(),
		cli.pcDescribe(),
		cli.pcWalk(),
	)
	return completer
}

/*
// Function constructor - constructs new function for listing given directory
func listFiles(path string) func(string) []string {
	return func(line string) []string {
		names := make([]string, 0)
		files, _ := os.ReadDir(path)
		for _, f := range files {
			names = append(names, f.Name())
		}
		return names
	}
}
*/

func (cli *client) listTables() func(string) []string {
	return func(line string) []string {
		rows, err := cli.db.Query("select NAME, TYPE, FLAG from M$SYS_TABLES order by NAME")
		if err != nil {
			// sess.log.Errorf("select m$sys_tables fail; %s", err.Error())
			return nil
		}
		defer rows.Close()
		// rt := []prompt.Suggest{}
		rt := []string{}
		for rows.Next() {
			var name string
			var typ int
			var flg int
			rows.Scan(&name, &typ, &flg)
			//desc := tableTypeDesc(typ, flg)
			// rt = append(rt, prompt.Suggest{Text: name, Description: desc})
			rt = append(rt, name)
		}
		return rt
	}
}
