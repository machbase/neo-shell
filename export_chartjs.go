package shell

import (
	"encoding/json"
	"fmt"
	"io"

	"github.com/machbase/neo-grpc/machrpc"
)

func (cli *client) exportRowsChartJs(writer io.Writer, rows *machrpc.Rows, cmd *SqlCmd) {
	cols, err := rows.Columns()
	if err != nil {
		cli.Println("ERR", err.Error())
		return
	}
	names := cli.columnNames(cols, cmd.Rownum)
	columnsJson, _ := json.Marshal(names)
	types := cli.columnTypes(cols, cmd.Rownum)
	typesJson, _ := json.Marshal(types)

	header := fmt.Sprintf(
		`{"data":{"columns":%s,"types":%s,"rows":[`,
		string(columnsJson), string(typesJson))
	footer := `]}}`

	writer.Write([]byte(header))
	defer writer.Write([]byte(footer))

	buf := makeBuffer(cols)
	nrow := 0
	for {
		if !rows.Next() {
			return
		}
		err := rows.Scan(buf...)
		if err != nil {
			cli.Println("ERR", err.Error())
			return
		}
		nrow++

		var recJson []byte
		if cmd.Rownum {
			vs := append([]any{nrow}, buf...)
			recJson, _ = json.Marshal(vs)
		} else {
			recJson, _ = json.Marshal(buf)
		}

		if nrow > 1 {
			writer.Write([]byte(","))
		}
		writer.Write(recJson)
	}
}
