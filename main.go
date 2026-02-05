package main

import (
	"context"
	"embed"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"net"
	"os"
	"os/exec"

	"github.com/machbase/jsh/engine"
	"github.com/machbase/jsh/native"
	"github.com/machbase/jsh/root"
	"github.com/machbase/neo-shell/internal/machcli"
	"github.com/machbase/neo-shell/internal/pretty"
	"github.com/machbase/neo-shell/internal/session"
	"github.com/nyaosorg/go-readline-ny"
	"golang.org/x/term"
)

//go:embed internal/usr/*
var usrFS embed.FS

// JSH options:
//  1. -C "script" : command to execute
//     ex: neo-shell -C "console.println(require('process').argv[2])" helloworld
//  2. script file : execute script file
//     ex: neo-shell script.js arg1 arg2
//  3. no args : start interactive shell
//     ex: neo-shell
func main() {
	var fstabs engine.FSTabs
	var envVars engine.EnvVars = make(map[string]any)
	var neoHost string
	var neoUser string
	var neoPassword string
	var err error

	src := flag.String("C", "", "command to execute")
	scf := flag.String("S", "", "configured file to start from")
	flag.Var(&fstabs, "v", "volume to mount (format: /mountpoint=source)")
	flag.Var(&envVars, "e", "environment variable (format: name=value)")
	flag.StringVar(&neoHost, "server", "", "machbase-neo host")
	flag.StringVar(&neoUser, "user", "", "user name (default: sys)")
	flag.StringVar(&neoPassword, "password", "", "password (default: manager)")
	flag.Parse()

	conf := engine.Config{}
	if *scf != "" {
		// when it starts with "-S", read secret box
		if err := engine.ReadSecretBox(*scf, &conf); err != nil {
			fmt.Println("Error reading secret file:", err.Error())
			os.Exit(1)
		}
		if host, ok := conf.Env["NEO_HOST"]; ok {
			neoHost = host.(string)
		}
		if user, ok := conf.Env["NEO_USER"]; ok {
			neoUser = user.(string)
		}
		if pass, ok := conf.Env["NEO_PASSWORD"]; ok {
			neoPassword = pass.(engine.SecureString).Value()
		}
		if neoUser == "" {
			neoUser, err = readLine("User", "SYS")
			if err != nil {
				fmt.Println("Error reading User:", err.Error())
				os.Exit(1)
			}
			conf.Env["NEO_USER"] = neoUser
		}
		if neoPassword == "" {
			neoPassword, err = readPassword("Password", "manager")
			if err != nil {
				fmt.Println("Error reading Password:", err.Error())
				os.Exit(1)
			}
			conf.Env["NEO_PASSWORD"] = engine.SecureString(neoPassword)
		}
	} else {
		if neoHost == "" {
			neoHost, err = readLine("Server", "127.0.0.1:5654")
			if err != nil {
				fmt.Println("Error reading Server:", err.Error())
				os.Exit(1)
			}
		}
		if _, port, err := net.SplitHostPort(neoHost); err != nil {
			port, err = readLine("Port", "5654")
			if err != nil {
				fmt.Println("Error reading Port:", err.Error())
				os.Exit(1)
			}
			neoHost = net.JoinHostPort(neoHost, port)
		}
		if neoUser == "" {
			neoUser, err = readLine("User", "SYS")
			if err != nil {
				fmt.Println("Error reading User:", err.Error())
				os.Exit(1)
			}
		}
		if neoPassword == "" {
			neoPassword, err = readPassword("Password", "manager")
			if err != nil {
				fmt.Println("Error reading Password:", err.Error())
				os.Exit(1)
			}
		}
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
	for k, v := range envVars {
		conf.Env[k] = v
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
	// setup ExecBuilder to enable re-execution
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
	eng.RegisterNativeModule("@jsh/session", session.Module)
	eng.RegisterNativeModule("@jsh/machcli", machcli.Module)
	eng.RegisterNativeModule("@jsh/pretty", pretty.Module)

	// configure default session
	if err := session.Configure(session.Config{
		Server:   neoHost,
		User:     neoUser,
		Password: neoPassword,
	}); err != nil {
		fmt.Println("Error configuring session:", err.Error())
		os.Exit(1)
	}

	os.Exit(eng.Main())
}

func readPassword(prompt string, defaultValue string) (string, error) {
	if defaultValue != "" {
		prompt = fmt.Sprintf("%s [%s]", prompt, defaultValue)
	}
	fmt.Printf("%s: ", prompt)
	b, err := term.ReadPassword(int(os.Stdin.Fd()))
	fmt.Println()
	if len(b) == 0 && defaultValue != "" {
		return defaultValue, err
	}
	return string(b), err
}

func readLine(prompt string, defaultValue string) (string, error) {
	var ctx = context.Background()
	var editor = &readline.Editor{
		PromptWriter: func(w io.Writer) (int, error) {
			if defaultValue != "" {
				return io.WriteString(w, fmt.Sprintf("%s [%s]: ", prompt, defaultValue))
			} else {
				return io.WriteString(w, fmt.Sprintf("%s: ", prompt))
			}
		},
	}
	text, err := editor.ReadLine(ctx)
	if err == nil && text == "" {
		text = defaultValue
	}
	return text, err
}
