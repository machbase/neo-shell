package shell

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/alecthomas/kong"
	"github.com/chzyer/readline"
	"github.com/mum4k/termdash"
	"github.com/mum4k/termdash/cell"
	"github.com/mum4k/termdash/container"
	"github.com/mum4k/termdash/keyboard"
	"github.com/mum4k/termdash/linestyle"
	"github.com/mum4k/termdash/terminal/tcell"
	"github.com/mum4k/termdash/terminal/terminalapi"
	"github.com/mum4k/termdash/widgets/linechart"
	"github.com/robfig/cron"
)

type ChartCmd struct {
	TableName string        `arg:"" name:"table" help:"table name"`
	Tags      []string      `arg:"" name:"tags" help:"tags with column name. ex) tag1 tag2.columnA tag3.columnB"`
	Range     time.Duration `name:"range" default:"5m" help:"time range of data, from now() - range to now()"`
	Timestamp string        `name:"time" default:"now" help:"time ex) now or \"2023-02-03 13:20:30\""`
	Refresh   time.Duration `name:"refresh" short:"r" default:"3s" help:"refresh period, effective only if time is \"now\""`
}

func (cli *client) pcChart() *readline.PrefixCompleter {
	return readline.PcItem("chart")
}

func (cli *client) doChart(args []string) {
	cmd := &ChartCmd{}
	parser, err := kong.New(cmd, kong.HelpOptions{Compact: true}, kong.Exit(func(int) {}))
	parser.Model.Name = "chart"
	if err != nil {
		cli.Writeln(err.Error())
		return
	}
	_, err = parser.Parse(args)
	if err != nil {
		cli.Writeln(err.Error())
		return
	}

	if len(cmd.TableName) == 0 {
		fmt.Println("no table is specified")
		return
	}
	if len(cmd.Tags) == 0 {
		fmt.Println("at least one tag should specified")
		return
	}

	if len(cmd.Timestamp) == 0 {
		cmd.Timestamp = "now"
	}

	var timestamp time.Time
	if cmd.Timestamp == "now" {
		timestamp = time.Now()
	} else {
		timeformat := "2006-01-02 15:04:05"
		if cli.conf.LocalTime {
			timestamp, err = time.ParseInLocation(timeformat, cmd.Timestamp, time.Local)
			timestamp = timestamp.UTC()
		} else {
			timestamp, err = time.Parse(timeformat, cmd.Timestamp)
		}
		if err != nil {
			fmt.Println(err.Error())
		}
	}

	queries := make([]*DataQuery, len(cmd.Tags))
	for i := range cmd.Tags {
		queries[i] = &DataQuery{}
		queries[i].table = cmd.TableName
		queries[i].rangeTo = timestamp
		queries[i].rangeFrom = timestamp.Add(-1 * cmd.Range)

		toks := strings.SplitN(cmd.Tags[i], ".", 2)
		if len(toks) == 2 {
			queries[i].tag = toks[0]
			queries[i].field = toks[1]
		} else {
			queries[i].tag = toks[0]
			queries[i].field = "VALUE"
		}
	}

	// context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// make terminal interface
	term, err := tcell.New()
	if err != nil {
		cli.Writeln(err.Error())
		return
	}
	defer term.Close()

	// line chart
	lchart, err := linechart.New(
		linechart.AxesCellOpts(cell.FgColor(cell.ColorRed)),
		linechart.YLabelCellOpts(cell.FgColor(cell.ColorGreen)),
		linechart.XLabelCellOpts(cell.FgColor(cell.ColorCyan)),
	)
	if err != nil {
		cli.Writeln(err.Error())
		return
	}

	// terminal container
	cont, err := container.New(
		term,
		container.Border(linestyle.Light),
		container.BorderTitle("ESC to quit"),
		container.PlaceWidget(lchart),
	)
	if err != nil {
		cli.Writeln(err.Error())
		return
	}

	// feed receiver
	runner := func() {
		// query
		for _, dq := range queries {
			rows, err := cli.db.Query(dq.makeQuery())
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			defer rows.Close()

			times := make([]string, 0)
			values := make([]float64, 0)

			for rows.Next() {
				var ts time.Time
				var value float64
				err = rows.Scan(&ts, &value)
				if err != nil {
					fmt.Println(err.Error())
					return
				}
				if cli.conf.LocalTime {
					ts = ts.Local()
				}
				times = append(times, ts.Format("15:04:05"))
				values = append(values, value)
			}

			xlabels := make(map[int]string)
			for i, s := range times {
				xlabels[i] = s
			}
			err = lchart.Series(
				dq.label,
				values,
				linechart.SeriesCellOpts(cell.FgColor(cell.ColorNumber(33))),
				linechart.SeriesXLabels(xlabels),
			)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
		}
	}

	var scheduler *cron.Cron
	if cmd.Timestamp == "now" {
		scheduler = cron.New()
		scheduler.AddFunc(fmt.Sprintf("@every %s", cmd.Refresh.String()), runner)
		go scheduler.Run()
	} else {
		runner()
	}

	quitter := func(k *terminalapi.Keyboard) {
		if k.Key == 'q' || k.Key == 'Q' || k.Key == keyboard.KeyEsc {
			// stop scheduler
			if scheduler != nil {
				scheduler.Stop()
			}
			// stop ui
			cancel()
		}
	}

	termOpts := []termdash.Option{
		termdash.KeyboardSubscriber(quitter),
		termdash.RedrawInterval(cmd.Refresh),
	}
	if err := termdash.Run(ctx, term, cont, termOpts...); err != nil {
		cli.Writeln(err.Error())
		return
	}
}

type DataFeed struct {
	Series []*DataSeries
}

type DataSeries struct {
	Legend string
	Values []float64
	Labels map[int]string
}

type DataFeeder interface {
	Start() error
	Stop()
	Query(string) error
}

type DataQuery struct {
	table     string
	tag       string
	field     string
	rangeFrom time.Time
	rangeTo   time.Time
	label     string
}

func (dq *DataQuery) makeQuery() string {
	if strings.ToUpper(dq.field) == "VALUE" {
		dq.label = strings.ToLower(dq.tag)
	} else {
		dq.label = strings.ToLower(fmt.Sprintf("%s-%s", dq.tag, dq.field))
	}
	return fmt.Sprintf(`select TIME, %s from %s where TIME between %d and %d AND name = '%s'`,
		dq.field, dq.table, dq.rangeFrom.UnixNano(), dq.rangeTo.UnixNano(), dq.tag)
}
