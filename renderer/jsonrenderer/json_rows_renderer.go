package jsonrenderer

import (
	"encoding/json"
	"fmt"
	"time"

	spi "github.com/machbase/neo-spi"
)

type Exporter struct {
	nrow int
	ctx  *spi.RowsRendererContext
}

func NewRowsRenderer() spi.RowsRenderer {
	return &Exporter{}
}

func (ex *Exporter) OpenRender(ctx *spi.RowsRendererContext) error {
	ex.ctx = ctx

	names := ctx.ColumnNames
	types := ctx.ColumnTypes
	if ctx.Rownum {
		names = append([]string{"ROWNUM"}, names...)
		types = append([]string{"string"}, types...)
	}

	columnsJson, _ := json.Marshal(names)
	typesJson, _ := json.Marshal(types)

	header := fmt.Sprintf(`{"data":{"columns":%s,"types":%s,"rows":[`,
		string(columnsJson), string(typesJson))
	ex.ctx.Sink.Write([]byte(header))

	return nil
}

func (ex *Exporter) CloseRender() {
	footer := "]}}\n"
	ex.ctx.Sink.Write([]byte(footer))
	ex.ctx.Sink.Close()
}

func (ex *Exporter) PageFlush(heading bool) {
	ex.ctx.Sink.Flush()
}

func (ex *Exporter) RenderRow(source []any) error {
	ex.nrow++

	if ex.ctx.TimeLocation == nil {
		ex.ctx.TimeLocation = time.UTC
	}

	values := make([]any, len(source))
	for i, field := range source {
		values[i] = field
		if v, ok := field.(*time.Time); ok {
			switch ex.ctx.TimeFormat {
			case "ns":
				values[i] = v.UnixNano()
			case "ms":
				values[i] = v.UnixMilli()
			case "us":
				values[i] = v.UnixMicro()
			case "s":
				values[i] = v.Unix()
			default:
				if ex.ctx.TimeLocation == nil {
					ex.ctx.TimeLocation = time.UTC
				}
				values[i] = v.In(ex.ctx.TimeLocation).Format(ex.ctx.TimeFormat)
			}
			continue
		}
	}
	var recJson []byte
	if ex.ctx.Rownum {
		vs := append([]any{ex.nrow}, values...)
		recJson, _ = json.Marshal(vs)
	} else {
		recJson, _ = json.Marshal(values)
	}

	if ex.nrow > 1 {
		ex.ctx.Sink.Write([]byte(","))
	}
	ex.ctx.Sink.Write(recJson)

	return nil
}
