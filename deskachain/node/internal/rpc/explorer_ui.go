package rpc

import (
	"embed"
	"net/http"
	"path"
	"strings"
)

//go:embed web/explorer/index.html web/explorer/assets/app.js web/explorer/assets/styles.css
var explorerUIFiles embed.FS

func (h handler) explorerUI(w http.ResponseWriter, r *http.Request) {
	name := "index.html"
	if r.URL.Path != "/explorer-ui" && r.URL.Path != "/explorer-ui/" {
		assetPath := strings.TrimPrefix(r.URL.Path, "/explorer-ui/")
		assetPath = path.Clean("/" + assetPath)
		assetPath = strings.TrimPrefix(assetPath, "/")
		if !strings.HasPrefix(assetPath, "assets/") || strings.HasSuffix(assetPath, "/") {
			http.NotFound(w, r)
			return
		}
		name = assetPath
	}

	data, err := explorerUIFiles.ReadFile("web/explorer/" + name)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	switch {
	case strings.HasSuffix(name, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(name, ".js"):
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case strings.HasSuffix(name, ".css"):
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
	w.Header().Set("X-Content-Type-Options", "nosniff")
	w.WriteHeader(http.StatusOK)
	_, _ = w.Write(data)
}
