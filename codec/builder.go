package codec

import (
	"io"
	"time"

	"github.com/machbase/neo-shell/codec/box"
	"github.com/machbase/neo-shell/codec/csv"
	"github.com/machbase/neo-shell/codec/json"
	spi "github.com/machbase/neo-spi"
)

type EncoderBuilder interface {
	Build() spi.RowsEncoder
	SetOutputStream(s spi.OutputStream) EncoderBuilder
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

type encBuilder struct {
	*spi.RowsEncoderContext
	encoderType        string
	csvDelimiter       string
	boxStyle           string
	boxSeparateColumns bool
	boxDrawBorder      bool
}

func NewEncoderBuilder(encoderType string) EncoderBuilder {
	return &encBuilder{
		RowsEncoderContext: &spi.RowsEncoderContext{},
		encoderType:        encoderType,
		csvDelimiter:       ",",
		boxStyle:           "default",
		boxSeparateColumns: true,
		boxDrawBorder:      true,
	}
}

func (b *encBuilder) Build() spi.RowsEncoder {
	switch b.encoderType {
	case "box":
		return box.NewEncoder(b.RowsEncoderContext, b.boxStyle, b.boxSeparateColumns, b.boxDrawBorder)
	case "csv":
		return csv.NewEncoder(b.RowsEncoderContext, b.csvDelimiter)
	default: // "json"
		return json.NewEncoder(b.RowsEncoderContext)
	}
}

func (b *encBuilder) SetOutputStream(s spi.OutputStream) EncoderBuilder {
	b.Output = s
	return b
}

func (b *encBuilder) SetTimeLocation(tz *time.Location) EncoderBuilder {
	b.TimeLocation = tz
	return b
}

func (b *encBuilder) SetTimeFormat(f string) EncoderBuilder {
	b.TimeFormat = spi.GetTimeformat(f)
	return b
}

func (b *encBuilder) SetPrecision(p int) EncoderBuilder {
	b.Precision = p
	return b
}

func (b *encBuilder) SetRownum(flag bool) EncoderBuilder {
	b.Rownum = flag
	return b
}

func (b *encBuilder) SetHeading(flag bool) EncoderBuilder {
	b.Heading = flag
	return b
}

func (b *encBuilder) SetCsvDelimieter(del string) EncoderBuilder {
	b.csvDelimiter = del
	return b
}

func (b *encBuilder) SetBoxStyle(style string) EncoderBuilder {
	b.boxStyle = style
	return b
}
func (b *encBuilder) SetBoxSeparateColumns(flag bool) EncoderBuilder {
	b.boxSeparateColumns = flag
	return b
}

func (b *encBuilder) SetBoxDrawBorder(flag bool) EncoderBuilder {
	b.boxDrawBorder = flag
	return b
}

type DecoderBuilder interface {
	Build(decoderType string) spi.RowsDecoder
	SetReader(reader io.Reader) DecoderBuilder
	SetColumns(columns spi.Columns) DecoderBuilder
	SetCsvDelimieter(del string) DecoderBuilder
}

type decBuilder struct {
	*spi.RowsDecoderContext
	csvDelimiter string
}

func NewDecoderBuilder() DecoderBuilder {
	return &decBuilder{
		RowsDecoderContext: &spi.RowsDecoderContext{},
		csvDelimiter:       ",",
	}
}

func (b *decBuilder) Build(decoderType string) spi.RowsDecoder {
	switch decoderType {
	case "csv":
		return csv.NewDecoder(b.RowsDecoderContext, b.csvDelimiter)
	default: // "json"
		return nil
	}
}

func (b *decBuilder) SetReader(reader io.Reader) DecoderBuilder {
	b.Reader = reader
	return b
}

func (b *decBuilder) SetColumns(columns spi.Columns) DecoderBuilder {
	b.RowsDecoderContext.Columns = columns
	return b
}

func (b *decBuilder) SetCsvDelimieter(del string) DecoderBuilder {
	b.csvDelimiter = del
	return b
}
