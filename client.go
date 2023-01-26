package shell

import (
	"io"
	"log"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/chzyer/readline"
	"github.com/machbase/neo-grpc/machrpc"
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

	Database() *machrpc.Client
	TimeLocation() *time.Location
}

var Formats = struct {
	Default string
	CSV     string
	Parse   func(string) string
}{
	Default: "-",
	CSV:     "csv",
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
	HistoryFile  string
	VimMode      bool
	Heading      bool
	TimeLocation *time.Location
	Format       string
	BoxStyle     string
	QueryTimeout time.Duration
	Lang         language.Tag
}

type client struct {
	conf *Config
	db   *machrpc.Client

	interactive bool
}

func DefaultConfig() *Config {
	return &Config{
		Stdin:        os.Stdin,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
		Prompt:       "\033[31mmachbase-neo»\033[0m ",
		HistoryFile:  "/tmp/readline.tmp",
		VimMode:      false,
		Heading:      true,
		QueryTimeout: 30 * time.Second,
		Lang:         language.English,
	}
}

func (c *Config) TimeZone() string {
	zone, _ := time.Now().In(c.TimeLocation).Zone()
	return zone
}

func New(conf *Config, interactive bool) Client {
	return &client{
		conf:        conf,
		interactive: interactive,
	}
}

func (cli *client) Start() error {
	machcli := machrpc.NewClient(machrpc.QueryTimeout(cli.conf.QueryTimeout))
	err := machcli.Connect(cli.conf.ServerAddr)
	if err != nil {
		return err
	}
	// TODO: check server reachable,
	// then return error if not reachabse

	cli.db = machcli
	return nil
}

func (cli *client) Stop() {
	if cli.db != nil {
		cli.db.Disconnect()
	}
}

func (cli *client) Database() *machrpc.Client {
	return cli.db
}

func (cli *client) TimeLocation() *time.Location {
	return cli.conf.TimeLocation
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
	Name    string
	Aliases []string
	PcFunc  func(cli Client) readline.PrefixCompleterInterface
	Action  func(cli Client, line string)
	Desc    string
	Usage   string
}

var commands = make(map[string]*Cmd)
var aliases = make(map[string]*Cmd)

func RegisterCmd(cmd *Cmd) {
	commands[cmd.Name] = cmd
	for _, a := range cmd.Aliases {
		aliases[a] = cmd
	}
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
	fields := splitFields(line)
	if len(fields) == 0 {
		return
	}

	cmdName := fields[0]
	cmd, ok := commands[cmdName]
	if !ok {
		cmd = aliases[cmdName]
	}
	if cmd == nil {
		// support trailing command
		// ex) select * from table \w
		tail := fields[len(fields)-1]
		if strings.HasPrefix(tail, `\`) && len(tail) > 1 {
			if cmd, ok = aliases[tail]; !ok {
				cmd = commands[tail[1:]]
			}
			if cmd != nil {
				line = strings.TrimSpace(line)
				line = line[0 : len(line)-len(tail)]
			}
		}
	} else {
		line = strings.TrimSpace(line[len(cmdName):])
	}

	if cmd != nil {
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
			rl.SetPrompt(">  ")
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

func splitFields(line string) []string {
	lastQuote := rune(0)
	f := func(c rune) bool {
		switch {
		case c == lastQuote:
			lastQuote = rune(0)
			return false
		case lastQuote != rune(0):
			return false
		case unicode.In(c, unicode.Quotation_Mark):
			lastQuote = c
			return false
		default:
			return unicode.IsSpace(c)
		}
	}
	fields := strings.FieldsFunc(line, f)

	for i, f := range fields {
		c := []rune(f)[0]
		if unicode.In(c, unicode.Quotation_Mark) {
			fields[i] = strings.Trim(f, string(c))
		}
	}
	return fields
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
