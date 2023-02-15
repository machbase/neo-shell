package shell

import (
	"fmt"
	"os"
	"strings"

	"github.com/machbase/neo-shell/client"
	_ "github.com/machbase/neo-shell/internal/cmd"
)

type ShellCmd struct {
	Args       []string `arg:"" optional:"" name:"ARGS" passthrough:""`
	ServerAddr string   `name:"server" short:"s" default:"tcp://127.0.0.1:5655" help:"server address"`
	User       string   `name:"user" short:"u" default:"sys"`
	BoxStyle   string   `name:"box-style" default:"light" enum:"simple,bold,double,light,round" help:"box table style [simple|bold|double|light|round]"`
}

func Shell(cmd *ShellCmd) {
	clientConf := client.DefaultConfig()
	clientConf.ServerAddr = cmd.ServerAddr
	clientConf.BoxStyle = cmd.BoxStyle

	var command = ""
	if len(cmd.Args) > 0 {
		for i := range cmd.Args {
			if strings.Contains(cmd.Args[i], "\"") {
				cmd.Args[i] = strings.ReplaceAll(cmd.Args[i], "\"", "\\\"")
			}
			if strings.Contains(cmd.Args[i], " ") || strings.Contains(cmd.Args[i], "\t") {
				cmd.Args[i] = "\"" + cmd.Args[i] + "\""
			}
		}
		command = strings.TrimSpace(strings.Join(cmd.Args, " "))
	}
	interactive := len(command) == 0

	client := client.New(clientConf, interactive)
	if err := client.Start(); err != nil {
		fmt.Fprintln(os.Stdout, "ERR", err.Error())
		return
	}
	defer client.Stop()

	client.Run(command)
}
