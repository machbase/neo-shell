package shell

import (
	"fmt"
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
	Close()
	Run(command string, interactive bool)
	Prompt()

	Write(p []byte) (int, error)
	Print(args ...any)
	Printf(format string, args ...any)
	Println(args ...any)
	Printfln(format string, args ...any)
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
}

func DefaultConfig() *Config {
	return &Config{
		Stdin:        os.Stdin,
		Stdout:       os.Stdout,
		Stderr:       os.Stderr,
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

func New(conf *Config) (Client, error) {
	machcli := machrpc.NewClient(machrpc.QueryTimeout(conf.QueryTimeout))
	err := machcli.Connect(conf.ServerAddr)
	if err != nil {
		return nil, err
	}
	cli := &client{
		conf: conf,
		db:   machcli,
	}
	return cli, nil
}

func (cli *client) Close() {
	if cli.db != nil {
		cli.db.Disconnect()
	}
}

func (cli *client) Config() *Config {
	return cli.conf
}

func (cli *client) Write(p []byte) (int, error) {
	return cli.conf.Stdout.Write(p)
}

func (cli *client) Print(args ...any) {
	fmt.Fprint(cli.conf.Stdout, args...)
}

func (cli *client) Printf(format string, args ...any) {
	str := fmt.Sprintf(format, args...)
	fmt.Fprint(cli.conf.Stdout, str)
}

func (cli *client) Println(args ...any) {
	fmt.Fprintln(cli.conf.Stdout, args...)
}

func (cli *client) Printfln(format string, args ...any) {
	fmt.Fprintf(cli.conf.Stdout, format+"\r\n", args...)
}

type Cmd struct {
	Name    string
	Aliases []string
	PcFunc  func(cli Client) readline.PrefixCompleterInterface
	Action  func(cli Client, line string, interactive bool)
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

func (cli *client) Run(line string, interactive bool) {
	fields := splitFields(line)
	if len(fields) == 0 {
		return
	}

	cmdName := fields[0]
	cmd, ok := commands[cmdName]
	if !ok {
		cmd = aliases[cmdName]
	}

	if cmd != nil {
		line = strings.TrimSpace(line[len(cmdName):])
		cmd.Action(cli, line, interactive)
	} else {
		tail := strings.TrimSpace(fields[len(fields)-1])
		if tail == `\w` || tail == `\walk` && len(fields) > 1 {
			line = strings.Join(fields[0:len(fields)-1], " ")
			doWalk(cli, line, interactive)
		} else {
			doSql(cli, line, interactive)
		}
	}
}

func (cli *client) Prompt() {
	prompt := "\033[31mmachsql»\033[0m "
	rl, err := readline.NewEx(&readline.Config{
		Prompt:                 prompt,
		HistoryFile:            "/tmp/readline.tmp",
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
			rl.SetPrompt("         ")
			continue
		}
		line = strings.Join(parts, " ")

	madeline:
		rl.SaveHistory(line)

		line = strings.TrimSuffix(line, ";")
		parts = parts[:0]
		rl.SetPrompt(prompt)
		cli.Run(line, true)
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
