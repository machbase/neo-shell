package shell

import (
	"encoding/csv"
	"fmt"
	"math"
	"os"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/alecthomas/kong"
	"github.com/chzyer/readline"
)

func init() {
	RegisterCmd(&Cmd{
		Name:    "export",
		Aliases: []string{},
		PcFunc:  pcExport,
		Action:  doExport,
		Desc:    "export table",
		Usage: `  export [options] <table>
    table               table name to read
  options:
    --output,-o <file>  input file, (default: '-' stdout)
    --format,-f        file format [csv] (default:'csv')
    --no-header        do not export header (default)
    --header           export header
    --delimiter,-d     csv delimiter (default:',')
    --timeformat,-t    time format [ns|ms|s|<date-time-format>] (default:'ns')
       ns, us, ms, s
         represents unix epoch time in nano-, micro-, milli- and seconds for each
       date-time-format  ex) '2006-01-02 15:04:05.999'
         year   2006
         month  01
         day    02
         hour   03 or 15
         minute 04
         second 05 or with sub-seconds '05.999999'
    --precision,-p <uint> precision of float value, if 0, disable round value (default: 0)`,
	})
}

type ExportCmd struct {
	Table      string `arg:"" name:"table"`
	Output     string `name:"output" short:"o" default:"-"`
	Header     bool   `name:"header" negatable:""`
	Delimiter  string `name:"delimiter" short:"d" default:","`
	TimeFormat string `name:"timeFormat" short:"t" default:"ns"`
	Precision  uint   `name:"precision" short:"p" default:"0"`
}

func pcExport(cli Client) readline.PrefixCompleterInterface {
	return readline.PcItem("export")
}

func doExport(cli Client, cmdLine string) {
	cmd := &ExportCmd{}
	parser, err := kong.New(cmd, kong.HelpOptions{Compact: true}, kong.Exit(func(int) {}))
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	x := splitFields(cmdLine)
	_, err = parser.Parse(x)
	if err != nil {
		cli.Println("ERR", err.Error(), strings.Join(x, "|"))
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

func makeCsvValues(buf []any, tz *time.Location, timeFormat string, precision uint) []string {
	var round = func(v float64, p uint) float64 {
		if p == 0 {
			return v
		}
		ratio := math.Pow(10, float64(p))
		return math.Round(v*ratio) / ratio
	}

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
			cols[i] = fmt.Sprintf("%.*f", precision, round(*v, precision))
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
