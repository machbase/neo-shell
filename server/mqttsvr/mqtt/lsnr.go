package mqtt

import (
	"net"
	"time"

	"github.com/machbase/neo-shell/server/allowance"
)

type Listener interface {
	Name() string
	Start() error
	Stop() error
	IsAlive() bool
	SetAllowance(allowance allowance.Allowance)
}

func NewTcpListener(cfg *TcpListenerConfig, acceptChan chan<- any) (Listener, error) {
	return newTcpListener(cfg, acceptChan)
}

func NewUnixSocketListener(cfg *UnixSocketListenerConfig, acceptChan chan<- any) (Listener, error) {
	return newUnixSocketListener(cfg, acceptChan)
}

func (c *TcpListenerConfig) configure(tcpConn *net.TCPConn) {
	if c == nil {
		return
	}

	tcpConn.SetLinger(c.SoLinger)
	if c.KeepAlive > 0 {
		tcpConn.SetKeepAlive(true)
		tcpConn.SetKeepAlivePeriod(time.Duration(c.KeepAlive))
	} else {
		tcpConn.SetKeepAlive(false)
	}
	tcpConn.SetNoDelay(c.NoDelay)
}
