package static

import (
	"embed"
	"io"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/imgbed/server/router"
)

//go:embed all:embed/admin
var adminFS embed.FS

//go:embed all:embed/site
var siteFS embed.FS

func Setup() *gin.Engine {
	r := router.SetupRouter()

	r.GET("/admin", func(c *gin.Context) {
		serveAdminFile(c, "/index.html")
	})

	r.GET("/admin/*filepath", func(c *gin.Context) {
		serveAdminFile(c, c.Param("filepath"))
	})

	r.GET("/assets/*filepath", func(c *gin.Context) {
		serveSiteFile(c, "/assets"+c.Param("filepath"))
	})

	r.NoRoute(func(c *gin.Context) {
		path := c.Request.URL.Path

		if strings.HasPrefix(path, "/api/") {
			c.String(404, "API not found")
			return
		}

		if path == "/favicon.svg" {
			serveFile(c, siteFS, "embed/site/", "/favicon.svg")
			return
		}

		if path == "/" {
			serveFile(c, siteFS, "embed/site/", "/index.html")
			return
		}

		serveFile(c, siteFS, "embed/site/", path)
	})

	return r
}

func serveAdminFile(c *gin.Context, filePath string) {
	if len(filePath) > 0 && filePath[0] == '/' {
		filePath = filePath[1:]
	}
	if filePath == "" {
		filePath = "index.html"
	}

	fullPath := "embed/admin/" + filePath

	f, err := adminFS.Open(fullPath)
	if err != nil {
		f, err = adminFS.Open("embed/admin/index.html")
		if err != nil {
			c.String(404, "not found")
			return
		}
		filePath = "index.html"
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		c.String(500, "failed to read file")
		return
	}
	c.Data(200, getContentType(filePath), data)
}

func serveSiteFile(c *gin.Context, filePath string) {
	if len(filePath) > 0 && filePath[0] == '/' {
		filePath = filePath[1:]
	}
	if filePath == "" {
		filePath = "index.html"
	}

	fullPath := "embed/site/" + filePath

	f, err := siteFS.Open(fullPath)
	if err != nil {
		f, err = siteFS.Open("embed/site/index.html")
		if err != nil {
			c.String(404, "not found")
			return
		}
		filePath = "index.html"
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		c.String(500, "failed to read file")
		return
	}
	c.Data(200, getContentType(filePath), data)
}

func serveFile(c *gin.Context, fsys embed.FS, prefix, filePath string) {
	if len(filePath) > 0 && filePath[0] == '/' {
		filePath = filePath[1:]
	}
	if filePath == "" {
		filePath = "index.html"
	}

	fullPath := prefix + filePath

	f, err := fsys.Open(fullPath)
	if err != nil {
		indexPath := prefix + "index.html"
		f, err = fsys.Open(indexPath)
		if err != nil {
			c.String(404, "not found")
			return
		}
		filePath = "index.html"
		fullPath = indexPath
	}
	defer f.Close()

	data, err := io.ReadAll(f)
	if err != nil {
		c.String(500, "failed to read file")
		return
	}
	c.Data(200, getContentType(fullPath), data)
}

func getContentType(path string) string {
	ext := path[strings.LastIndex(path, "."):]
	switch ext {
	case ".html":
		return "text/html; charset=utf-8"
	case ".css":
		return "text/css; charset=utf-8"
	case ".js":
		return "application/javascript; charset=utf-8"
	case ".json":
		return "application/json; charset=utf-8"
	case ".png":
		return "image/png"
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".gif":
		return "image/gif"
	case ".svg":
		return "image/svg+xml"
	case ".ico":
		return "image/x-icon"
	case ".woff":
		return "font/woff"
	case ".woff2":
		return "font/woff2"
	case ".ttf":
		return "font/ttf"
	case ".eot":
		return "application/vnd.ms-fontobject"
	default:
		return "application/octet-stream"
	}
}
