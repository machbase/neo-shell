package api

import (
	"context"
)

type SeriesData struct {
	Name   string
	Values []float64
	Labels []string
}

type SeriesRenderer interface {
	Render(ctx context.Context, sink Sink, data []*SeriesData) error
}
