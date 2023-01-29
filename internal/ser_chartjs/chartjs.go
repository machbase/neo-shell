package ser_chartjs

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"

	api "github.com/machbase/neo-shell/api"
)

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

func convertChartJsModel(data []*api.SeriesData) (*ChartJsModel, error) {
	cm := &ChartJsModel{}
	cm.Type = "line"
	cm.Data = ChartJsData{}
	cm.Data.Labels = data[0].Labels
	cm.Data.Datasets = []ChartJsDataset{}
	for _, series := range data {
		cm.Data.Datasets = append(cm.Data.Datasets, ChartJsDataset{
			Label:       series.Name,
			Data:        series.Values,
			BorderWidth: 1,
		})
	}
	cm.Options = ChartJsOptions{}
	cm.Options.Scales = ChartJsScalesOption{
		Y: ChartJsScale{BeginAtZero: false},
	}
	return cm, nil
}

///////////////////////////////////////////////
// JSON Renderer

type JsonRenderer struct {
}

func (r *JsonRenderer) Render(ctx context.Context, writer io.Writer, data []*api.SeriesData) error {
	model, err := convertChartJsModel(data)
	if err != nil {
		return err
	}
	buf, err := json.Marshal(model)
	if err != nil {
		return err
	}
	writer.Write(buf)
	return nil
}

///////////////////////////////////////////////
// HTML Renderer

//go:embed chartjs.html
var chartHtmlTemplate string

type ChartHtmlVars struct {
	HtmlOptions
	ChartData template.JS
}

type HtmlOptions struct {
	Title    string
	Subtitle string
	Width    string
	Height   string
}

type HtmlRenderer struct {
	Options HtmlOptions
}

func (r *HtmlRenderer) Render(ctx context.Context, writer io.Writer, data []*api.SeriesData) error {
	tmpl, err := template.New("chart_template").Parse(chartHtmlTemplate)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

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
	vars := &ChartHtmlVars{HtmlOptions: HtmlOptions(r.Options)}
	vars.ChartData = template.JS(string(dataJson))
	err = tmpl.Execute(buff, vars)
	if err != nil {
		fmt.Println(err.Error())
		return err
	}

	writer.Write(buff.Bytes())
	return nil
}
