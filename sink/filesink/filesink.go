package filesink

import (
	"bufio"
	"io"
	"os"
	"sync"

	"github.com/machbase/neo-grpc/spi"
)

type sink struct {
	path  string
	w     io.WriteCloser
	buf   *bufio.Writer
	mutex sync.Mutex
}

func New(path string) (spi.Sink, error) {
	sink := &sink{
		path: path,
	}
	if err := sink.Reset(); err != nil {
		return nil, err
	}
	return sink, nil
}

func (s *sink) Write(p []byte) (n int, err error) {
	if s.buf == nil {
		return 0, io.EOF
	}
	return s.buf.Write(p)
}

func (s *sink) Flush() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.buf == nil {
		return nil
	}
	return s.buf.Flush()
}

// Deprecated do not call from outside.
func (s *sink) Reset() error {
	s.Close()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.path == "-" {
		s.w = os.Stdout
	} else {
		var err error
		s.w, err = os.OpenFile(s.path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0644)
		if err != nil {
			return err
		}
	}
	s.buf = bufio.NewWriter(s.w)
	return nil
}

func (s *sink) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.buf != nil {
		if err := s.buf.Flush(); err != nil {
			return err
		}
		s.buf = nil
	}
	if s.w != nil && s.path != "-" {
		if err := s.w.Close(); err != nil {
			return err
		}
		s.w = nil
	}
	return nil
}
