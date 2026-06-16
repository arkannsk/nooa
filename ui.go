package nooa

import (
	"embed"
	"io"
	"log"
	"net/http"
	"slices"
	"strconv"
	"strings"
)

//go:embed static/swagger/*
var swaggerFS embed.FS

//go:embed static/redoc/*
var redocFS embed.FS

//go:embed static/scalar/*
var scalarFS embed.FS

func SwaggerUIHandler(basePrefix string, specURL string) http.Handler {
	return embeddedUIHandler(swaggerFS, "static/swagger/", basePrefix, specURL, "index.html", "swagger-initializer.js")
}

func RedocUIHandler(basePrefix string, specURL string) http.Handler {
	return embeddedUIHandler(redocFS, "static/redoc/", basePrefix, specURL, "index.html")
}

func ScalarUIHandler(basePrefix string, specURL string) http.Handler {
	return embeddedUIHandler(scalarFS, "static/scalar/", basePrefix, specURL, "index.html")
}

// embeddedUIHandler creates an HTTP handler that serves static files from an embedded FS,
// replacing {{SPEC_URL}} in the specified modified files.
func embeddedUIHandler(fs embed.FS, embedPath, basePrefix, specURL string, modifiedFiles ...string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		cleanPath := strings.TrimSuffix(path, "/")

		// Redirect to index.html when accessing the prefix root
		if cleanPath == basePrefix || path == basePrefix+"/" {
			http.Redirect(w, r, path+"index.html", http.StatusFound)
			return
		}

		if !strings.HasPrefix(path, basePrefix+"/") && path != basePrefix {
			http.NotFound(w, r)
			return
		}

		filePath := strings.TrimPrefix(path, basePrefix+"/")
		if filePath == "" {
			filePath = "index.html"
		}

		f, err := fs.Open(embedPath + filePath)
		if err != nil {
			log.Printf("ERROR: File not found in embed: %s%s. Err: %v", embedPath, filePath, err)
			http.NotFound(w, r)
			return
		}
		defer f.Close()

		stat, err := f.Stat()
		isModifiedFile := isListed(filePath, modifiedFiles)

		if stat != nil && !isModifiedFile {
			w.Header().Set("Content-Length", strconv.FormatInt(stat.Size(), 10))
			if !stat.ModTime().IsZero() {
				w.Header().Set("Last-Modified", stat.ModTime().UTC().Format(http.TimeFormat))
			}
		}

		setContentType(w, filePath)

		if isModifiedFile {
			data, err := io.ReadAll(f)
			if err != nil {
				http.Error(w, "Internal Server Error", http.StatusInternalServerError)
				return
			}

			content := strings.Replace(string(data), "{{SPEC_URL}}", specURL, -1)

			_, err = w.Write([]byte(content))
			if err != nil {
				log.Printf("Error writing response for %s: %v", filePath, err)
			}
			return
		}

		_, err = io.Copy(w, f)
		if err != nil {
			log.Printf("Error copying %s to response: %v", filePath, err)
		}
	})
}

func isListed(s string, list []string) bool {
	return slices.Contains(list, s)
}

func setContentType(w http.ResponseWriter, path string) {
	switch {
	case strings.HasSuffix(path, ".html"):
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	case strings.HasSuffix(path, ".js"):
		w.Header().Set("Content-Type", "application/javascript; charset=utf-8")
	case strings.HasSuffix(path, ".css"):
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	case strings.HasSuffix(path, ".png"):
		w.Header().Set("Content-Type", "image/png")
	case strings.HasSuffix(path, ".svg"):
		w.Header().Set("Content-Type", "image/svg+xml")
	case strings.HasSuffix(path, ".json"):
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
	default:
		w.Header().Set("Content-Type", "application/octet-stream")
	}
}
