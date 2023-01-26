package main

import (
	"github.com/alecthomas/kong"
	shell "github.com/machbase/neo-shell"
)

func main() {
	var cli shell.ShellCmd
	_ = kong.Parse(&cli,
		kong.HelpOptions{NoAppSummary: false, Compact: true, FlagsLast: true},
		kong.UsageOnError(),
	)
	shell.Shell(&cli)
}
