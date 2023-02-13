package shell

import (
	"errors"
	"io"
	"log"
	"os"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/machbase/neo-grpc/machrpc"
	"github.com/machbase/neo-grpc/mgmt"
	spi "github.com/machbase/neo-spi"
	"golang.org/x/net/context"
	"golang.org/x/text/language"
	"golang.org/x/text/message"
)

type Client interface {
	Boxer

	Start() error
	Stop()

	Run(command string)

	Interactive() bool

	Write(p []byte) (int, error)
	Print(args ...any)
	Printf(format string, args ...any)
	Println(args ...any)
	Printfln(format string, args ...any)

	Stdin() io.Reader
	Stdout() io.Writer
	Stderr() io.Writer

	Database() spi.Database
}

var Formats = struct {
	Default string
	CSV     string
	JSON    string
	Parse   func(string) string
}{
	Default: "-",
	CSV:     "csv",
	JSON:    "json",
	Parse: func(str string) string {
		switch str {
		default:
			return "-"
		case "csv":
			return "csv"
		}
	},
}

type Config struct {
	ServerAddr   string
	Stdin        io.ReadCloser
	Stdout       io.Writer
	Stderr       io.Writer
	Prompt       string
	PromptCont   string
	HistoryFile  string
	VimMode      bool
	BoxStyle     string
	QueryTimeout time.Duration
	Lang         language.Tag
}

type client struct {
	conf *Config
	db   spi.DatabaseClient

	interactive   bool
	remoteSession bool
}

func DefaultConfig() *Config {
	return &Config{
		Stdin:        os.Stdin,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
		Prompt:       "\033[31mmachbase-neo»\033[0m ",
		PromptCont:   "\033[37m>\033[0m  ",
		HistoryFile:  "/tmp/readline.tmp",
		VimMode:      false,
		QueryTimeout: 30 * time.Second,
		Lang:         language.English,
	}
}

// func (c *Config) TimeZone() string {
// 	zone, _ := time.Now().In(c.TimeLocation).Zone()
// 	return zone
// }

func New(conf *Config, interactive bool) Client {
	return &client{
		conf:        conf,
		interactive: interactive,
	}
}

func (cli *client) Start() error {
	machcli := machrpc.NewClient()
	err := machcli.Connect(cli.conf.ServerAddr, machrpc.QueryTimeout(cli.conf.QueryTimeout))
	if err != nil {
		return err
	}

	// check connectivity to server
	serverInfo, err := machcli.GetServerInfo()
	if err != nil {
		return err
	}

	cli.remoteSession = true
	if strings.HasPrefix(cli.conf.ServerAddr, "tcp://127.0.0.1:") {
		cli.remoteSession = false
	} else if !strings.HasPrefix(cli.conf.ServerAddr, "tcp://") {
		serverPid := int(serverInfo.Runtime.Pid)
		if os.Getppid() != serverPid {
			// if my ppid is same with server pid, this client was invoked from server directly.
			// which means connected remotely via ssh.
			cli.remoteSession = false
		}
	}

	cli.db = machcli
	return nil
}

func (cli *client) Stop() {
	if cli.db != nil {
		cli.db.Disconnect()
	}
}

func (cli *client) Database() spi.Database {
	return cli.db
}

func (cli *client) ShutdownServer() error {
	if cli.remoteSession {
		return errors.New("remote session is not allowed to shutdown")
	}
	mgmtcli, err := cli.NewManagementClient()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	_, err = mgmtcli.Shutdown(ctx, &mgmt.ShutdownRequest{})
	if err != nil {
		return err
	}
	return nil
}

func (cli *client) NewManagementClient() (mgmt.ManagementClient, error) {
	conn, err := machrpc.MakeGrpcConn(cli.conf.ServerAddr)
	if err != nil {
		return nil, err
	}
	return mgmt.NewManagementClient(conn), nil
}

func (cli *client) Run(command string) {
	if len(command) == 0 {
		cli.Prompt()
	} else {
		cli.Process(command)
	}
}

func (cli *client) Interactive() bool {
	return cli.interactive
}

func (cli *client) Config() *Config {
	return cli.conf
}

type Cmd struct {
	Name   string
	PcFunc func(cli Client) readline.PrefixCompleterInterface
	Action func(cli Client, line string)
	Desc   string
	Usage  string
}

