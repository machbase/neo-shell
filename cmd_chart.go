package shell

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"os"
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
	TagPaths     []string      `arg:"" name:"tags"`
	Range        time.Duration `name:"range" default:"1m"`
	Timestamp    string        `name:"time" default:"now"`
	Refresh      time.Duration `name:"refresh" short:"r" default:"0"`
	Count        int           `name:"count" short:"n" default:"0"`
	Output       string        `name:"output" short:"o" default:"-"`
	Format       string        `name:"format" short:"f" enum:"none,json,html" default:"none"`
	HtmlTitle    string        `name:"html-title" default:"Chart"`
	HtmlSubtitle string        `name:"html-subtitle" default:""`
	HtmlWidth    string        `name:"html-width" default:"1600"`
	HtmlHeight   string        `name:"html-height" default:"900"`
}

func pcChart(cli Client) readline.PrefixCompleterInterface {
	return readline.PcItem("chart")
}

func doChart(cli Client, line string) {
	cmd := &ChartCmd{}
	parser, err := kong.New(cmd, kong.HelpOptions{Compact: true}, kong.Exit(func(int) {}),
		kong.Help(func(options kong.HelpOptions, ctx *kong.Context) error {
			cli.Println(helpChart)
			return nil
		}))
	if err != nil {
		cli.Println(err.Error())
		return
	}
	_, err = parser.Parse(splitFields(line, true))
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

	var writer io.Writer
	switch cmd.Output {
	case "-":
		buf := bufio.NewWriter(cli.Stdout())
		defer func() {
			buf.Flush()
		}()
		writer = buf
	default:
		f, err := os.OpenFile(cmd.Output, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			cli.Println("ERR", err.Error())
			return
		}
		buf := bufio.NewWriter(f)
		defer func() {
			buf.Flush()
			f.Close()
		}()
		writer = buf
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

	dataCh := make(chan *DataSeries, 1)

	runCount := 0
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

			dataCh <- &DataSeries{
				Name:   dq.label,
				Values: values,
				Labels: labels,
			}
			runCount++

			if cmd.Count > 0 && cmd.Count <= runCount {
				close(dataCh)
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

		defer func() {
			// stop scheduler
			if scheduler != nil {
				scheduler.Stop()
			}
		}()
	} else {
		runner()
	}

	switch cmd.Format {
	default: /*none*/
		if err := chartTerm(dataCh, cmd.Refresh); err != nil {
			cli.Println("ERR", err.Error())
		}
	case "json":
		if err := chartJson(writer, dataCh); err != nil {
			cli.Println("ERR", err.Error())
		}
	case "html":
		opt := ChartHtmlOptions{
			Title:    cmd.HtmlTitle,
			Subtitle: cmd.HtmlSubtitle,
			Width:    cmd.HtmlWidth,
			Height:   cmd.HtmlHeight,
		}
		if err := chartHtml(writer, dataCh, opt); err != nil {
			cli.Println("ERR", err.Error())
		}
	}
}

type DataQuery struct {
	table     string
	tag       string
	field     string
	rangeFunc func() (time.Time, time.Time)
	label     string
}

type DataSeries struct {
	Name   string
	Values []float64
	Labels []string
}

type ChartJsModel struct {
	Type    string         `json:"type"`
	Data    ChartJsData    `json:"data"`
	Options ChartJsOptions `json:"options"`
}

type ChartJsData struct {
	Labels   []string         `json:"labels"`
	Datasets []ChartJsDataset `json:"datasets"`
}

type ChartJsDataset struct {
	Label       string    `json:"label"`
	Data        []float64 `json:"data"`
	BorderWidth int       `json:"borderWidth"`
}

type ChartJsOptions struct {
	Scales ChartJsScalesOption `json:"scales"`
}

type ChartJsScalesOption struct {
	Y ChartJsScale `json:"y"`
}

type ChartJsScale struct {
	BeginAtZero bool `json:"beginAtZero"`
}

func convertChartJsModel(data *DataSeries) (*ChartJsModel, error) {
	cm := &ChartJsModel{}
	cm.Type = "line"
	cm.Data = ChartJsData{}
	cm.Data.Labels = data.Labels
	cm.Data.Datasets = []ChartJsDataset{
		{
			Label:       data.Name,
			Data:        data.Values,
			BorderWidth: 1,
		},
	}
	cm.Options = ChartJsOptions{}
	cm.Options.Scales = ChartJsScalesOption{
		Y: ChartJsScale{BeginAtZero: false},
	}
	return cm, nil
}

func chartJson(writer io.Writer, dataChan <-chan *DataSeries) error {
	for data := range dataChan {
		model, err := convertChartJsModel(data)
		if err != nil {
			return err
		}
		buf, err := json.Marshal(model)
		if err != nil {
			return err
		}
		writer.Write(buf)
	}
	return nil
}

//go:embed cmd_chart.html
var chartHtmlTemplate string

type ChartHtmlVars struct {
	ChartHtmlOptions
	ChartData template.JS
}

type ChartHtmlOptions struct {
	Title    string
	Subtitle string
	Width    string
	Height   string
}

func chartHtml(writer io.Writer, dataChan <-chan *DataSeries, opt ChartHtmlOptions) error {
	tmpl, err := template.New("chart_template").Parse(chartHtmlTemplate)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}
	for data := range dataChan {
		model, err := convertChartJsModel(data)
		if err != nil {
			return err
		}
		dataJson, err := json.Marshal(model)
		if err != nil {
			fmt.Println(err.Error())
			return err
		}

		buff := &bytes.Buffer{}
		vars := &ChartHtmlVars{ChartHtmlOptions: opt}
		vars.ChartData = template.JS(string(dataJson))
		err = tmpl.Execute(buff, vars)
		if err != nil {
			fmt.Println(err.Error())
			return err
		}

		writer.Write(buff.Bytes())
	}
	return nil
}

