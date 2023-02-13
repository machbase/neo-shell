package sink

import (
	"io"
	"strings"

	"github.com/machbase/neo-shell/sink/execsink"
	"github.com/machbase/neo-shell/sink/filesink"
	spi "github.com/machbase/neo-spi"
)

func MakeSink(output string) (sink spi.Sink, err error) {
	var outputFields = strings.Fields(output)
	if len(outputFields) > 0 && outputFields[0] == "exec" {
		binArgs := strings.TrimSpace(strings.TrimPrefix(output, "exec"))
		sink, err = execsink.New(binArgs)
		if err != nil {
			return
		}
	} else {
		sink, err = filesink.New(output)
		if err != nil {
			return
		}
	}
	return
}

type WriterSink struct {
	Writer io.Writer
}

func (s *WriterSink) Write(buf []byte) (int, error) {
	return s.Writer.Write(buf)
}

func (s *WriterSink) Flush() error {
	return nil
}

func (s *WriterSink) Close() error {
	if wc, ok := s.Writer.(io.WriteCloser); ok {
		return wc.Close()
	}
	return nil
}