var commands = make(map[string]*Cmd)

func RegisterCmd(cmd *Cmd) {
	commands[cmd.Name] = cmd
}

func (cli *client) completer() readline.PrefixCompleterInterface {
	pc := make([]readline.PrefixCompleterInterface, 0)
	for _, cmd := range commands {
		if cmd.PcFunc != nil {
			pc = append(pc, cmd.PcFunc(cli))
		}
	}
	return readline.NewPrefixCompleter(pc...)
}

func (cli *client) Process(line string) {
	fields := splitFields(line, true)
	if len(fields) == 0 {
		return
	}

	cmdName := fields[0]
	if cmd, ok := commands[cmdName]; ok {
		line = strings.TrimSpace(line[len(cmdName):])
		cmd.Action(cli, line)
	} else {
		doSql(cli, line)
	}
}

func (cli *client) Prompt() {
	rl, err := readline.NewEx(&readline.Config{
		Prompt:                 cli.conf.Prompt,
		HistoryFile:            cli.conf.HistoryFile,
		DisableAutoSaveHistory: true,
		AutoComplete:           cli.completer(),
		InterruptPrompt:        "^C",
		EOFPrompt:              "exit",
		Stdin:                  cli.conf.Stdin,
		Stdout:                 cli.conf.Stdout,
		Stderr:                 cli.conf.Stderr,
		HistorySearchFold:      true,
		FuncFilterInputRune:    filterInput,
	})
	if err != nil {
		panic(err)
	}
	defer rl.Close()

	rl.CaptureExitSignal()
	rl.SetVimMode(cli.conf.VimMode)

	log.SetOutput(rl.Stderr())

	var parts []string
	for {
		line, err := rl.Readline()
		if err == readline.ErrInterrupt {
			if len(line) == 0 {
				break
			} else {
				continue
			}
		} else if err == io.EOF {
			break
		}

		line = strings.TrimSpace(line)
		if line == "" {
			parts = parts[:0]
			rl.SetPrompt(cli.conf.Prompt)
			continue
		}
		if len(parts) == 0 {
			if line == "exit" || line == "exit;" {
				goto exit
			} else if strings.HasPrefix(line, "help") {
				goto madeline
			} else if line == "set" || strings.HasPrefix(line, "set ") {
				goto madeline
			}
		}

		parts = append(parts, line)
		if !strings.HasSuffix(line, ";") {
			rl.SetPrompt(cli.conf.PromptCont)
			continue
		}
		line = strings.Join(parts, " ")

	madeline:
		rl.SaveHistory(line)

		line = strings.TrimSuffix(line, ";")
		parts = parts[:0]
		rl.SetPrompt(cli.conf.Prompt)
		cli.Process(line)
	}
exit:
}

func filterInput(r rune) (rune, bool) {
	switch r {
	case readline.CharCtrlZ: // block CtrlZ feature
		return r, false
	}
	return r, true
}

func (cli *client) listTables() func(string) []string {
	return func(line string) []string {
		rows, err := cli.db.Query("select NAME, TYPE, FLAG from M$SYS_TABLES order by NAME")
		if err != nil {
			return nil
		}
		defer rows.Close()
		rt := []string{}
		for rows.Next() {
			var name string
			var typ int
			var flg int
			rows.Scan(&name, &typ, &flg)
			rt = append(rt, name)
		}
		return rt
	}
}

func (cli *client) bytesUnit(v uint64) string {
	p := message.NewPrinter(cli.conf.Lang)
	f := float64(v)
	u := ""
	switch {
	case v > 1024*1024*1024:
		f = f / (1024 * 1024 * 1024)
		u = "GB"
	case v > 1024*1024:
		f = f / (1024 * 1024)
		u = "MB"
	case v > 1024:
		f = f / 1024
		u = "KB"
	}
	return p.Sprintf("%.1f %s", f, u)
}

func (cli *client) Printer() *message.Printer {
	return message.NewPrinter(cli.conf.Lang)
}

var sqlHistory = make([]string, 0)

func (cli *client) AddSqlHistory(sqlText string) {
	if len(sqlHistory) > 10 {
		sqlHistory = sqlHistory[len(sqlHistory)-10:]
	}

	sqlHistory = append(sqlHistory, sqlText)
}

func (cli *client) SqlHistory(line string) []string {
	return sqlHistory
}
