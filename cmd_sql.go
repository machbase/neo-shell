package shell

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"os"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/chzyer/readline"
	"github.com/machbase/neo-grpc/machrpc"
	"golang.org/x/term"
)

func init() {
	RegisterCmd(&Cmd{
		Name:   "sql",
		PcFunc: pcSql,
		Action: doSql,
		Desc:   "Execute sql query",
		Usage:  helpSql,
	})
}

const helpSql string = `  sql [options] <query>
    query         sql query to execute
  options:
    --output,-o <file>     output file (default:'-' stdout)
    --format,-f <format>   output format
      none       non-export mode (default)
      csv        csv format
      json       json format
    --delimiter,-d       delimiter for csv format (default:',')
    --[no-]rownum        include rownum as first column (default:true)
    --timeformat,-t      time format [ns|ms|s|<date-time-format>] (default:'ns')
      ns, us, ms, s
        represents unix epoch time in nano-, micro-, milli- and seconds for each
      date-time-format  ex) '2006-01-02 15:04:05.999'
        year   2006
        month  01
        day    02
        hour   03 or 15
        minute 04
        second 05 or with sub-seconds '05.999999'
    --precision,-p <int>  set precision of float value to force round`

type SqlCmd struct {
	Output      string   `name:"output" short:"o" default:"-"`
	Format      string   `name:"format" short:"f" enum:"none,csv,json" default:"none"`
	Delimiter   string   `name:"delimiter" short:"d" default:","`
	Rownum      bool     `name:"rownum" negatable:"" default:"true"`
	TimeFormat  string   `name:"timeFormat" short:"t" default:"ns"`
	Precision   int      `name:"precision" short:"p" default:"-1"`
	Interactive bool     `kong:"-"`
	Query       []string `arg:"" name:"query" passthrough:""`
}

func pcSql(cc Client) readline.PrefixCompleterInterface {
	cli := cc.(*client)
	return readline.PcItem("sql",
		readline.PcItemDynamic(cli.SqlHistory),
	)
}

func doSql(cc Client, cmdLine string) {
	cmd := &SqlCmd{}
	parser, err := kong.New(cmd, kong.HelpOptions{Compact: true}, kong.Exit(func(int) {}),
		kong.Help(
			func(options kong.HelpOptions, ctx *kong.Context) error {
				cc.Println(helpSql)
				return nil
			}))
	if err != nil {
		cc.Println("ERR", err.Error())
		return
	}
	_, err = parser.Parse(splitFields(cmdLine, false))
	if err != nil {
		cc.Println("ERR", err.Error())
		return
	}

	sqlText := stripQuote(strings.Join(cmd.Query, " "))
	// cc.Println("SQL", sqlText)
	// cc.Printfln("    %+v", cmd)

	db := cc.Database()
	rows, err := db.Query(sqlText)
	if err != nil {
		cc.Println("ERR", err.Error())
		return
	}
	defer rows.Close()

	cli := cc.(*client)
	if cc.Interactive() {
		cli.AddSqlHistory(sqlText)
	}

	if !rows.IsFetchable() {
		cli.Println(rows.Message())
		return
	}

	var writer io.Writer
	switch cmd.Output {
	case "-":
		cmd.Interactive = cc.Interactive()
		buf := bufio.NewWriter(cc.Stdout())
		defer func() {
			buf.Flush()
		}()
		writer = buf
	default:
		cmd.Interactive = false
		f, err := os.OpenFile(cmd.Output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			cc.Println("ERR", err.Error())
			return
		}
		buf := bufio.NewWriter(f)
		defer func() {
			buf.Flush()
			f.Close()
		}()
		writer = buf
	}

	// json       json format
	// chart.js   export result in json for chart.js
	// 		      if output file's extension is '.html', result json will be embeded in html.
	switch cmd.Format {
	default:
		cli.exportRowsNone(writer, rows, cmd)
	case "csv":
		cli.exportRowsCsv(writer, rows, cmd)
	case "json":
		cli.exportRowsJson(writer, rows, cmd)
	case "chart.js":
		cli.exportRowsChartJs(writer, rows, cmd)
	}
}

func (cli *client) columnNames(cols []*machrpc.Column, withRowNum bool) []string {
	var names []string
	var colIdxOffset int
	if withRowNum {
		names = make([]string, len(cols)+1)
		names[0] = "#"
		colIdxOffset = 1
	} else {
		names = make([]string, len(cols))
		colIdxOffset = 0
	}
	for i := range cols {
		if cols[i].Type == "datetime" {
			names[i+colIdxOffset] = fmt.Sprintf("%s(%s)", cols[i].Name, cli.conf.TimeLocation.String())
		} else {
			names[i+colIdxOffset] = cols[i].Name
		}
	}
	return names
}

func (cli *client) columnTypes(cols []*machrpc.Column, withRowNum bool) []string {
	var types []string
	var colIdxOffset int
	if withRowNum {
		types = make([]string, len(cols)+1)
		types[0] = "int64"
		colIdxOffset = 1
	} else {
		types = make([]string, len(cols))
		colIdxOffset = 0
	}
	for i := range cols {
		types[i+colIdxOffset] = cols[i].Type
	}
	return types
}

func (cli *client) pauseForMore() bool {
	cli.Print(":")
	// switch stdin into 'raw' mode
	if oldState, err := term.MakeRaw(int(os.Stdin.Fd())); err == nil {
		b := make([]byte, 3)
		if _, err = os.Stdin.Read(b); err == nil {
			term.Restore(int(os.Stdin.Fd()), oldState)
			// ':' prompt를 삭제한다.
			// erase line
			fmt.Fprintf(os.Stdout, "%s%s", "\x1b", "[2K")
			// cursor backward
			fmt.Fprintf(os.Stdout, "%s%s", "\x1b", "[1D")
			switch b[0] {
			case 'q', 'Q':
				return false
			default:
				return true
			}
		}
	}
	return true
}

func makeValues(rec []any, tz *time.Location, precision int) []string {
	cols := make([]string, len(rec))
	for i, r := range rec {
		if r == nil {
			cols[i] = "NULL"
			continue
		}
		switch v := r.(type) {
		case *string:
			cols[i] = *v
		case *time.Time:
			timeformat := "2006-01-02 15:04:05.000000"
			cols[i] = v.In(tz).Format(timeformat)
		case *float64:
			if precision < 0 {
				cols[i] = fmt.Sprintf("%f", *v)
			} else {
				cols[i] = fmt.Sprintf("%.*f", precision, *v)
			}
		case *int:
			cols[i] = fmt.Sprintf("%d", *v)
		case *int32:
			cols[i] = fmt.Sprintf("%d", *v)
		case *int64:
			cols[i] = fmt.Sprintf("%d", *v)
		default:
			cols[i] = fmt.Sprintf("%T", r)
		}
	}
	return cols
}

func makeBuffer(cols []*machrpc.Column) []any {
	rec := make([]any, len(cols))
	for i := range cols {
		switch cols[i].Type {
		case "int16":
			rec[i] = new(int16)
		case "int32":
			rec[i] = new(int32)
		case "int64":
			rec[i] = new(int64)
		case "datetime":
			rec[i] = new(time.Time)
		case "float":
			rec[i] = new(float32)
		case "double":
			rec[i] = new(float64)
		case "ipv4":
			rec[i] = new(net.IP)
		case "ipv6":
			rec[i] = new(net.IP)
		case "string":
			rec[i] = new(string)
		case "binary":
			rec[i] = new([]byte)
		}
	}
	return rec
}
