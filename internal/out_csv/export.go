package out_csv

import (
	"encoding/csv"
	"fmt"
	"strconv"
	"time"
	"unicode/utf8"

	"github.com/machbase/neo-shell/api"
)

type Exporter struct {
	rownum int64

	writer *csv.Writer
	Comma  rune

	ctx *api.RowsContext
}

func (ex *Exporter) SetDelimiter(delimiter string) {
	delmiter, _ := utf8.DecodeRuneInString(delimiter)
	ex.Comma = delmiter
}

func (ex *Exporter) OpenRender(ctx *api.RowsContext) error {
	ex.ctx = ctx
	ex.writer = csv.NewWriter(ctx.Sink)

	if ex.Comma != 0 {
		ex.writer.Comma = ex.Comma
	}

	if ctx.Heading {
		// TODO check if write() returns error, when csvWritter.Comma is not valid
		if ctx.Rownum {
			ex.writer.Write(append([]string{"#"}, ctx.ColumnNames...))
		} else {
			ex.writer.Write(ctx.ColumnNames)
		}
	}

	return nil
}

func (ex *Exporter) CloseRender() {
	ex.writer.Flush()
	ex.ctx.Sink.Close()
}

func (ex *Exporter) PageFlush(heading bool) {
	ex.writer.Flush()
	ex.ctx.Sink.Flush()
}

func (ex *Exporter) RenderRow(values []any) error {
	var cols = make([]string, len(values))

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
		return ex.writer.Write(append([]string{strconv.FormatInt(ex.rownum, 10)}, cols...))
	} else {
		return ex.writer.Write(cols)
	}
}