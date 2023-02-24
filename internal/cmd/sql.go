package cmd

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/machbase/neo-shell/client"
	"github.com/machbase/neo-shell/codec"
	"github.com/machbase/neo-shell/do"
	"github.com/machbase/neo-shell/stream"
	"github.com/machbase/neo-shell/util"
	spi "github.com/machbase/neo-spi"
	"golang.org/x/term"
)

func init() {
	client.RegisterCmd(&client.Cmd{
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
	  box        box format (default)
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
	Format       string         `name:"format" short:"f" default:"box" enum:"box,csv,json"`
	Delimiter    string         `name:"delimiter" short:"d" default:","`
	Rownum       bool           `name:"rownum" negatable:"" default:"true"`
	TimeFormat   string         `name:"timeformat" short:"t" default:"default"`
	Precision    int            `name:"precision" short:"p" default:"-1"`
	Interactive  bool           `kong:"-"`
	Help         bool           `kong:"-"`
	Query        []string       `arg:"" name:"query" passthrough:""`
}

func pcSql() readline.PrefixCompleterInterface {
	return readline.PcItem("sql",
		readline.PcItemDynamic(client.SqlHistory),
	)
}

func doSql(ctx *client.ActionContext) {
	cmd := &SqlCmd{}
	parser, err := client.Kong(cmd, func() error { ctx.Println(helpSql); cmd.Help = true; return nil })
	if err != nil {
		ctx.Println("ERR", err.Error())
		return
	}
	_, err = parser.Parse(util.SplitFields(ctx.Line, false))
	if cmd.Help {
		return
	}
	if err != nil {
		ctx.Println("ERR", err.Error())
		return
	}

	var outputPath = util.StripQuote(cmd.Output)
	output, err := stream.NewOutputStream(outputPath)
	if err != nil {
		ctx.Println("ERR", err.Error())
	}

	if outputPath == "-" {
		cmd.Interactive = ctx.Interactive
	} else {
		cmd.Interactive = false
	}

	encoder := codec.NewEncoderBuilder(cmd.Format).
		SetOutputStream(output).
		SetTimeLocation(cmd.TimeLocation).
		SetTimeFormat(cmd.TimeFormat).
		SetPrecision(cmd.Precision).
		SetRownum(cmd.Rownum).
		SetHeading(cmd.Heading).
		SetBoxStyle("light").
		SetBoxSeparateColumns(cmd.Interactive).
		SetBoxDrawBorder(cmd.Interactive).
		SetCsvDelimieter(cmd.Delimiter).
		Build()

	headerHeight := 0
	switch cmd.Format {
	default: // "box"
		headerHeight = 4
	case "csv":
		headerHeight = 1
	case "json":
		headerHeight = 0
	}

	windowHeight := 0
	if cmd.Interactive && term.IsTerminal(0) {
		if _, height, err := term.GetSize(0); err == nil {
			windowHeight = height
		}
	}
	pageHeight := windowHeight - 1
	if cmd.Heading {
		pageHeight -= headerHeight
	}
	nextPauseRow := int64(pageHeight)

	queryCtx := &do.QueryContext{
		DB: ctx.DB,
		OnFetchStart: func(cols spi.Columns) {
			encoder.Open(cols)
		},
		OnFetch: func(nrow int64, values []any) bool {
			err := encoder.AddRow(values)
			if err != nil {
				ctx.Println("ERR", err.Error())
			}
			if nextPauseRow > 0 && nextPauseRow == nrow {
				nextPauseRow += int64(pageHeight)
				encoder.Flush(cmd.Heading)
				if !pauseForMore() {
					return false
				}
			}
			if nextPauseRow <= 0 && nrow%1000 == 0 {
				encoder.Flush(false)
			}
			return true
		},
		OnFetchEnd: func() {
			encoder.Close()
		},
	}

	sqlText := util.StripQuote(strings.Join(cmd.Query, " "))
	msg, err := do.Query(queryCtx, sqlText)
	if err != nil {
		ctx.Println("ERR", err.Error())
	} else {
		ctx.Println(msg)
	}
	client.AddSqlHistory(sqlText)
}

func pauseForMore() bool {
	fmt.Fprintf(os.Stdout, ":")
	// switch stdin into 'raw' mode
	if oldState, err := term.MakeRaw(int(os.Stdin.Fd())); err == nil {
		b := make([]byte, 3)
		if _, err = os.Stdin.Read(b); err == nil {
			term.Restore(int(os.Stdin.Fd()), oldState)
			// remove ':' prompt'd line
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
