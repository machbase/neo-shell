package shell

import "github.com/chzyer/readline"

func init() {
	RegisterCmd(&Cmd{
		Name:    "append",
		Aliases: []string{},
		PcFunc:  pcAppend,
		Action:  doAppend,
		Desc:    "append table",
		Usage: `  append [options]
  options:
    --input, -i   input file, (default: '-' stdin)
    <<EOF         EOF mark, use any string matches [a-zA-Z0-9]+ for "EOF"`,
	})
}

func pcAppend(cc Client) readline.PrefixCompleterInterface {
	return readline.PcItem("append")
}

func doAppend(cc Client, sqlText string) {

}
