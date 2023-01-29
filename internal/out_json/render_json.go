package out_json

import (
	"encoding/json"
	"fmt"

	"github.com/machbase/neo-shell/api"
)

type Exporter struct {
	api.RowsRenderer

	nrow  int
	ctx   *api.RowsContext
	flush func() error
}

func (ex *Exporter) OpenRender(ctx *api.RowsContext) error {
	ex.ctx = ctx
	ex.flush = ctx.Writer.Flush

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
	ex.ctx.Writer.Write([]byte(header))

	return nil
}

func (ex *Exporter) CloseRender() {
	footer := "]}}\n"
	ex.ctx.Writer.Write([]byte(footer))
	ex.flush()
}

func (ex *Exporter) PageFlush(heading bool) {
	ex.flush()
}

func (ex *Exporter) RenderRow(values []any) error {
	ex.nrow++
	var recJson []byte
	if ex.ctx.Rownum {
		vs := append([]any{ex.nrow}, values...)
		recJson, _ = json.Marshal(vs)
	} else {
		recJson, _ = json.Marshal(values)
	}

	if ex.nrow > 1 {
		ex.ctx.Writer.Write([]byte(","))
	}
	ex.ctx.Writer.Write(recJson)

	return nil
}
