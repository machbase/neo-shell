package csv

import (
	"encoding/csv"
	"fmt"
	"io"
	"math"
	"strconv"
	"time"
	"unicode/utf8"

	spi "github.com/machbase/neo-spi"
	"github.com/pkg/errors"
)

type Decoder struct {
	reader      *csv.Reader
	comma       rune
	columnTypes []string
	ctx         *spi.RowsDecoderContext
}

func NewDecoder(ctx *spi.RowsDecoderContext, delimiter string) spi.RowsDecoder {
	rr := &Decoder{ctx: ctx}
	rr.reader = csv.NewReader(ctx.Reader)
	rr.SetDelimiter(delimiter)
	rr.columnTypes = ctx.Columns.Types()
	return rr
}

func (dec *Decoder) SetDelimiter(delimiter string) {
	delmiter, _ := utf8.DecodeRuneInString(delimiter)
	dec.comma = delmiter
}

func (dec *Decoder) NextRow() ([]any, error) {
	if dec.reader == nil {
		return nil, io.EOF
	}

	fields, err := dec.reader.Read()
	if err != nil {
		return nil, err
	}
	if len(fields) > len(dec.columnTypes) {
		return nil, fmt.Errorf("too many columns (%d); table %s has %d columns",
			len(fields), dec.ctx.TableName, len(dec.columnTypes))
	}

	values := make([]any, len(dec.columnTypes))
	for i, field := range fields {
		switch dec.columnTypes[i] {
		case "string":
			values[i] = field
		case "datetime":
			var ts int64
			if ts, err = strconv.ParseInt(field, 10, 64); err != nil {
				return nil, errors.Wrap(err, "unable parse time in timeformat")
			}
			switch dec.ctx.TimeFormat {
			case "s":
				values[i] = time.Unix(ts, 0)
			case "ms":
				values[i] = time.Unix(0, ts*int64(time.Millisecond))
			case "us":
				values[i] = time.Unix(0, ts*int64(time.Microsecond))
			default: // "ns"
				values[i] = time.Unix(0, ts)
			}
		case "double":
			if values[i], err = strconv.ParseFloat(field, 64); err != nil {
				values[i] = math.NaN()
			}
		case "int":
			if values[i], err = strconv.ParseInt(field, 10, 32); err != nil {
				values[i] = math.NaN()
			}
		case "int32":
			if values[i], err = strconv.ParseInt(field, 10, 32); err != nil {
				values[i] = math.NaN()
			}
		case "int64":
			if values[i], err = strconv.ParseInt(field, 10, 64); err != nil {
				values[i] = math.NaN()
			}
		default:
			return nil, fmt.Errorf("unsupported column type; %s", dec.columnTypes[i])
		}
	}
	return values, nil
}
