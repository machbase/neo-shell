package main

import (
	"flag"
	"os"

	"github.com/machbase/neo-shell/entry"
)

func main() {
	self, err := os.Executable()
	if err != nil {
		panic(err)
	}
	entry.Main(flag.CommandLine, []string{self}, os.Args[1:])
}
