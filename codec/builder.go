package codec

import (
	"time"

	"github.com/machbase/neo-shell/codec/box"
	"github.com/machbase/neo-shell/codec/csv"
	"github.com/machbase/neo-shell/codec/json"
	spi "github.com/machbase/neo-spi"
)

type EncoderBuilder interface {
	Build(rendererType string) spi.RowsEncoder
	SetSink(s spi.Sink) EncoderBuilder
	SetTimeLocation(tz *time.Location) EncoderBuilder
	SetTimeFormat(f string) EncoderBuilder
	SetPrecision(p int) EncoderBuilder
	SetRownum(flag bool) EncoderBuilder
	SetHeading(flag bool) EncoderBuilder
	// CSV only
	SetCsvDelimieter(del string) EncoderBuilder
	// BOX only
	SetBoxStyle(style string) EncoderBuilder
	SetBoxSeparateColumns(flag bool) EncoderBuilder
	SetBoxDrawBorder(flag bool) EncoderBuilder
}

type builder struct {
	*spi.RowsEncoderContext
	csvDelimiter       string
	boxStyle           string
	boxSeparateColumns bool
	boxDrawBorder      bool
}

func NewEncoderBuilder() EncoderBuilder {
	return &builder{
		RowsEncoderContext: &spi.RowsEncoderContext{},
		csvDelimiter:       ",",
		boxStyle:           "default",
		boxSeparateColumns: true,
		boxDrawBorder:      true,
	}
}

func (b *builder) Build(rendererType string) spi.RowsEncoder {
	switch rendererType {
	case "box":
		return box.NewEncoder(b.RowsEncoderContext, b.boxStyle, b.boxSeparateColumns, b.boxDrawBorder)
	case "csv":
		return csv.NewEncoder(b.RowsEncoderContext, b.csvDelimiter)
	default: // "json"
		return json.NewEncoder(b.RowsEncoderContext)
	}
}

func (b *builder) SetSink(s spi.Sink) EncoderBuilder {
	b.Sink = s
	return b
}

func (b *builder) SetTimeLocation(tz *time.Location) EncoderBuilder {
	b.TimeLocation = tz
	return b
}

func (b *builder) SetTimeFormat(f string) EncoderBuilder {
	b.TimeFormat = spi.GetTimeformat(f)
	return b
}

func (b *builder) SetPrecision(p int) EncoderBuilder {
	b.Precision = p
	return b
}

func (b *builder) SetRownum(flag bool) EncoderBuilder {
	b.Rownum = flag
	return b
}

func (b *builder) SetHeading(flag bool) EncoderBuilder {
	b.Heading = flag
	return b
}

func (b *builder) SetCsvDelimieter(del string) EncoderBuilder {
	b.csvDelimiter = del
	return b
}

func (b *builder) SetBoxStyle(style string) EncoderBuilder {
	b.boxStyle = style
	return b
}
func (b *builder) SetBoxSeparateColumns(flag bool) EncoderBuilder {
	b.boxSeparateColumns = flag
	return b
}

func (b *builder) SetBoxDrawBorder(flag bool) EncoderBuilder {
	b.boxDrawBorder = flag
	return b
}
