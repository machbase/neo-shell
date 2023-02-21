package with

import (
	"context"
	"database/sql"

	spi "github.com/machbase/neo-spi"
)

type drv struct {
	rdb *sql.DB
}

func New(rdb *sql.DB) spi.Database {
	return &drv{
		rdb: rdb,
	}
}

func (d *drv) GetServerInfo() (*spi.ServerInfo, error) {
	return nil, spi.ErrNotImplemented
}

func (d *drv) Explain(sqlText string) (string, error) {
	return "", spi.ErrNotImplemented
}

func (d *drv) Exec(sqlText string, params ...any) spi.Result {
	return d.ExecContext(context.Background(), sqlText, params...)
}

func (d *drv) ExecContext(ctx context.Context, sqlText string, params ...any) spi.Result {
	r, err := d.rdb.ExecContext(ctx, sqlText, params...)
	return &ResultWrap{
		r:   r,
		err: err,
	}
}

func (d *drv) Query(sqlText string, params ...any) (spi.Rows, error) {
	return d.QueryContext(context.Background(), sqlText, params...)
}

func (d *drv) QueryContext(ctx context.Context, sqlText string, params ...any) (spi.Rows, error) {
	rows, err := d.rdb.QueryContext(ctx, sqlText, params...)
	if err != nil {
		return nil, err
	}
	return &RowsWrap{r: rows}, nil
}

func (d *drv) QueryRow(sqlText string, params ...any) spi.Row {
	return d.QueryRowContext(context.Background(), sqlText, params...)
}

func (d *drv) QueryRowContext(ctx context.Context, sqlText string, params ...any) spi.Row {
	row := d.rdb.QueryRowContext(ctx, sqlText, params...)
	return &RowWrap{
		r: row,
	}
}

func (d *drv) Appender(tableName string) (spi.Appender, error) {
	return nil, spi.ErrNotImplemented
}

type ResultWrap struct {
	r   sql.Result
	err error
}

func (rw *ResultWrap) Err() error {
	return rw.err
}

func (rw *ResultWrap) LastInsertId() (int64, error) {
	return rw.r.LastInsertId()
}

func (rw *ResultWrap) RowsAffected() int64 {
	n, err := rw.r.RowsAffected()
	if err != nil {
		return 0
	}
	return n
}

func (rw *ResultWrap) Message() string {
	return ""
}

type RowWrap struct {
	r *sql.Row
}

func (rw *RowWrap) Success() bool {
	return true
}
func (rw *RowWrap) Err() error {
	return nil
}
func (rw *RowWrap) Scan(cols ...any) error {
	return nil
}
func (rw *RowWrap) Values() []any {
	return nil
}
func (rw *RowWrap) RowsAffected() int64 {
	return 0
}
func (rw *RowWrap) Message() string {
	return ""
}

type RowsWrap struct {
	r *sql.Rows
}

func (rw *RowsWrap) Columns() (spi.Columns, error) {
	typs, err := rw.r.ColumnTypes()
	if err != nil {
		return nil, err
	}
	result := []*spi.Column{}
	for _, t := range typs {
		len, _ := t.Length()
		sc := spi.Column{
			Name:   t.Name(),
			Type:   t.ScanType().Name(),
			Size:   int(t.ScanType().Size()),
			Length: int(len),
		}
		result = append(result, &sc)
	}
	return result, nil
}

func (rw *RowsWrap) Next() bool {
	if rw.r == nil {
		return false
	}
	return rw.r.Next()
}

func (rw *RowsWrap) Scan(cols ...any) error {
	return rw.r.Scan(cols...)
}

func (rw *RowsWrap) Close() error {
	if rw.r == nil {
		return nil
	}
	return rw.r.Close()
}

func (rw *RowsWrap) IsFetchable() bool {
	return rw.r.NextResultSet()
}

func (rw *RowsWrap) RowsAffected() int64 {
	return 0
}
func (rw *RowsWrap) Message() string {
	if rw.r == nil {
		return ""
	}
	e := rw.r.Err()
	if e != nil {
		return rw.r.Err().Error()
	}
	return ""
}
