package websvr

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gin-contrib/static"
	"github.com/gin-gonic/gin"
	logging "github.com/machbase/neo-logging"
	spi "github.com/machbase/neo-spi"
)

func New(db spi.Database, conf *Config) (*Server, error) {
	return &Server{
		conf:    conf,
		log:     logging.GetLog("websvr"),
		db:      db,
		rtTable: make(map[string]*JwtCacheValue),
	}, nil
}

type Config struct {
	Prefix string
}

type Server struct {
	conf    *Config
	log     logging.Log
	db      spi.Database
	rtTable map[string]*JwtCacheValue
	rtLock  sync.RWMutex
}

func (svr *Server) Start() error {
	return nil
}

func (svr *Server) Stop() {

}

func (svr *Server) Base() string {
	return svr.conf.Prefix
}

func (svr *Server) Route(r *gin.Engine) {
	contentBase := "/ui/"
	rootPrefix := strings.TrimSuffix(svr.conf.Prefix, "/")

	svr.conf.Prefix, _ = url.JoinPath("/", rootPrefix, contentBase)

	prefix := svr.conf.Prefix
	svr.log.Debugf("Add handler %s webui", prefix)

	dochandler := static.Serve(svr.conf.Prefix, &docFS{
		FileSystem: http.FS(GetAssets(contentBase)),
	})
	r.Use(dochandler)
	r.GET(rootPrefix+"/", func(ctx *gin.Context) {
		ctx.Redirect(http.StatusFound, prefix)
	})
	r.POST(rootPrefix+"/api/login", svr.handleLogin)
	r.POST(rootPrefix+"/api/relogin", svr.handleReLogin)
	r.POST(rootPrefix+"/api/logout", svr.handleLogout)
	r.Any("/machbase", func(ctx *gin.Context) {
		fmt.Println(ctx.Request.Method, ctx.Request.URL.Path)
	})
	r.NoRoute(svr.noroute(dochandler))
}

type JwtCacheValue struct {
	Rt   string
	When time.Time
}

func (svr *Server) SetRefreshToken(id string, rt string) {
	svr.rtLock.Lock()
	defer svr.rtLock.Unlock()
	svr.rtTable[id] = &JwtCacheValue{
		Rt:   rt,
		When: time.Now(),
	}
}

func (svr *Server) GetRefreshToken(id string) (string, bool) {
	svr.rtLock.RLock()
	defer svr.rtLock.RUnlock()
	val, ok := svr.rtTable[id]
	if val != nil {
		return val.Rt, ok
	} else {
		return "", ok
	}
}

func (svr *Server) RemoveRefreshToken(id string) {
	svr.rtLock.Lock()
	defer svr.rtLock.Unlock()
	delete(svr.rtTable, id)
}
