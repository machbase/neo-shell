package mqtt

import (
	"time"

	"github.com/machbase/neo-shell/server/allowance"
)

type MqttConfig struct {
	Name             string
	TcpListeners     []TcpListenerConfig
	UnixSocketConfig UnixSocketListenerConfig
	Allowance        allowance.AllowanceConfig
	HealthCheckAddrs []string

	MaxMessageSizeLimit int
}

type TcpListenerConfig struct {
	ListenAddress string
	SoLinger      int
	KeepAlive     int
	NoDelay       bool
	Tls           TlsListenerConfig
}

type TlsListenerConfig struct {
	Disabled         bool
	LoadSystemCAs    bool          // LoadSystemCAs: 시스템에서 CA pool을 읽어 초기화 여부, false일 경우 empty CA pool을 생성
	LoadPrivateCAs   bool          // 서버의 인증서를 CA pool에 추가할지 여부, true일 경우 CertFile, KeyFile에 지정된 인증서를 CA pool에 추가
	CertFile         string        // 인증서 PEM 파일의 경로
	KeyFile          string        // 서버 private key의 PEM 파일 경로
	HandshakeTimeout time.Duration // client에서 연결만 한 상태로 아무런 메지 없을 경우 timeout에 의해 연결을 강제로 종료한다.
}

type UnixSocketListenerConfig struct {
	Path       string
	Permission int
}
