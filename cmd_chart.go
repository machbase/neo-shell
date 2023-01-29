package shell

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/chzyer/readline"
	"github.com/machbase/neo-shell/api"
	"github.com/machbase/neo-shell/internal/ser_chartjs"
	"github.com/machbase/neo-shell/internal/ser_termchart"
	"github.com/robfig/cron"
)

func init() {
	RegisterCmd(&Cmd{
		Name:   "chart",
		PcFunc: pcChart,
		Action: doChart,
		Desc:   "chart [options] <tag_path> ...",
		Usage:  helpChart,
	})
}

const helpChart = `  chart [options] <tag_path>...
  arguments:
    tag_path ...   tag path as <table>/<tag>#<column>. ex) mytable/sensor.tag1#column
                   since all tag tables have 'value' column,
                   '#<column>' part can be omitted for default '#value' ex) mytable/sensor
  options:
    --tz                     timezone for handling datetime
    --time  <time>           base time, now or time string in format "2023-02-03 13:20:30" (default: now)
    --range <duration>       time range of data, from time specified by '--time' (default: 1m)
    --refresh,-r <duration>  refresh period (default: 0)
                             effective only if '--time' is "now".
                             value format is '[0-9]+(s|m)'  ex) '3s' for 3 seconds, '1m' for 1 minute
                             auto refresh is disabled if value is 0 which is default
    --count,-n <count>       repeat times (default: 0)
                             set 0 for unlimit
    --output,-o <file>       output file (default:'-' stdout)
    --format,-f <format>     output format
        none     terminal chart (default)
        json     json format
        html     generate chart page in html format
    --html-title <title>     title text for html output (default:"Chart")
    --html-subtitle <title>  sub title text for html output (default:"")`

type ChartCmd struct {
	TagPaths     []string       `arg:"" name:"tags"`
	TimeLocation *time.Location `name:"tz" default:"UTC"`
	Range        time.Duration  `name:"range" default:"1m"`
	Timestamp    string         `name:"time" default:"now"`
	Refresh      time.Duration  `name:"refresh" short:"r" default:"0"`
	Count        int            `name:"count" short:"n" default:"0"`
	Output       string         `name:"output" short:"o" default:"-"`
	Format       string         `name:"format" short:"f" enum:"none,json,html" default:"none"`
	HtmlTitle    string         `name:"html-title" default:"Chart"`
	HtmlSubtitle string         `name:"html-subtitle" default:""`
	HtmlWidth    string         `name:"html-width" default:"1600"`
	HtmlHeight   string         `name:"html-height" default:"900"`
	Help         bool           `kong:"-"`
}

func pcChart(cli Client) readline.PrefixCompleterInterface {
	return readline.PcItem("chart")
}

