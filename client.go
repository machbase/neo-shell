package shell

import (
	"fmt"
	"io"
	"os"
	"strings"
	"time"
	"unicode"

	"github.com/machbase/neo-grpc/machrpc"
)

type Client interface {
	Close()
	Run(command string, interactive bool)
	RunPrompt()
}

type Config struct {
	ServerAddr   string
	Stdin        io.ReadCloser
	Stdout       io.Writer
	Stderr       io.Writer
	VimMode      bool
	Heading      bool
	LocalTime    bool
	QueryTimeout time.Duration
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
	}
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

func (cli *client) Println(args ...any) {
	fmt.Fprintln(cli.conf.Stdout, args...)
}

func (cli *client) Printf(format string, args ...any) {
	fmt.Fprintf(cli.conf.Stdout, format, args...)
}

func (cli *client) Writeln(args ...any) {
	fmt.Fprintln(cli.conf.Stdout, args...)
}

func (cli *client) Writef(format string, args ...any) {
	fmt.Fprintf(cli.conf.Stdout, format+"\r\n", args...)
}

func (cli *client) Run(line string, interactive bool) {
	fields := splitFields(line)
	if len(fields) == 0 {
		return
	}
	switch strings.ToLower(fields[0]) {
	case "help":
		cmd := strings.TrimSpace(strings.ToLower(line[4:]))
		usage(cli.conf.Stdout, cli.completer(), cmd)
	case "show":
		cli.doShow(fields[1:])
	case "explain":
		sql := strings.TrimSpace(line[7:])
		cli.doExplain(sql)
	case "describe":
		object := strings.TrimSpace(line[8:])
		cli.doDescribe(object)
	case "desc":
		object := strings.TrimSpace(line[4:])
		cli.doDescribe(object)
	case "chart":
		cli.doChart(fields[1:])
	case "set":
		cli.doSet(fields[1:])
	case "sql":
		sql := strings.TrimSpace(line[3:])
		cli.doSql(sql)
	case "walk":
		sql := strings.TrimSpace(line[4:])
		cli.doWalk(sql)
	default:
		if interactive {
			cli.doWalk(line)
		} else {
			cli.doSql(line)
		}
	}
}

func (cli *client) RunPrompt() {
	cli.doPrompt()
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
