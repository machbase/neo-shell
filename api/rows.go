package api

import (
	"time"
)

type RowsContext struct {
	Sink         Sink
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
