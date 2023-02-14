package shell

import (
	"time"

	"github.com/chzyer/readline"
	"github.com/machbase/neo-shell/codec"
	"github.com/machbase/neo-shell/do"
	"github.com/machbase/neo-shell/sink"
	"github.com/machbase/neo-shell/util"
	spi "github.com/machbase/neo-spi"
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
    --format,-f <format> output format [csv,json] (default:'csv')
    --[no-]header        export header (default:false)
    --delimiter,-d      csv delimiter (default:',')
    --tz                timezone for handling datetime
    --timeformat,-t     time format [ns|ms|s|<timeformat>] (default:'ns')
       ns, us, ms, s
         represents unix epoch time in nano-, micro-, milli- and seconds for each
       timeformat
         consult "help timeformat"
    --precision,-p <int>  set precision of float value to force round`

type ExportCmd struct {
	Table        string         `arg:"" name:"table"`
	Output       string         `name:"output" short:"o" default:"-"`
	Heading      bool           `name:"heading" negatable:""`
	TimeLocation *time.Location `name:"tz" default:"UTC"`
	Format       string         `name:"format" short:"f" default:"csv" enum:"box,csv,json"`
	Delimiter    string         `name:"delimiter" short:"d" default:","`
	TimeFormat   string         `name:"timeformat" short:"t" default:"ns"`
	Precision    int            `name:"precision" short:"p" default:"-1"`
	Help         bool           `kong:"-"`
}

func pcExport(cli Client) readline.PrefixCompleterInterface {
	return readline.PcItem("export")
}

func doExport(cli Client, cmdLine string) {
	cmd := &ExportCmd{}
	parser, err := Kong(cmd, func() error { cli.Println(helpExport); cmd.Help = true; return nil })
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	_, err = parser.Parse(splitFields(cmdLine, false))
	if cmd.Help {
		return
	}
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}

	var outputPath = util.StripQuote(cmd.Output)
	sink, err := sink.MakeSink(outputPath)
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}

	encoder := codec.NewEncoderBuilder().
		SetSink(sink).
		SetTimeLocation(cmd.TimeLocation).
		SetTimeFormat(cmd.TimeFormat).
		SetPrecision(cmd.Precision).
		SetRownum(false).
		SetHeading(cmd.Heading).
		SetBoxStyle("light").
		SetBoxSeparateColumns(true).
		SetBoxDrawBorder(true).
		SetCsvDelimieter(cmd.Delimiter).
		Build(cmd.Format)

	queryCtx := &do.QueryContext{
		DB: cli.Database(),
		OnFetchStart: func(cols spi.Columns) {
			encoder.Open(cols)
		},
		OnFetch: func(nrow int64, values []any) bool {
			err := encoder.AddRow(values)
			if err != nil {
				cli.Println("ERR", err.Error())
			}
			return true
		},
		OnFetchEnd: func() {
			encoder.Close()
			sink.Close()
		},
	}

	err = do.Query(queryCtx, "select * from "+cmd.Table+" order by time")
	if err != nil {
		cli.Println("ERR", err.Error())
	}
}
