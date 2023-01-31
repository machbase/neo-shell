package sink_exec

import (
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"strings"
	"sync"

	"github.com/machbase/neo-shell/api"
	"github.com/machbase/neo-shell/internal/util"
)

type sink struct {
	bin  string
	args []string

	cmd    *exec.Cmd
	stdin  io.WriteCloser
	stdout io.ReadCloser
	mutex  sync.Mutex
}

func New(cmdLine string) (api.Sink, error) {
	fields := util.SplitFields(cmdLine, true)
	if len(fields) == 0 {
		return nil, errors.New("empty command line")
	}
	sink := &sink{bin: fields[0]}

	if len(fields) > 1 {
		sink.args = fields[1:]
	}

	if err := sink.reset(); err != nil {
		return nil, err
	}
	return sink, nil
}

func (s *sink) Write(p []byte) (n int, err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.cmd == nil {
		return 0, io.EOF
	}

	return s.stdin.Write(p)
}

func (s *sink) Flush() error {
	return nil
}

func (s *sink) Close() error {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	if s.stdin != nil {
		if err := s.stdin.Close(); err != nil {
			return err
		}
		s.stdin = nil
	}

	if s.cmd != nil {
		if err := s.cmd.Wait(); err != nil {
			return err
		}
		code := s.cmd.ProcessState.ExitCode()
		if code != 0 {
			return fmt.Errorf("'%s %s' exit %d", s.bin, strings.Join(s.args, " "), code)
		}
		s.cmd = nil
	}

	return nil
}

func (s *sink) reset() error {
	s.Close()

	s.mutex.Lock()
	defer s.mutex.Unlock()

	var err error

	s.cmd = exec.Command(s.bin, s.args...)
	s.stdin, err = s.cmd.StdinPipe()
	if err != nil {
		fmt.Println("ERR", err.Error())
		return err
	}
	s.stdout, err = s.cmd.StdoutPipe()
	if err != nil {
		fmt.Println("ERR", err.Error())
		return err
	}

	go func() {
		io.Copy(os.Stdout, s.stdout)
	}()

	err = s.cmd.Start()
	if err != nil {
		fmt.Println("ERR start", err.Error())
		return err
	}

	return nil
}
