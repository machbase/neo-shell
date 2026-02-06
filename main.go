package main

import (
	"flag"
	"os"

	"github.com/machbase/neo-shell/entry"
)

func main() {
	entry.Main(flag.CommandLine, os.Args[1:])
}
