package websvr

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"
	"net/url"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed dist/*
var distFs embed.FS

func GetAssets(dir string) fs.FS {
	dir = strings.TrimPrefix(strings.TrimSuffix(dir, "/"), "/")
	subfs, err := fs.Sub(distFs, "dist/"+dir)
	if err != nil {
		panic(err)
	}
	return subfs
}

type docFS struct {
	http.FileSystem
}

func (fs *docFS) Exists(prefix string, path string) bool {
	// fmt.Println("=>", path, ",", prefix)
	if !strings.HasPrefix(path, prefix) {
		return false
	}
	path = strings.TrimPrefix(path, prefix)
	if len(path) == 0 {
		// this path will be served as '/index.html'
		return true
	}
	f, err := fs.Open(path)
	if err != nil {
		return false
	}
	s, err := f.Stat()
	if err != nil {
		return false
	}
	if s.IsDir() {
		// it should return false,
		// so that client can find / -> /index.html
		return false
	}
	return true
}

func (svr *Server) noroute(dochandler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet {
			c.Data(http.StatusNotFound, "", nil)
			return
		}

		if c.Request.Method == http.MethodGet &&
			strings.HasPrefix(c.Request.URL.Path, svr.conf.Prefix) {
			if isWellKnownFileType(c.Request.URL.Path) {
				fmt.Println("->", c.Request.URL.Path)
				c.Request.URL.Path = svr.conf.Prefix
				dochandler(c)
			} else {
				c.Request.URL.Path, _ = url.JoinPath(svr.Base(), "index.html")
				dochandler(c)
			}
			return
		} else if strings.HasPrefix(c.Request.URL.Path, "/api/") {
			return
		}
		fmt.Printf("noroute %v %v\n", c.Request.Method, c.Request.URL)
		svr.log.Warnf("NOROUTE %v %v", c.Request.Method, c.Request.URL)
	}
}

var wellknowns = map[string]bool{
	".css":   true,
	".gif":   true,
	".html":  true,
	".htm":   true,
	".ico":   true,
	".jpg":   true,
	".jpeg":  true,
	".json":  true,
	".js":    true,
	".map":   true,
	".png":   true,
	".svg":   true,
	".ttf":   true,
	".txt":   true,
	".yaml":  true,
	".yml":   true,
	".woff2": true,
}

func isWellKnownFileType(name string) bool {
	ext := filepath.Ext(name)
	if len(ext) == 0 {
		return false
	}

	if _, ok := wellknowns[strings.ToLower(ext)]; ok {
		return true
	}
	return false
}
