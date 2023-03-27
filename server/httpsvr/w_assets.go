package httpsvr

import (
	"embed"
	"io/fs"
	"net/http"
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
	path = strings.TrimPrefix(strings.TrimPrefix(path, prefix), "/")
	if len(path) == 0 {
		// this path will be served as '/index.html'
		return true
	}
	f, err := fs.Open(path)
	if err != nil {
		return false
	}
	defer f.Close()

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

func (svr *Server) noroute(docPrefix string, index string, dochandler gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method != http.MethodGet {
			c.Data(http.StatusNotFound, "", nil)
			return
		}

		if c.Request.Method == http.MethodGet &&
			strings.HasPrefix(c.Request.URL.Path, docPrefix) {
			if isWellKnownFileType(c.Request.URL.Path) {
				dochandler(c)
			} else {
				dochandler(c)
			}
			return
		}
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
