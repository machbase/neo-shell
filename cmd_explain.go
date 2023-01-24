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

func pcExplain(c Client) readline.PrefixCompleterInterface {
	return readline.PcItem("explain")
}

func doExplain(c Client, line string, interactive bool) {
	cli := c.(*client)
	plan, err := cli.db.Explain(line)
	if err != nil {
		cli.Println(err.Error())
		return
	}
	cli.Println(plan)
}
