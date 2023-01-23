package shell

import (
	"fmt"
	"os"
	"strings"
)

type ShellCmd struct {
	Args       []string `arg:"" optional:"" name:"ARGS" passthrough:""`
	ServerAddr string   `name:"server" short:"s" default:"tcp://127.0.0.1:5655" help:"server address"`
	User       string   `name:"user" short:"u" default:"sys"`
	Heading    bool     `name:"heading" negatable:"" default:"true"`
	Format     string   `name:"format" default:"-" enum:"-,csv" help:"outout format"`
}

func Shell(cmd *ShellCmd) {
	clientConf := DefaultConfig()
	clientConf.ServerAddr = cmd.ServerAddr
	clientConf.Heading = cmd.Heading
	clientConf.Format = cmd.Format

	client, err := New(clientConf)
	if err != nil {
		fmt.Fprintln(os.Stdout, "ERR", err.Error())
		return
	}
	defer client.Close()

	var command = ""
	if len(cmd.Args) > 0 {
		command = strings.TrimSpace(strings.Join(cmd.Args, " "))
	}

	if len(command) > 0 {
		client.Run(command, false)
	} else {
		client.Prompt()
	}
}
