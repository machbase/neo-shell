package shell

import (
	"bufio"
	"fmt"
	"net"
	"os"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/machbase/neo-grpc/machrpc"
	"github.com/machbase/neo-shell/api"
	"github.com/machbase/neo-shell/internal/out_csv"
	"github.com/machbase/neo-shell/internal/out_default"
	"github.com/machbase/neo-shell/internal/out_json"
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
  arguments:
    query         sql query to execute
  options:
    --output,-o <file>     output file (default:'-' stdout)
    --format,-f <format>   output format
      -          default format
      csv        csv format
      json       json format
    --delimiter,-d       csv delimiter (default:',')
    --[no-]rownum        include rownum as first column (default:true)
    --timeformat,-t      time format [ns|ms|s|<timeformat>] (default:'ns')
      ns, us, ms, s
        represents unix epoch time in nano-, micro-, milli- and seconds for each
      timeformat
        consult "help timeformat"
    --tz                  timezone for handling datetime
    --[no-]heading        print header
    --precision,-p <int>  set precision of float value to force round`

type SqlCmd struct {
	Output       string         `name:"output" short:"o" default:"-"`
	Heading      bool           `name:"heading" negatable:"" default:"true"`
	TimeLocation *time.Location `name:"tz" default:"UTC"`
	Format       string         `name:"format" default:"-" enum:"-,csv,json"`
	Delimiter    string         `name:"delimiter" short:"d" default:","`
	Rownum       bool           `name:"rownum" negatable:"" default:"true"`
	TimeFormat   string         `name:"timeformat" short:"t" default:"ns"`
	Precision    int            `name:"precision" short:"p" default:"-1"`
	Interactive  bool           `kong:"-"`
	Help         bool           `kong:"-"`
	Query        []string       `arg:"" name:"query" passthrough:""`
}

func pcSql(cc Client) readline.PrefixCompleterInterface {
	cli := cc.(*client)
	return readline.PcItem("sql",
		readline.PcItemDynamic(cli.SqlHistory),
	)
}

func doSql(cc Client, cmdLine string) {
	cmd := &SqlCmd{}
	parser, err := Kong(cmd, func() error { cc.Println(helpSql); cmd.Help = true; return nil })
	if err != nil {
		cc.Println("ERR", err.Error())
		return
	}
	_, err = parser.Parse(splitFields(cmdLine, false))
	if cmd.Help {
		return
	}
	if err != nil {
		cc.Println("ERR", err.Error())
		return
	}

	sqlText := stripQuote(strings.Join(cmd.Query, " "))

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

	var writer *bufio.Writer
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

	var renderCtx = &api.RowsContext{
		Writer:       writer,
		TimeLocation: cmd.TimeLocation,
		TimeFormat:   cmd.TimeFormat,
		Precision:    cmd.Precision,
		Rownum:       cmd.Rownum,
		Heading:      cmd.Heading,
	}
	var renderer api.RowsRenderer
	switch cmd.Format {
	default:
		renderCtx.HeaderHeight = 4
		renderer = &out_default.Exporter{
			Style:           "light",
			SeparateColumns: cmd.Interactive,
			DrawBorder:      cmd.Interactive,
		}
	case "csv":
		renderCtx.HeaderHeight = 1
		exporter := &out_csv.Exporter{}
		exporter.SetDelimiter(cmd.Delimiter)
		renderer = exporter
	case "json":
		renderCtx.HeaderHeight = 0
		renderer = &out_json.Exporter{}
	}
	if renderer == nil {
		return
	}

	if err := cli.exportRows(renderCtx, rows, renderer, cmd.Interactive); err != nil {
		cli.Println("ERR", err.Error())
	}
}

func (cli *client) exportRows(ctx *api.RowsContext, rows *machrpc.Rows, renderer api.RowsRenderer, interactive bool) error {
	cols, err := rows.Columns()
	if err != nil {
		return err
	}

	windowHeight := 0
	if interactive && term.IsTerminal(0) {
		if _, height, err := term.GetSize(0); err == nil {
			windowHeight = height
		}
	}
	pageHeight := windowHeight - 1
	if ctx.Heading {
		pageHeight -= ctx.HeaderHeight
	}
	nextPauseRow := pageHeight

	ctx.ColumnNames = cli.columnNames(cols, ctx.TimeLocation, false)
	ctx.ColumnTypes = cli.columnTypes(cols, false)

	renderer.OpenRender(ctx)
	defer renderer.CloseRender()

	buf := makeBuffer(cols)
	nrow := 0
	for rows.Next() {
		err := rows.Scan(buf...)
		if err != nil {
			cli.Println("ERR", err.Error())
			return err
		}
		nrow++

		renderer.RenderRow(buf)

		if nextPauseRow > 0 && nextPauseRow == nrow {
			nextPauseRow += pageHeight
			renderer.PageFlush(ctx.Heading)
			if !cli.pauseForMore() {
				return nil
			}
		}

		if nextPauseRow <= 0 && nrow%1000 == 0 {
			renderer.PageFlush(false)
		}
	}
	return nil
}

func (cli *client) columnNames(cols []*machrpc.Column, tz *time.Location, withRowNum bool) []string {
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
			names[i+colIdxOffset] = fmt.Sprintf("%s(%s)", cols[i].Name, tz.String())
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
