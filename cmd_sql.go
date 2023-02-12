package shell

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/machbase/neo-grpc/spi"
	"github.com/machbase/neo-shell/renderer/boxrenderer"
	"github.com/machbase/neo-shell/renderer/csvrenderer"
	"github.com/machbase/neo-shell/renderer/jsonrenderer"
	"github.com/machbase/neo-shell/sink/execsink"
	"github.com/machbase/neo-shell/sink/filesink"
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
    --timeformat,-t      time format [ns|ms|s|<timeformat>] (default:'default')
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
	Format       string         `name:"format" short:"f" default:"-" enum:"-,csv,json"`
	Delimiter    string         `name:"delimiter" short:"d" default:","`
	Rownum       bool           `name:"rownum" negatable:"" default:"true"`
	TimeFormat   string         `name:"timeformat" short:"t" default:"default"`
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

	rows, err := cc.Database().Query(sqlText)
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

	var sink spi.Sink
	var outputPath = stripQuote(cmd.Output)
	var outputFields = strings.Fields(outputPath)
	if outputFields[0] == "exec" {
		binArgs := strings.TrimSpace(strings.TrimPrefix(outputPath, "exec"))
		sink, err = execsink.New(binArgs)
		if err != nil {
			cli.Println("ERR", err.Error())
			return
		}
	} else {
		sink, err = filesink.New(outputPath)
		if err != nil {
			cli.Println("ERR", err.Error())
			return
		}
	}

	if outputPath == "-" {
		cmd.Interactive = cc.Interactive()
	} else {
		cmd.Interactive = false
	}

	var renderCtx = &spi.RowsRendererContext{
		Sink:         sink,
		TimeLocation: cmd.TimeLocation,
		TimeFormat:   spi.GetTimeformat(cmd.TimeFormat),
		Precision:    cmd.Precision,
		Rownum:       cmd.Rownum,
		Heading:      cmd.Heading,
	}
	var renderer spi.RowsRenderer
	switch cmd.Format {
	default:
		renderCtx.HeaderHeight = 4
		renderer = boxrenderer.NewRowsRenderer("light", cmd.Interactive, cmd.Interactive)
	case "csv":
		renderCtx.HeaderHeight = 1
		renderer = csvrenderer.NewRowsRenderer(cmd.Delimiter)
	case "json":
		renderCtx.HeaderHeight = 0
		renderer = jsonrenderer.NewRowsRenderer()
	}
	if renderer == nil {
		return
	}

	if err := cli.exportRows(renderCtx, rows, renderer, cmd.Interactive); err != nil {
		cli.Println("ERR", err.Error())
	}
}

func (cli *client) exportRows(ctx *spi.RowsRendererContext, rows spi.Rows, renderer spi.RowsRenderer, interactive bool) error {
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

	ctx.ColumnNames = cols.Names(ctx.TimeLocation)
	ctx.ColumnTypes = cols.Types()

	renderer.OpenRender(ctx)
	defer renderer.CloseRender()

	buf := cols.MakeBuffer()
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
