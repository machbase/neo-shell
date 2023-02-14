package renderer

import (
	"github.com/machbase/neo-shell/renderer/jschart"
	"github.com/machbase/neo-shell/renderer/termchart"
	spi "github.com/machbase/neo-spi"
)

type ChartRendererBuilder interface {
	Build() spi.Renderer
	SetTitle(string) ChartRendererBuilder
	SetSubtitle(string) ChartRendererBuilder
	SetSize(width, height string) ChartRendererBuilder
}

type chartbuilder struct {
	chartType string
	title     string
	subtitle  string
	width     string
	height    string
}

func NewChartRendererBuilder(format string) ChartRendererBuilder {
	return &chartbuilder{chartType: format}
}

func (cb *chartbuilder) Build() spi.Renderer {
	switch cb.chartType {
	case "json":
		return jschart.NewJsonSeriesRenderer()
	case "html":
		return jschart.NewHtmlSeriesRenderer(
			jschart.HtmlOptions{
				Title:    cb.title,
				Subtitle: cb.subtitle,
				Width:    cb.width,
				Height:   cb.height,
			},
		)
	default: // "term"
		return termchart.NewSeriesRenderer()
	}
}

func (cb *chartbuilder) SetTitle(title string) ChartRendererBuilder {
	cb.title = title
	return cb
}

func (cb *chartbuilder) SetSubtitle(subtitle string) ChartRendererBuilder {
	cb.subtitle = subtitle
	return cb
}

func (cb *chartbuilder) SetSize(width, height string) ChartRendererBuilder {
	cb.width = width
	cb.height = height
	return cb
}
