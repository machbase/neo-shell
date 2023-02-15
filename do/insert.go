package do

import (
	"fmt"
	"strings"

	spi "github.com/machbase/neo-spi"
)

func Insert(db spi.Database, tableName string, columns []string, rows [][]any) spi.Result {
	vf := make([]string, len(columns))
	for i := range vf {
		vf[i] = "?"
	}
	valuesPlaces := strings.Join(vf, ",")
	columnsPhrase := strings.Join(columns, ",")

	sqlText := fmt.Sprintf("insert into %s (%s) values(%s)", tableName, columnsPhrase, valuesPlaces)
	var nrows int64
	for _, rec := range rows {
		result := db.Exec(sqlText, rec...)
		if result.Err() != nil {
			return &InsertResult{
				err:          result.Err(),
				rowsAffected: nrows,
				message:      "batch inserts aborted by error",
			}
		}
		nrows++
	}
	return &InsertResult{
		rowsAffected: nrows,
		message:      fmt.Sprintf("%d rows inserted", nrows),
	}
}

// implements spi.Result
type InsertResult struct {
	err          error
	rowsAffected int64
	message      string
}

func (ir *InsertResult) Err() error {
	return ir.err
}

func (ir *InsertResult) RowsAffected() int64 {
	return ir.rowsAffected
}

func (ir *InsertResult) Message() string {
	return ir.message
}
