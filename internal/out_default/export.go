package out_default

import (
	"fmt"
	"strconv"
	"time"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/machbase/neo-shell/api"
)

type Exporter struct {
	api.RowsRenderer
	writer table.Writer
	flush  func() error
	rownum int64
	ctx    *api.RowsContext

	Style           string
	SeparateColumns bool
	DrawBorder      bool
}

func (ex *Exporter) OpenRender(ctx *api.RowsContext) error {
	ex.ctx = ctx
	ex.writer = table.NewWriter()
	ex.writer.SetOutputMirror(ctx.Writer)
	//if bw, ok := ctx.Writer.(*bufio.Writer); ok {
	ex.flush = ctx.Writer.Flush
	//}

	style := table.StyleDefault
	switch ex.Style {
	case "bold":
		style = table.StyleBold
	case "double":
		style = table.StyleDouble
	case "light":
		style = table.StyleLight
	case "round":
		style = table.StyleRounded
	}
	style.Options.SeparateColumns = ex.SeparateColumns
	style.Options.DrawBorder = ex.DrawBorder

	ex.writer.SetStyle(style)

	if ctx.Heading {
		vs := make([]any, len(ctx.ColumnNames))
		for i, h := range ctx.ColumnNames {
			vs[i] = h
		}
		if ex.ctx.Rownum {
			ex.writer.AppendHeader(table.Row(append([]any{"#"}, vs...)))
		} else {
			ex.writer.AppendHeader(table.Row(vs))
		}
	}

	return nil
}

func (ex *Exporter) CloseRender() {
	if ex.writer.Length() > 0 {
		ex.writer.Render()
		ex.writer.ResetRows()
	}
}

func (ex *Exporter) PageFlush(heading bool) {
	ex.writer.Render()
	ex.flush()

	ex.writer.ResetRows()
	if !heading {
		ex.writer.ResetHeaders()
	}
}

func (ex *Exporter) RenderRow(values []any) error {
	var cols = make([]any, len(values))

	for i, r := range values {
		if r == nil {
			cols[i] = "NULL"
			continue
		}
		switch v := r.(type) {
		case *string:
			cols[i] = *v
		case string:
			cols[i] = v
		case *time.Time:
			switch ex.ctx.TimeFormat {
			case "ns":
				cols[i] = strconv.FormatInt(v.UnixNano(), 10)
			case "ms":
				cols[i] = strconv.FormatInt(v.UnixMilli(), 10)
			case "us":
				cols[i] = strconv.FormatInt(v.UnixMicro(), 10)
			case "s":
				cols[i] = strconv.FormatInt(v.Unix(), 10)
			default:
				if ex.ctx.TimeLocation == nil {
					ex.ctx.TimeLocation = time.UTC
				}
				cols[i] = v.In(ex.ctx.TimeLocation).Format(ex.ctx.TimeFormat)
			}
		case *float64:
			if ex.ctx.Precision < 0 {
				cols[i] = fmt.Sprintf("%f", *v)
			} else {
				cols[i] = fmt.Sprintf("%.*f", ex.ctx.Precision, *v)
			}
		case *int:
			cols[i] = strconv.FormatInt(int64(*v), 10)
		case int:
			cols[i] = strconv.FormatInt(int64(v), 10)
		case *int32:
			cols[i] = strconv.FormatInt(int64(*v), 10)
		case int32:
			cols[i] = strconv.FormatInt(int64(v), 10)
		case *int64:
			cols[i] = strconv.FormatInt(*v, 10)
		case int64:
			cols[i] = strconv.FormatInt(v, 10)
		default:
			cols[i] = fmt.Sprintf("%T", r)
		}
	}

	ex.rownum++

	if ex.ctx.Rownum {
		ex.writer.AppendRow(table.Row(append([]any{strconv.FormatInt(ex.rownum, 10)}, cols...)))
	} else {
		ex.writer.AppendRow(table.Row(cols))
	}

	return nil
}