func chartTerm(dataChan <-chan *DataSeries, refresh time.Duration) error {
	// make terminal interface
	term, err := tcell.New()
	if err != nil {
		return err
	}
	defer term.Close()

	// line chart
	lchart, err := linechart.New(
		linechart.AxesCellOpts(cell.FgColor(cell.ColorRed)),
		linechart.YLabelCellOpts(cell.FgColor(cell.ColorGreen)),
		linechart.XLabelCellOpts(cell.FgColor(cell.ColorCyan)),
	)
	if err != nil {
		return err
	}

	// terminal container
	cont, err := container.New(
		term,
		container.Border(linestyle.Light),
		container.BorderTitle("ESC to quit"),
		container.PlaceWidget(lchart),
	)
	if err != nil {
		return err
	}
	// context
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go func() {
		for {
			select {
			case data := <-dataChan:
				if data == nil {
					cancel()
					return
				}
				xlabels := make(map[int]string)
				for i, n := range data.Labels {
					xlabels[i] = n
				}
				err = lchart.Series(
					data.Name,
					data.Values,
					linechart.SeriesCellOpts(cell.FgColor(cell.ColorNumber(33))),
					linechart.SeriesXLabels(xlabels),
				)
				if err != nil {
					fmt.Println(err.Error())
					return
				}
			case <-ctx.Done():
				cancel()
				return
			}
		}
	}()

	quitter := func(k *terminalapi.Keyboard) {
		if k.Key == keyboard.KeyEsc {
			// stop ui
			cancel()
		}
	}

	if refresh < time.Second {
		refresh = time.Second
	}
	termOpts := []termdash.Option{
		termdash.KeyboardSubscriber(quitter),
		termdash.RedrawInterval(refresh),
	}
	if err := termdash.Run(ctx, term, cont, termOpts...); err != nil {
		return err
	}
	return nil
}
