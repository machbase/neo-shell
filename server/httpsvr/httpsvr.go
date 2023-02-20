package httpsvr

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	logging "github.com/machbase/neo-logging"
	spi "github.com/machbase/neo-spi"
)

type AuthServer interface {
	ValidateClientToken(token string) (bool, error)
}

func New(db spi.Database, conf *Config) (*Server, error) {
	return &Server{
		conf: conf,
		log:  logging.GetLog("httpsvr"),
		db:   db,
	}, nil
}

type Config struct {
	Handlers []HandlerConfig
}

type HandlerConfig struct {
	Prefix  string
	Handler string
}

type Server struct {
	conf *Config
	log  logging.Log
	db   spi.Database

	authServer AuthServer // injection point
}

func (svr *Server) Start() error {
	return nil
}

func (svr *Server) Stop() {
}

func (svr *Server) SetAuthServer(authServer AuthServer) {
	svr.authServer = authServer
}

func (svr *Server) Route(r *gin.Engine) {
	if svr.authServer != nil {
		r.Use(svr.handleAuthToken)
	}
	for _, h := range svr.conf.Handlers {
		prefix := h.Prefix
		// remove trailing slash
		for strings.HasSuffix(prefix, "/") {
			prefix = prefix[0 : len(prefix)-1]
		}

		svr.log.Debugf("Add handler %s '%s'", h.Handler, prefix)

		switch h.Handler {
		case "influx": // "influx line protocol"
			r.POST(prefix+"/:oper", svr.handleLineProtocol)
		default: // "machbase"
			r.GET(prefix+"/query", svr.handleQuery)
			r.POST(prefix+"/query", svr.handleQuery)
			r.GET(prefix+"/chart", svr.handleChart)
			r.POST(prefix+"/chart", svr.handleChart)
			r.POST(prefix+"/write", svr.handleWrite)
			r.POST(prefix+"/write/:table", svr.handleWrite)
		}
	}
}

func (svr *Server) handleAuthToken(ctx *gin.Context) {
	if svr.authServer == nil {
		ctx.JSON(http.StatusUnauthorized, map[string]any{"success": false, "reason": "no auth server"})
		ctx.Abort()
	}
	auth, exist := ctx.Request.Header["Authorization"]
	if !exist {
		ctx.JSON(http.StatusUnauthorized, map[string]any{"success": false, "reason": "missing authorization header"})
		ctx.Abort()
	}
	found := false
	for _, h := range auth {
		if !strings.HasPrefix(strings.ToUpper(h), "BEARER ") {
			continue
		}
		tok := h[7:]
		result, err := svr.authServer.ValidateClientToken(tok)
		if err != nil {
			svr.log.Errorf("server private key", err)
		}
		if result {
			found = true
			break
		}
	}
	if !found {
		ctx.JSON(http.StatusUnauthorized, map[string]any{"success": false, "reason": "missing valid token"})
		ctx.Abort()
	}
}
