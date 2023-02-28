package main

import (
	"fmt"

	"github.com/alecthomas/kong"
	shell "github.com/machbase/neo-shell"
	"github.com/machbase/neo-shell/client"
)

func main() {
	var cli shell.ShellCmd
	_ = kong.Parse(&cli,
		kong.HelpOptions{NoAppSummary: false, Compact: true, FlagsLast: true},
		kong.UsageOnError(),
		kong.Help(func(options kong.HelpOptions, ctx *kong.Context) error {
			serverAddr := "tcp://127.0.0.1:5655"
			if pref, err := client.LoadPref(); err == nil {
				serverAddr = pref.Server().Value()
			}
			fmt.Printf(`Usage: neoshell [<flags>] [<args>...]
  Flags:
    -h, --help             Show context-sensitive help.
    	--version          show version
    -s, --server=<addr>    server address (default %s)
			`, serverAddr)
			return nil
		}),
	)
	shell.Shell(&cli)
}
