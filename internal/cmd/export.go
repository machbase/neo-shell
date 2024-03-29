package cmd

import (
	"compress/gzip"
	"time"

	"github.com/chzyer/readline"
	"github.com/machbase/neo-shell/client"
	"github.com/machbase/neo-shell/codec"
	"github.com/machbase/neo-shell/do"
	"github.com/machbase/neo-shell/stream"
	"github.com/machbase/neo-shell/util"
	spi "github.com/machbase/neo-spi"
)

func init() {
	client.RegisterCmd(&client.Cmd{
		Name:   "export",
		PcFunc: pcExport,
		Action: doExport,
		Desc:   "Export table",
		Usage:  helpExport,
	})
}

const helpExport = `  export [options] <table>
  arguments:
    table                    table name to read
  options:
    -o,--output <file>       output file (default:'-' stdout)
    -f,--format <format>     output format
                csv          csv format (default)
                json         json format
       --compress <method>   compression method [gzip] (default is not compressed)
       --[no-]heading        print header message (default:false)
       --[no-]footer         print footer message (default:false)
    -d,--delimiter           csv delimiter (default:',')
       --tz                  timezone for handling datetime
    -t,--timeformat          time format [ns|ms|s|<timeformat>] (default:'ns')
                             consult "help timeformat"
    -p,--precision <int>     set precision of float value to force round
`

type ExportCmd struct {
	Table   string `arg:"" name:"table"`
	Output  string `name:"output" short:"o" default:"-"`
	Heading bool   `name:"heading" negatable:""`
	Footer  bool   `name:"footer" negatable:""`

	TimeLocation *time.Location `name:"tz"`
	Format       string         `name:"format" short:"f" default:"csv" enum:"box,csv,json"`
	Compress     string         `name:"compress" default:"-" enum:"-,gzip"`
	Delimiter    string         `name:"delimiter" short:"d" default:","`
	Timeformat   string         `name:"timeformat" short:"t"`
	Precision    int            `name:"precision" short:"p" default:"-1"`
	Help         bool           `kong:"-"`
}

func pcExport() readline.PrefixCompleterInterface {
	return readline.PcItem("export")
}

func doExport(ctx *client.ActionContext) {
	cmd := &ExportCmd{}
	parser, err := client.Kong(cmd, func() error { ctx.Println(helpExport); cmd.Help = true; return nil })
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

	if cmd.TimeLocation == nil {
		cmd.TimeLocation = ctx.Pref().TimeZone().TimezoneValue()
	}
	if cmd.Timeformat == "" {
		cmd.Timeformat = ctx.Pref().Timeformat().Value()
	}
	cmd.Timeformat = util.StripQuote(cmd.Timeformat)

	if len(cmd.Table) == 0 {
		ctx.Println("ERR", "no table is specified")
		return
	}

	var outputPath = util.StripQuote(cmd.Output)
	var output spi.OutputStream
	output, err = stream.NewOutputStream(outputPath)
	if err != nil {
		ctx.Println("ERR", err.Error())
		return
	}
	defer output.Close()

	if cmd.Compress == "gzip" {
		gw := gzip.NewWriter(output)
		defer func() {
			if gw != nil {
				err := gw.Close()
				if err != nil {
					ctx.Println("ERR", err.Error())
				}
			}
		}()
		output = &stream.WriterOutputStream{Writer: gw}
	}

	encoder := codec.NewEncoderBuilder(cmd.Format).
		SetOutputStream(output).
		SetTimeLocation(cmd.TimeLocation).
		SetTimeFormat(cmd.Timeformat).
		SetPrecision(cmd.Precision).
		SetRownum(false).
		SetHeading(cmd.Heading).
		SetBoxStyle("light").
		SetBoxSeparateColumns(true).
		SetBoxDrawBorder(true).
		SetCsvDelimieter(cmd.Delimiter).
		Build()

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
			return true
		},
		OnFetchEnd: func() {
			encoder.Close()
		},
	}

	if _, err := do.Query(queryCtx, "select * from "+cmd.Table); err != nil {
		ctx.Println("ERR", err.Error())
	}
}
