package stream

import (
	"io"
	"strings"

	"github.com/machbase/neo-shell/stream/fio"
	"github.com/machbase/neo-shell/stream/pio"
	spi "github.com/machbase/neo-spi"
)

func NewOutputStream(output string) (sink spi.OutputStream, err error) {
	var outputFields = strings.Fields(output)
	if len(outputFields) > 0 && outputFields[0] == "exec" {
		binArgs := strings.TrimSpace(strings.TrimPrefix(output, "exec"))
		sink, err = pio.New(binArgs)
		if err != nil {
			return
		}
	} else {
		sink, err = fio.New(output)
		if err != nil {
			return
		}
	}
	return
}

type WriterOutputStream struct {
	Writer io.Writer
}

func (s *WriterOutputStream) Write(buf []byte) (int, error) {
	return s.Writer.Write(buf)
}

func (s *WriterOutputStream) Flush() error {
	return nil
}

func (s *WriterOutputStream) Close() error {
	if wc, ok := s.Writer.(io.WriteCloser); ok {
		return wc.Close()
	}
	return nil
}
