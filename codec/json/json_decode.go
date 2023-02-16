package json

import (
	gojson "encoding/json"
	"fmt"
	"io"
	"strconv"
	"time"

	spi "github.com/machbase/neo-spi"
	"github.com/pkg/errors"
)

type Decoder struct {
	columnTypes []string
	reader      *gojson.Decoder
	dataDepth   int
	nrow        int64
	ctx         *spi.RowsDecoderContext
}

func NewDecoder(ctx *spi.RowsDecoderContext) spi.RowsDecoder {
	dec := &Decoder{ctx: ctx}
	dec.columnTypes = ctx.Columns.Types()
	return dec
}

func (dec *Decoder) NextRow() ([]any, error) {
	fields, err := dec.nextRow0()
	if err != nil {
		return nil, err
	}

	dec.nrow++

	if len(fields) != len(dec.columnTypes) {
		return nil, fmt.Errorf("#[%d] number of columns not matched (%d); table '%s' has %d columns",
			dec.nrow, len(fields), dec.ctx.TableName, len(dec.columnTypes))
	}

	values := make([]any, len(dec.columnTypes))
	for i, field := range fields {
		switch dec.columnTypes[i] {
		case "string":
			switch v := field.(type) {
			case string:
				values[i] = v
			default:
				return nil, fmt.Errorf("#[%d] column[%d] is not a string", dec.nrow, i)
			}
		case "datetime":
			var strexp string
			switch v := field.(type) {
			case float64: // json has only float type, no int
				strexp = strconv.FormatInt(int64(v), 10)
			case string:
				strexp = v
			default:
				return nil, fmt.Errorf("#[%d] column[%d] is not datetime convertable", dec.nrow, i)
			}
			var ts int64
			if ts, err = strconv.ParseInt(strexp, 10, 64); err != nil {
				return nil, errors.Wrapf(err, "#[%d] column[%d] is not datetime convertable", dec.nrow, i)
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
			switch v := field.(type) {
			case float64:
				values[i] = v
			default:
				return nil, fmt.Errorf("#[%d] column[%d] is not compatible with double", dec.nrow, i)
			}
		case "int":
			switch v := field.(type) {
			case float64:
				values[i] = int(v)
			default:
				return nil, fmt.Errorf("#[%d] column[%d] is not compatible with int", dec.nrow, i)
			}
		case "int32":
			switch v := field.(type) {
			case float64:
				values[i] = int32(v)
			default:
				return nil, fmt.Errorf("#[%d] column[%d] is not compatible with int32", dec.nrow, i)
			}
		case "int64":
			switch v := field.(type) {
			case float64:
				values[i] = int64(v)
			default:
				return nil, fmt.Errorf("#[%d] column[%d] is not compatible with int64", dec.nrow, i)
			}
		default:
			return nil, fmt.Errorf("unsupported column type; %s", dec.columnTypes[i])
		}
	}
	return values, nil
}

func (dec *Decoder) nextRow0() ([]any, error) {
	if dec.reader == nil {
		dec.reader = gojson.NewDecoder(dec.ctx.Reader)
		// find first '{'
		if tok, err := dec.reader.Token(); err != nil {
			return nil, err
		} else if delim, ok := tok.(gojson.Delim); !ok || delim != '{' {
			return nil, errors.New("invalid top level delimiter")
		}
		// find "data" field
		foundDataField := false
		for {
			if tok, err := dec.reader.Token(); err != nil {
				return nil, err
			} else if key, ok := tok.(string); ok && key == "data" {
				foundDataField = true
				break
			}
		}
		if !foundDataField {
			return nil, errors.New("'data' field not found")
		}
		// find data's array '['
		if tok, err := dec.reader.Token(); err != nil {
			return nil, err
		} else if delim, ok := tok.(gojson.Delim); !ok || delim != '[' {
			return nil, errors.New("'data' field should be an array")
		}
		dec.dataDepth = 1
	}

	if dec.dataDepth == 0 {
		return nil, io.EOF
	}

	tuple := make([]any, 0)
	for dec.reader.More() {
		tok, err := dec.reader.Token()
		if err != nil {
			return nil, err
		}

		if delim, ok := tok.(gojson.Delim); ok {
			if delim == '[' {
				dec.dataDepth++
			} else if delim == '{' {
				return nil, fmt.Errorf("invalid data format at %d", dec.reader.InputOffset())
			}
			tuple = tuple[:0]
			continue
		} else {
			// append element of tuple
			tuple = append(tuple, tok)
		}
	}

	tok, err := dec.reader.Token()
	if err != nil {
		return nil, err
	}
	if delim, ok := tok.(gojson.Delim); ok {
		if delim == ']' {
			dec.dataDepth--
		}
	} else {
		return nil, fmt.Errorf("invalid syntax at %d", dec.reader.InputOffset())
	}

	if len(tuple) == 0 {
		return nil, io.EOF
	}
	return tuple, nil
}
