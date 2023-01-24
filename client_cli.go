package shell

import (
	"fmt"
	"os"
	"strings"
	"time"
)

type ShellCmd struct {
	Args       []string `arg:"" optional:"" name:"ARGS" passthrough:""`
	ServerAddr string   `name:"server" short:"s" default:"tcp://127.0.0.1:5655" help:"server address"`
	User       string   `name:"user" short:"u" default:"sys"`
	Heading    bool     `name:"heading" negatable:"" default:"true"`
	TimeZone   string   `name:"tz" default:"UTC" help:"timezone to handle datetime"`
	Format     string   `name:"format" default:"-" enum:"-,csv" help:"outout format"`
	BoxStyle   string   `name:"box-style" default:"light" enum:"simple,bold,double,light,round" help:"box table style [simple|bold|double|light|round]"`
}

func Shell(cmd *ShellCmd) {
	clientConf := DefaultConfig()
	clientConf.ServerAddr = cmd.ServerAddr
	clientConf.Heading = cmd.Heading
	clientConf.Format = cmd.Format
	clientConf.BoxStyle = cmd.BoxStyle
	if tz, err := time.LoadLocation(cmd.TimeZone); err == nil {
		clientConf.TimeLocation = tz
	} else {
		fmt.Fprintln(os.Stdout, "ERR timezone", err.Error())
		return
	}

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
