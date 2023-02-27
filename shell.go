package shell

import (
	"fmt"
	"os"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/machbase/neo-shell/client"
	_ "github.com/machbase/neo-shell/internal/cmd"
)

type ShellCmd struct {
	Args       []string `arg:"" optional:"" name:"ARGS" passthrough:""`
	Version    bool     `name:"version" default:"false" help:"show version"`
	ServerAddr string   `name:"server" short:"s" default:"tcp://127.0.0.1:5655" help:"server address"`
	User       string   `name:"user" short:"u" default:"sys"`
}

func Shell(cmd *ShellCmd) {
	if cmd.Version {
		fmt.Fprintf(os.Stdout, "neoshell %s (%s %s)\n", versionString, buildTimestamp, versionGitSHA)
		return
	}

	for _, f := range cmd.Args {
		if f == "--help" || f == "-h" {
			targetCmd := client.FindCmd(cmd.Args[0])
			if targetCmd == nil {
				fmt.Fprintf(os.Stdout, "unknown sub-command %s\n\n", cmd.Args[0])
				return
			}
			fmt.Fprintf(os.Stdout, "%s\n", targetCmd.Usage)
			return
		}
	}

	clientConf := client.DefaultConfig()
	clientConf.ServerAddr = cmd.ServerAddr

	// enum:"simple,bold,double,light,round"
	clientConf.BoxStyle = "light"

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

var (
	versionString   = ""
	versionGitSHA   = ""
	buildTimestamp  = ""
	goVersionString = ""
)

type Version struct {
	Major  int    `json:"major"`
	Minor  int    `json:"minor"`
	Patch  int    `json:"patch"`
	GitSHA string `json:"git"`
}

var _version *Version

func GetVersion() *Version {
	if _version == nil {
		v, err := semver.NewVersion(versionString)
		if err != nil {
			_version = &Version{}
		} else {
			_version = &Version{
				Major:  int(v.Major()),
				Minor:  int(v.Minor()),
				Patch:  int(v.Patch()),
				GitSHA: versionGitSHA,
			}
		}
	}
	return _version
}

func VersionString() string {
	return fmt.Sprintf("%s (%v, %v)", versionString, versionGitSHA, buildTimestamp)
}

func BuildCompiler() string {
	return goVersionString
}

func BuildTimestamp() string {
	return buildTimestamp
}
