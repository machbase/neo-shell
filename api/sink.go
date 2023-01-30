package api

type Sink interface {
	Write(p []byte) (n int, err error)
	Flush() error
	Reset() error
	Close() error
}
