package shell

import "github.com/chzyer/readline"

func init() {
	RegisterCmd(&Cmd{
		Name:   "explain",
		PcFunc: pcExplain,
		Action: doExplain,
		Desc:   "Display execution plan of query",
		Usage:  "  explain <query>",
	})
}

func pcExplain(cli Client) readline.PrefixCompleterInterface {
	return readline.PcItem("explain")
}

func doExplain(cli Client, line string) {
	db := cli.Database()
	plan, err := db.Explain(line)
	if err != nil {
		cli.Println(err.Error())
		return
	}
	cli.Println(plan)
}
