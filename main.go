package main

import (
	"embed"
	"flag"
	"fmt"
	"io/fs"
	"os"

	"github.com/OutOfBedlam/jsh/engine"
	"github.com/OutOfBedlam/jsh/native"
	"github.com/machbase/neo-jsh/internal/machcli"
	"github.com/machbase/neo-jsh/internal/pretty"
)

//go:embed internal/usr/*
var usrFS embed.FS

// JSH options:
//  1. -c "script" : command to execute
//     ex: jsh -c "console.println(require('process').argv[2])" helloworld
//  2. script file : execute script file
//     ex: jsh script.js arg1 arg2
//  3. no args : start interactive shell
//     ex: jsh
func main() {
	var fstabs engine.FSTabs
	src := flag.String("c", "", "command to execute")
	scf := flag.String("s", "", "configured file to start from")
	flag.Var(&fstabs, "v", "volume to mount (format: /mountpoint=source)")
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
		conf.Default = "/usr/bin/shell.js" // default script to run if no args
		conf.Env = map[string]any{
			"PATH": "/sbin:/lib:/usr/bin:/usr/lib:/work",
			"HOME": "/work",
			"PWD":  "/work",
		}
	}
	conf.AddFSTabHook(func(tabs engine.FSTabs) engine.FSTabs {
		if !tabs.HasMountPoint("/") {
			tabs = append([]engine.FSTab{native.RootFSTab()}, tabs...)
		}
		if !tabs.HasMountPoint("/usr") {
			dirfs, _ := fs.Sub(usrFS, "internal/usr")
			tabs = append(tabs, engine.FSTab{MountPoint: "/usr", FS: dirfs})
		}
		if !tabs.HasMountPoint("/work") {
			dirfs, _ := engine.DirFS(".")
			tabs = append(tabs, engine.FSTab{MountPoint: "/work", FS: dirfs})
		}
		return tabs
	})
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
