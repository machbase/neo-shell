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

func init() {
	RegisterCmd(&Cmd{
		Name:    "chart",
		Aliases: []string{},
		PcFunc:  pcChart,
		Action:  doChart,
		Desc:    "chart [options] <tag_path> ...",
		Usage: `  arguments:
    tags_path ...   tag path as <table>/<tag>#<column>. ex) mytable/sensor.tag1#column
                    since all tag tables have 'value' column,
                    '#<column>' part can be omitted for default '#value' ex) mytable/sensor
  options:
    --time  <time>           base time, now or time string in format "2023-02-03 13:20:30" (default: now)
    --range <duration>       time range of data, from time specified by '--time'
    --refresh,-r <duration>  refresh period, effective only if time is "now" (default: 1s)`,
	})
}

type ChartCmd struct {
	TagPaths  []string      `arg:"" name:"tags"`
	Range     time.Duration `name:"range" default:"5m"`
	Timestamp string        `name:"time" default:"now"`
	Refresh   time.Duration `name:"refresh" short:"r" default:"1s"`
}

func pcChart(cli Client) readline.PrefixCompleterInterface {
	return readline.PcItem("chart")
}

func doChart(cli Client, line string) {
	// cli := c.(*client)

	cmd := &ChartCmd{}
	parser, err := kong.New(cmd, kong.HelpOptions{Compact: true}, kong.Exit(func(int) {}))
	parser.Model.Name = "chart"
	if err != nil {
		cli.Println(err.Error())
		return
	}
	_, err = parser.Parse(splitFields(line))
	if err != nil {
		cli.Println(err.Error())
		return
	}

	if len(cmd.TagPaths) == 0 {
		cli.Println("at least one tag should be specified")
		return
	}

	if len(cmd.Timestamp) == 0 {
		cmd.Timestamp = "now"
	}

	queries := make([]*DataQuery, len(cmd.TagPaths))
	tz := cli.TimeLocation()
	for i, path := range cmd.TagPaths {
		// path는 <table>/<tag>#<column> 형식으로 구성된다.
		toks := strings.SplitN(path, "/", 2)
		if len(toks) == 2 {
			queries[i] = &DataQuery{}
			queries[i].table = toks[0]
		} else {
			cli.Printfln("table name not found in '%s'", path)
			return
		}
		toks = strings.SplitN(toks[1], "#", 2)
		if len(toks) == 2 {
			queries[i].tag = toks[0]
			queries[i].field = toks[1]
		} else {
			queries[i].tag = toks[0]
			queries[i].field = "VALUE"
		}

		queries[i].rangeFunc = func() (time.Time, time.Time) {
			var timestamp time.Time
			if cmd.Timestamp == "now" {
				timestamp = time.Now()
			} else {
				timeformat := "2006-01-02 15:04:05"
				timestamp, err = time.ParseInLocation(timeformat, cmd.Timestamp, tz)
				timestamp = timestamp.UTC()
				if err != nil {
					fmt.Println(err.Error())
				}
			}
			return timestamp.Add(-1 * cmd.Range), timestamp
		}
	}

	// context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// make terminal interface
	term, err := tcell.New()
	if err != nil {
		cli.Println(err.Error())
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
		cli.Println(err.Error())
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
		cli.Println(err.Error())
		return
	}

	runner := func() {
		db := cli.Database()
		tz := cli.TimeLocation()
		// query
		for _, dq := range queries {
			if strings.ToUpper(dq.field) == "VALUE" {
				dq.label = strings.ToLower(dq.tag)
			} else {
				dq.label = strings.ToLower(fmt.Sprintf("%s-%s", dq.tag, dq.field))
			}
			rangeFrom, rangeTo := dq.rangeFunc()

			lastSql := fmt.Sprintf(`select TIME, %s from %s where NAME = ? AND TIME between ? AND ? order by time`, dq.field, dq.table)

			rows, err := db.Query(lastSql, dq.tag, rangeFrom, rangeTo)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			defer rows.Close()

			values := make([]float64, 0)
			xlabels := make(map[int]string)

			idx := 0
			for rows.Next() {
				var ts time.Time
				var value float64
				err = rows.Scan(&ts, &value)
				if err != nil {
					fmt.Println(err.Error())
					return
				}
				label := ts.In(tz).Format("15:04:05")
				values = append(values, value)
				xlabels[idx] = label
				idx++
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
		err := scheduler.AddFunc(fmt.Sprintf("@every %s", cmd.Refresh.String()), runner)
		if err != nil {
			fmt.Println(err.Error())
		}
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
		cli.Println(err.Error())
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
	rangeFunc func() (time.Time, time.Time)
	label     string
}
