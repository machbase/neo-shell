package shell

import (
	"fmt"
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
    --export,-e <format>   export query result into output file
      none       non-export mode (default)
      csv        csv format
      json       json format
      chart.js   export result in json for chart.js
                 if output file's extension is '.html', result json will be embeded in html.
    --output,-o <file>  output file (default:'-' stdout)
    --delimiter,-d      delimiter for csv format (default:',')
    --[no-]rownum       include rownum as first column (default:true)
    --timeformat,-t     time format [ns|ms|s|<date-time-format>] (default:'ns')
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
	Output     string   `name:"output" short:"o" default:"-"`
	Export     string   `name:"export" short:"e" default:"none"`
	Delimiter  string   `name:"delimiter" short:"d" default:","`
	Rownum     bool     `name:"rownum" negatable:"" default:"true"`
	TimeFormat string   `name:"timeFormat" short:"t" default:"ns"`
	Precision  int      `name:"precision" short:"p" default:"-1"`
	Query      []string `arg:"" name:"query" passthrough:""`
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
	cc.Println("SQL", sqlText)
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

	cli.exportRowsNone(rows, cmd.Rownum, cli.Interactive(), cmd.Precision)
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

func (cli *client) exportRowsNone(rows *machrpc.Rows, includeRowNum bool, interactiveMode bool, precision int) {
	cols, err := rows.Columns()
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	rec := makeBuffer(cols)

	names := cli.columnNames(cols, includeRowNum)

	box := cli.newBox(names, !interactiveMode)

	windowHeight := 0
	//windowWidth := 0
	if term.IsTerminal(0) {
		if _, height, err := term.GetSize(0); err == nil {
			windowHeight = height
			//windowWidth = width
		}
	}

	height := windowHeight - 4
	if cli.conf.Heading {
		height--
	}

	nrow := 0
	for {
		if !rows.Next() {
			box.Render()
			box.ResetRows()
			return
		}
		err := rows.Scan(rec...)
		if err != nil {
			cli.Println("ERR", err.Error())
			return
		}
		nrow++
		vs := makeValues(rec, cli.conf.TimeLocation, precision)
		values := make([]any, len(vs)+1)
		values[0] = nrow
		for i := range vs {
			values[i+1] = vs[i]
		}
		box.AppendRow(values...)

		if windowHeight > 0 && nrow%height == 0 {
			box.Render()
			box.ResetRows()
			if interactiveMode {
				if !pauseForMore(cli) {
					return
				}
			} else {
				box.ResetHeaders()
			}
		}
	}
}

func pauseForMore(cli Client) bool {
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
