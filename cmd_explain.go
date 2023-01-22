package shell

import "github.com/chzyer/readline"

func (cli *client) pcExplain() *readline.PrefixCompleter {
	return readline.PcItem("explain")
}

func (cli *client) doExplain(sqlText string) {
	plan, err := cli.db.Explain(sqlText)
	if err != nil {
		cli.Println(err.Error())
		return
	}
	cli.Println(plan)
}
