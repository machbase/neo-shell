package shell

import (
	"context"
	"fmt"
	"time"

	"github.com/chzyer/readline"
	"github.com/machbase/neo-shell/renderer"
	"github.com/machbase/neo-shell/sink/filesink"
	spi "github.com/machbase/neo-spi"
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
    --timeformat,-t      time format [ns|ms|s|<timeformat>] (default:'default')
      ns, us, ms, s
        represents unix epoch time in nano-, micro-, milli- and seconds for each
      timeformat
        consult "help timeformat"    
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
    --title <title>          title text for html output (default:"Chart")
    --subtitle <title>       sub title text for html output (default:"")
	--width <string>         chart width for html output (default:"1600")
	--height <string>        chart height (default:"900")`

type ChartCmd struct {
	TagPaths     []string       `arg:"" name:"tags"`
	TimeLocation *time.Location `name:"tz" default:"UTC"`
	Timeformat   string         `name:"timeformat" default:"default"`
	Range        time.Duration  `name:"range" default:"1m"`
	Timestamp    string         `name:"time" default:"now"`
	Refresh      time.Duration  `name:"refresh" short:"r" default:"0"`
	Count        int            `name:"count" short:"n" default:"0"`
	Output       string         `name:"output" short:"o" default:"-"`
	Format       string         `name:"format" short:"f" enum:"none,json,html" default:"none"`
	HtmlTitle    string         `name:"title" default:"Chart"`
	HtmlSubtitle string         `name:"subtitle" default:""`
	HtmlWidth    string         `name:"width" default:"1600"`
	HtmlHeight   string         `name:"height" default:"900"`
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

	queries, err := renderer.BuildChartQueries(cmd.TagPaths, cmd.Timestamp, cmd.Range, cmd.Timeformat, cmd.TimeLocation)
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}

	render := renderer.NewChartRendererBuilder().
		SetTitle(cmd.HtmlTitle).
		SetSubtitle(cmd.HtmlSubtitle).
		SetSize(cmd.HtmlWidth, cmd.HtmlHeight).
		Build(cmd.Format)

	switch cmd.Format {
	default:
		// termdash의 경우 refresh cycle이 cmd.Count에 도달하여
		// 외부에서 close하는 경우 정상적으로 화면이 복구 되지 않는 문제가 있어
		// Count를 무조건 0 (무한 루프)으로 강제 설정한다.
		cmd.Count = 0
	case "json":
	case "html":
	}

	var scheduler *cron.Cron
	var quitCh = make(chan bool, 1)
	var ctx, cancel = context.WithCancel(context.Background())
	defer cancel()

	runCount := 0
	runCanceled := false
	runner := func() {
		sink, err := filesink.New(cmd.Output)
		if err != nil {
			cli.Println("ERR", err.Error())
			return
		}
		defer sink.Close()

		db := cli.Database()
		series := []*spi.RenderingData{}
		// query
		for _, dq := range queries {
			data, err := dq.Query(db)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			series = append(series, data)
		}
		runCount++

		if err = render.Render(ctx, sink, series); err != nil {
			runCanceled = true
			if err != nil && err != spi.ErrUserCancel {
				cli.Println("ERR", err.Error())
			}
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
