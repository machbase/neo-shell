package api

import (
	"context"
	"io"
)

type SeriesData struct {
	Name   string
	Values []float64
	Labels []string
}

type SeriesRenderer interface {
	Render(ctx context.Context, writer io.Writer, data []*SeriesData) error
}
