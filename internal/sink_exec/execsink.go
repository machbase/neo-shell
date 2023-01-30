package sink_exec

import "github.com/machbase/neo-shell/api"

type sink struct {
}

func New() api.Sink {
	sink := &sink{}

	return sink
}

func (s *sink) Write(p []byte) (n int, err error) {
	return 0, nil
}

func (s *sink) Flush() error {
	return nil
}

func (s *sink) Reset() error {
	return nil
}

func (s *sink) Close() error {
	return nil
}
