package shell

import (
	"encoding/csv"
	"fmt"
	"os"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/alecthomas/kong"
	"github.com/chzyer/readline"
)

func init() {
	RegisterCmd(&Cmd{
		Name:   "export",
		PcFunc: pcExport,
		Action: doExport,
		Desc:   "export table",
		Usage:  helpExport,
	})
}

const helpExport = `  export [options] <table>
    table               table name to read
  options:
    --output,-o <file>   output file (default:'-' stdout)
    --format,-f <format> output format [csv] (default:'csv')
    --[no-]header        export header (default:false)
    --delimiter,-d      csv delimiter (default:',')
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

type ExportCmd struct {
	Table      string `arg:"" name:"table"`
	Output     string `name:"output" short:"o" default:"-"`
	Format     string `name:"format" short:"f" default:"csv"`
	Header     bool   `name:"header" negatable:""`
	Delimiter  string `name:"delimiter" short:"d" default:","`
	TimeFormat string `name:"timeFormat" short:"t" default:"ns"`
	Precision  int    `name:"precision" short:"p" default:"-1"`
}

func pcExport(cli Client) readline.PrefixCompleterInterface {
	return readline.PcItem("export")
}

func doExport(cli Client, cmdLine string) {
	cmd := &ExportCmd{}
	parser, err := kong.New(cmd, kong.HelpOptions{Compact: true}, kong.Exit(func(int) {}),
		kong.Help(
			func(options kong.HelpOptions, ctx *kong.Context) error {
				cli.Println(helpExport)
				return nil
			}))
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	_, err = parser.Parse(splitFields(cmdLine, false))
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}

	var w *csv.Writer
	if cmd.Output == "-" {
		w = csv.NewWriter(cli.Stdout())
		defer w.Flush()
	} else {
		f, err := os.OpenFile(cmd.Output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			cli.Println(err.Error())
			return
		}
		w = csv.NewWriter(f)
		defer func() {
			w.Flush()
			f.Close()
		}()
	}
	w.Comma, _ = utf8.DecodeRuneInString(cmd.Delimiter)

	db := cli.Database()
	rows, err := db.Query("select * from " + cmd.Table + " order by time")
	if err != nil {
		fmt.Println("ERR", err.Error())
		return
	}
	defer rows.Close()

	cols, err := rows.Columns()
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}

	buf := makeBuffer(cols)
	names := make([]string, len(cols))
	for i := range cols {
		names[i] = cols[i].Name
	}

	if cmd.Header {
		w.Write(names)
	}

	for rows.Next() {
		err := rows.Scan(buf...)
		if err != nil {
			cli.Println("ERR", err.Error())
			return
		}
		vs := makeCsvValues(buf, cli.TimeLocation(), cmd.TimeFormat, cmd.Precision)
		w.Write(vs)
	}
}

func makeCsvValues(buf []any, tz *time.Location, timeFormat string, precision int) []string {
	cols := make([]string, len(buf))
	for i, r := range buf {
		if r == nil {
			cols[i] = "NULL"
			continue
		}
		switch v := r.(type) {
		case *string:
			cols[i] = *v
		case *time.Time:
			switch timeFormat {
			case "ns":
				cols[i] = strconv.FormatInt(v.UnixNano(), 10)
			case "ms":
				cols[i] = strconv.FormatInt(v.UnixMilli(), 10)
			case "us":
				cols[i] = strconv.FormatInt(v.UnixMicro(), 10)
			case "s":
				cols[i] = strconv.FormatInt(v.Unix(), 10)
			default:
				cols[i] = v.In(tz).Format(timeFormat)
			}
		case *float64:
			if precision < 0 {
				cols[i] = fmt.Sprintf("%f", *v)
			} else {
				cols[i] = fmt.Sprintf("%.*f", precision, *v)
			}
		case *int:
			cols[i] = strconv.FormatInt(int64(*v), 10)
		case *int32:
			cols[i] = strconv.FormatInt(int64(*v), 10)
		case *int64:
			cols[i] = strconv.FormatInt(*v, 10)
		default:
			cols[i] = fmt.Sprintf("%T", r)
		}
	}
	return cols
}