func doChart(cli Client, line string) {
	cmd := &ChartCmd{}
	parser, err := Kong(cmd, func() error { cli.Println(helpSql); cmd.Help = true; return nil })
	if err != nil {
		cli.Println(err.Error())
		return
	}
	_, err = parser.Parse(splitFields(line, true))
	if cmd.Help {
		return
	}
	if err != nil {
		cli.Println(err.Error())
		return
	}

	if len(cmd.TagPaths) == 0 {
		cli.Println("at least one tag_path should be specified")
		return
	}

	if len(cmd.Timestamp) == 0 {
		cmd.Timestamp = "now"
	}

	queries, err := buildDataQueries(cmd.TagPaths, cmd.Timestamp, cmd.Range, cmd.TimeLocation)
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}

	openWriter := func() (io.Writer, func(), error) {
		var writer io.Writer
		var closer func()
		switch cmd.Output {
		case "-":
			buf := bufio.NewWriter(cli.Stdout())
			closer = func() {
				buf.Flush()
			}
			writer = buf
		default:
			f, err := os.OpenFile(cmd.Output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
			if err != nil {
				cli.Println("ERR", err.Error())
				return nil, nil, err
			}
			buf := bufio.NewWriter(f)
			closer = func() {
				buf.Flush()
				f.Close()
			}
			writer = buf
		}
		return writer, closer, nil
	}

	var renderer api.SeriesRenderer
	switch cmd.Format {
	default:
		renderer = &ser_termchart.Renderer{}
		// termdash는 항상 tty를 사용해야하므로
		// 별도의 output 설정이 의미 없음.
		openWriter = nil
		// termdash의 경우 refresh cycle이 cmd.Count에 도달하여
		// 외부에서 close하는 경우 정상적으로 화면이 복구 되지 않는 문제가 있어
		// Count를 무조건 0 (무한 루프)으로 강제 설정한다.
		cmd.Count = 0
	case "json":
		renderer = &ser_chartjs.JsonRenderer{}
	case "html":
		renderer = &ser_chartjs.HtmlRenderer{
			Options: ser_chartjs.HtmlOptions{
				Title:    cmd.HtmlTitle,
				Subtitle: cmd.HtmlSubtitle,
				Width:    cmd.HtmlWidth,
				Height:   cmd.HtmlHeight,
			},
		}
	}

	var scheduler *cron.Cron
	var quitCh = make(chan bool, 1)
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	runCount := 0
	runCanceled := false
	runner := func() {
		var writer io.Writer
		var closer func()
		var closeOnce sync.Once
		if openWriter != nil {
			writer, closer, err = openWriter()
			if err != nil {
				cli.Println("ERR", err.Error())
				return
			}
			defer closeOnce.Do(closer)
		}

		db := cli.Database()
		tz := cmd.TimeLocation
		series := []*api.SeriesData{}
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
			labels := make([]string, 0)
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
				labels = append(labels, label)
				idx++
			}
			series = append(series, &api.SeriesData{
				Name:   dq.label,
				Values: values,
				Labels: labels,
			})
		}
		runCount++

		if err = renderer.Render(ctx, writer, series); err != nil {
			runCanceled = true
			if err != nil && err != api.ErrUserCancel {
				cli.Println("ERR", err.Error())
			}
		}
		if closer != nil {
			closeOnce.Do(closer)
		}
		if runCanceled || cmd.Count > 0 && cmd.Count <= runCount {
			quitCh <- true
		}
	}

	// run first round
	runner()
	// repeat ?
	if cmd.Count != 1 && !runCanceled {
		scheduler = cron.New()
		go func() {
			<-quitCh
			scheduler.Stop()
			cancel()
		}()

		if err := scheduler.AddFunc(fmt.Sprintf("@every %s", cmd.Refresh.String()), runner); err != nil {
			fmt.Println(err.Error())
			return
		}
		scheduler.Run()
	}
}

type DataQuery struct {
	table     string
	tag       string
	field     string
	rangeFunc func() (time.Time, time.Time)
	label     string
}

func buildDataQueries(tagPaths []string, cmdTimestamp string, cmdRange time.Duration, tz *time.Location) ([]*DataQuery, error) {
	queries := make([]*DataQuery, len(tagPaths))
	for i, path := range tagPaths {
		// path는 <table>/<tag>#<column> 형식으로 구성된다.
		toks := strings.SplitN(path, "/", 2)
		if len(toks) == 2 {
			queries[i] = &DataQuery{}
			queries[i].table = toks[0]
		} else {
			return nil, fmt.Errorf("table name not found in '%s'", path)
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
			var err error
			if cmdTimestamp == "now" {
				timestamp = time.Now()
			} else {
				timeformat := "2006-01-02 15:04:05"
				timestamp, err = time.ParseInLocation(timeformat, cmdTimestamp, tz)
				timestamp = timestamp.UTC()
				if err != nil {
					fmt.Println(err.Error())
				}
			}
			return timestamp.Add(-1 * cmdRange), timestamp
		}
	}
	return queries, nil
}
