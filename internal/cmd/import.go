package cmd

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/chzyer/readline"
	"github.com/machbase/neo-shell/client"
	"github.com/machbase/neo-shell/codec"
	"github.com/machbase/neo-shell/do"
	"github.com/machbase/neo-shell/util"
	spi "github.com/machbase/neo-spi"
)

func init() {
	client.RegisterCmd(&client.Cmd{
		Name:   "import",
		PcFunc: pcImport,
		Action: doImport,
		Desc:   "import table",
		Usage:  helpImport,
	})
}

const helpImport = `  import [options] <table>
    table               table name to write
  options:
    --input,-i <file>  input file, (default: '-' stdin)
    --format,-f        file format [csv] (default:'csv')
    --no-header        there is no header, do not skip first line (default)
    --header           first line is header, skip it
    --method           write method [insert|append] (default:'insert')
	--create-table     create table if it doesn't exist (default:false)
	--truncate-table   truncate table ahead importing new data (default:false)
    --delimiter,-d     csv delimiter (default:',')
    --tz               timezone for handling datetime
    --timeformat,-t    time format [ns|ms|s|<timeformat>] (default:'ns')
       ns, us, ms, s
         represents unix epoch time in nano-, micro-, milli- and seconds for each
       timeformat
         consult "help timeformat"
    --eof <string>     specify eof line, use any string matches [a-zA-Z0-9]+ (default: '.')`

type ImportCmd struct {
	Table         string         `arg:"" name:"table"`
	Input         string         `name:"input" short:"i" default:"-"`
	HasHeader     bool           `name:"header" negatable:""`
	EofMark       string         `name:"eof" default:"."`
	InputFormat   string         `name:"format" short:"f" default:"csv"`
	Method        string         `name:"method" default:"insert" enum:"insert,append"`
	CreateTable   bool           `name:"create-table" default:"false"`
	TruncateTable bool           `name:"truncate-table" default:"false"`
	Delimiter     string         `name:"delimiter" short:"d" default:","`
	TimeFormat    string         `name:"timeformat" short:"t" default:"ns"`
	TimeLocation  *time.Location `name:"tz" default:"UTC"`
	Help          bool           `kong:"-"`
}

func pcImport() readline.PrefixCompleterInterface {
	return readline.PcItem("import")
}

func doImport(ctx *client.ActionContext) {
	cmd := &ImportCmd{}
	parser, err := client.Kong(cmd, func() error { ctx.Println(helpImport); cmd.Help = true; return nil })
	if err != nil {
		ctx.Println(err.Error())
		return
	}

	_, err = parser.Parse(util.SplitFields(ctx.Line, true))
	if cmd.Help {
		return
	}
	if err != nil {
		ctx.Println(err.Error())
		return
	}

	var r *bufio.Reader
	if cmd.Input == "-" {
		r = bufio.NewReader(ctx.Stdin)
	} else {
		f, err := os.Open(cmd.Input)
		if err != nil {
			ctx.Println(err.Error())
			return
		}
		defer f.Close()
		r = bufio.NewReader(f)
	}

	_desc, err := do.Describe(ctx.DB, cmd.Table, false)
	if err != nil {
		ctx.Printfln("ERR fail to get table info '%s', %s", cmd.Table, err.Error())
		return
	}
	desc := (_desc).(*do.TableDescription)

	if ctx.Interactive {
		ctx.Printfln("# Enter %s⏎ to quit", cmd.EofMark)
		colNames := desc.Columns.Columns().Names()
		ctx.Println("#", strings.Join(colNames, cmd.Delimiter))

		buff := []byte{}
		for {
			bs, _, err := r.ReadLine()
			if err != nil {
				break
			}
			if string(bs) == cmd.EofMark {
				break
			}
			buff = append(buff, bs...)
		}
		r = bufio.NewReader(bytes.NewReader(buff))
	}

	decoder := codec.NewDecoderBuilder().
		SetReader(r).
		SetColumns(desc.Columns.Columns()).
		SetCsvDelimieter(cmd.Delimiter).
		Build(cmd.InputFormat)

	var appender spi.Appender
	hold := []string{}
	lineno := 0
	for {
		vals, err := decoder.NextRow()
		if err != nil {
			if err != io.EOF {
				ctx.Println("ERR", err.Error())
			}
			break
		}
		lineno++

		if len(vals) != len(desc.Columns) {
			ctx.Printfln("line %d contains %d columns, but expected %d", lineno, len(vals), len(desc.Columns))
			break
		}
		if cmd.Method == "insert" {
			for i := 0; i < len(desc.Columns); i++ {
				hold = append(hold, "?")
			}
			query := fmt.Sprintf("insert into %s values(%s)", cmd.Table, strings.Join(hold, ","))
			if result := ctx.DB.Exec(query, vals...); result.Err() != nil {
				ctx.Println(result.Err().Error())
				break
			}
			hold = hold[:0]
		} else { // append
			if appender == nil {
				appender, err = ctx.DB.Appender(cmd.Table)
				if err != nil {
					ctx.Println("ERR", err.Error())
					break
				}
				defer appender.Close()
			}

			err = appender.Append(vals...)
			if err != nil {
				ctx.Println("ERR", err.Error())
				break
			}
		}
	}
	ctx.Printfln("import total %d record(s) %sed", lineno, cmd.Method)
}
