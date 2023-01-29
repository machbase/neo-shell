package api

import (
	"bufio"
	"time"
)

type RowsContext struct {
	Writer       *bufio.Writer
	Rownum       bool
	Heading      bool
	TimeLocation *time.Location
	TimeFormat   string
	Precision    int
	HeaderHeight int
	ColumnNames  []string
	ColumnTypes  []string
}

type RowsRenderer interface {
	OpenRender(ctx *RowsContext) error
	CloseRender()
	RenderRow(values []any) error
	PageFlush(heading bool)
}
