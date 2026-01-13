package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"

	"github.com/OutOfBedlam/jsh/engine"
	"github.com/OutOfBedlam/jsh/native"
	"github.com/OutOfBedlam/jsh/root"
	"github.com/machbase/neo-jsh/internal/machcli"
	"github.com/machbase/neo-jsh/internal/pretty"
)

//go:embed internal/usr/*
var usrFS embed.FS

// JSH options:
//  1. -C "script" : command to execute
//     ex: jsh -C "console.println(require('process').argv[2])" helloworld
//  2. script file : execute script file
//     ex: jsh script.js arg1 arg2
//  3. no args : start interactive shell
//     ex: jsh
func main() {
	var fstabs engine.FSTabs
	var neoHost string
	var neoUser string
	var neoPassword string
	src := flag.String("C", "", "command to execute")
	scf := flag.String("S", "", "configured file to start from")
	flag.Var(&fstabs, "v", "volume to mount (format: /mountpoint=source)")
	flag.StringVar(&neoHost, "server", "127.0.0.1:5654", "machbase-neo host (default: 127.0.0.1:5654)")
	flag.StringVar(&neoUser, "user", "sys", "user name (default: sys)")
	flag.StringVar(&neoPassword, "password", "manager", "password (default: manager)")
	flag.Parse()

	conf := engine.Config{}
	if *scf != "" {
		// when it starts with "-s", read secret box
		if err := engine.ReadSecretBox(*scf, &conf); err != nil {
			fmt.Println("Error reading secret file:", err.Error())
			os.Exit(1)
		}
	} else {
		// otherwise, use command args to build ExecPass
		conf.Code = *src
		conf.FSTabs = fstabs
		conf.Args = flag.Args()
		conf.Default = "/usr/bin/neo-shell.js" // default script to run if no args
		conf.Env = map[string]any{
			"PATH":         "/usr/bin:/usr/lib:/sbin:/lib:/work",
			"HOME":         "/work",
			"PWD":          "/work",
			"NEO_HOST":     neoHost,
			"NEO_USER":     neoUser,
			"NEO_PASSWORD": engine.SecureString(neoPassword),
		}
		conf.Aliases = map[string]string{
			"describe": "show table",
			"desc":     "show table",
		}
	}
	if !conf.FSTabs.HasMountPoint("/") {
		conf.FSTabs = append([]engine.FSTab{root.RootFSTab()}, conf.FSTabs...)
	}
	if !conf.FSTabs.HasMountPoint("/usr") {
		dirfs, _ := fs.Sub(usrFS, "internal/usr")
		conf.FSTabs = append(conf.FSTabs, engine.FSTab{MountPoint: "/usr", FS: dirfs})
	}
	if !conf.FSTabs.HasMountPoint("/work") {
		dirfs, _ := engine.DirFS(".")
		conf.FSTabs = append(conf.FSTabs, engine.FSTab{MountPoint: "/work", FS: dirfs})
	}
	conf.ExecBuilder = func(code string, args []string, env map[string]any) (*exec.Cmd, error) {
		self, err := os.Executable()
		if err != nil {
			return nil, err
		}
		conf := engine.Config{
			Code:   code,
			Args:   args,
			FSTabs: fstabs,
			Env:    env,
		}
		secretBox, err := engine.NewSecretBox(conf)
		if err != nil {
			return nil, err
		}
		execCmd := exec.Command(self, "-S", secretBox.FilePath(), args[0])
		return execCmd, nil
	}
	eng, err := engine.New(conf)
	if err != nil {
		fmt.Println(err.Error())
		os.Exit(1)
	}
	native.Enable(eng)
	eng.RegisterNativeModule("@jsh/machcli", machcli.Module)
	eng.RegisterNativeModule("@jsh/pretty", pretty.Module)

	os.Exit(eng.Main())
}
