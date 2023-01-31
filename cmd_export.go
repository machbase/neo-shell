package shell

import (
	"time"

	"github.com/chzyer/readline"
	"github.com/machbase/neo-shell/api"
	"github.com/machbase/neo-shell/internal/out_csv"
	"github.com/machbase/neo-shell/internal/sink_file"
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
	Format       string         `name:"format" short:"f" default:"csv"`
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

	db := cli.Database()
	rows, err := db.Query("select * from " + cmd.Table + " order by time")
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	defer rows.Close()

	sink, err := sink_file.New(cmd.Output)
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}

	var renderer api.RowsRenderer
	var renderCtx = &api.RowsContext{
		Sink:         sink,
		TimeLocation: cmd.TimeLocation,
		TimeFormat:   GetTimeformat(cmd.TimeFormat),
		Precision:    cmd.Precision,
		Rownum:       false,
		Heading:      cmd.Heading,
	}

	switch cmd.Format {
	case "csv":
		exporter := &out_csv.Exporter{}
		exporter.SetDelimiter(cmd.Delimiter)
		renderer = exporter
	}
	if renderer == nil {
		return
	}
	cc := cli.(*client)
	if err := cc.exportRows(renderCtx, rows, renderer, false); err != nil {
		cli.Println("ERR", err.Error())
	}
}
