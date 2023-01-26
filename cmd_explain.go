package shell

import "github.com/chzyer/readline"

func init() {
	RegisterCmd(&Cmd{
		Name:    "explain",
		Aliases: []string{},
		PcFunc:  pcExplain,
		Action:  doExplain,
		Desc:    "explain <sql>",
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
